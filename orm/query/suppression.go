package query

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// SuppressionScope describes where a suppression applies.
type SuppressionScope string

const (
	SuppressionScopeQuery  SuppressionScope = "query"
	SuppressionScopeInline SuppressionScope = "inline"
	SuppressionScopeConfig SuppressionScope = "config"
)

// Suppression suppresses an expected warning while keeping accountability data.
type Suppression struct {
	Code      string           `json:"code"`
	Reason    string           `json:"reason"`
	Scope     SuppressionScope `json:"scope"`
	Location  *SourceLocation  `json:"location,omitempty"`
	ExpiresAt *time.Time       `json:"expires_at,omitempty"`
	Owner     string           `json:"owner,omitempty"`
}

// SuppressionOption configures a runtime suppression.
type SuppressionOption func(*Suppression)

// SuppressionExpiresAt sets the expiration timestamp for a suppression.
func SuppressionExpiresAt(t time.Time) SuppressionOption {
	return func(s *Suppression) {
		s.ExpiresAt = &t
	}
}

// SuppressionOwner sets the suppression owner.
func SuppressionOwner(owner string) SuppressionOption {
	return func(s *Suppression) {
		s.Owner = strings.TrimSpace(owner)
	}
}

// NewSuppression creates a query-scoped suppression.
func NewSuppression(code, reason string, opts ...SuppressionOption) (Suppression, error) {
	s := Suppression{
		Code:   strings.TrimSpace(code),
		Reason: strings.TrimSpace(reason),
		Scope:  SuppressionScopeQuery,
	}
	if s.Code == "" {
		return Suppression{}, fmt.Errorf("goquent: suppression code is required")
	}
	if s.Reason == "" {
		return Suppression{}, fmt.Errorf("goquent: suppression reason is required")
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&s)
		}
	}
	return s, nil
}

// ParseInlineSuppression parses comments like:
// goquent:suppress LIMIT_MISSING reason="batch export" expires="2026-07-01"
func ParseInlineSuppression(comment string) (Suppression, bool, error) {
	const marker = "goquent:suppress"
	idx := strings.Index(comment, marker)
	if idx < 0 {
		return Suppression{}, false, nil
	}
	tokens, err := splitSuppressionTokens(comment[idx+len(marker):])
	if err != nil {
		return Suppression{}, true, err
	}
	if len(tokens) == 0 {
		return Suppression{}, true, fmt.Errorf("goquent: suppression code is required")
	}

	s := Suppression{
		Code:  strings.TrimSpace(tokens[0]),
		Scope: SuppressionScopeInline,
	}
	for _, token := range tokens[1:] {
		key, value, ok := strings.Cut(token, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if unquoted, err := strconv.Unquote(value); err == nil {
			value = unquoted
		}
		switch key {
		case "reason":
			s.Reason = strings.TrimSpace(value)
		case "expires":
			expiresAt, err := parseSuppressionTime(value)
			if err != nil {
				return Suppression{}, true, err
			}
			s.ExpiresAt = &expiresAt
		case "owner":
			s.Owner = strings.TrimSpace(value)
		}
	}
	if s.Code == "" {
		return Suppression{}, true, fmt.Errorf("goquent: suppression code is required")
	}
	if s.Reason == "" {
		return Suppression{}, true, fmt.Errorf("goquent: suppression reason is required")
	}
	return s, true, nil
}

func splitSuppressionTokens(s string) ([]string, error) {
	var tokens []string
	for len(strings.TrimSpace(s)) > 0 {
		s = strings.TrimSpace(s)
		var b strings.Builder
		inQuote := false
		escaped := false
		i := 0
		for ; i < len(s); i++ {
			ch := s[i]
			if escaped {
				b.WriteByte(ch)
				escaped = false
				continue
			}
			if ch == '\\' && inQuote {
				b.WriteByte(ch)
				escaped = true
				continue
			}
			if ch == '"' {
				inQuote = !inQuote
				b.WriteByte(ch)
				continue
			}
			if !inQuote && (ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r') {
				break
			}
			b.WriteByte(ch)
		}
		if inQuote {
			return nil, fmt.Errorf("goquent: unterminated suppression quote")
		}
		tokens = append(tokens, b.String())
		if i >= len(s) {
			break
		}
		s = s[i+1:]
	}
	return tokens, nil
}

func parseSuppressionTime(value string) (time.Time, error) {
	if t, err := time.Parse("2006-01-02", value); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("goquent: invalid suppression expires %q", value)
}

func applySuppressions(warnings []Warning, suppressions []Suppression, now time.Time) ([]Warning, []Warning, []Warning) {
	if len(warnings) == 0 || len(suppressions) == 0 {
		return warnings, nil, nil
	}

	var kept []Warning
	var suppressed []Warning
	var suppressionWarnings []Warning
	for _, warning := range warnings {
		suppression, ok := findSuppression(warning.Code, suppressions)
		if !ok {
			kept = append(kept, warning)
			continue
		}
		if suppression.Reason == "" {
			kept = append(kept, warning)
			suppressionWarnings = append(suppressionWarnings, newWarning(WarningSuppressionNotAllowed, RiskMedium,
				"suppression reason is required",
				"add a reason to the suppression",
				false,
				false,
			))
			continue
		}
		if suppression.ExpiresAt != nil && !suppression.ExpiresAt.After(now) {
			kept = append(kept, warning)
			suppressionWarnings = append(suppressionWarnings, newWarning(WarningSuppressionExpired, RiskMedium,
				"suppression has expired",
				"remove the suppression or renew it with a current reason",
				false,
				false,
			))
			continue
		}
		if !warning.Suppressible {
			kept = append(kept, warning)
			suppressionWarnings = append(suppressionWarnings, newWarning(WarningSuppressionNotAllowed, RiskMedium,
				"warning is not suppressible",
				"remove the suppression and fix or explicitly approve the operation when allowed",
				false,
				false,
			))
			continue
		}
		suppressed = append(suppressed, warning)
	}
	return kept, suppressed, suppressionWarnings
}

func findSuppression(code string, suppressions []Suppression) (Suppression, bool) {
	for _, suppression := range suppressions {
		if suppression.Code == code {
			return suppression, true
		}
	}
	return Suppression{}, false
}
