package stringutil

import (
	"strings"
	"unicode"
)

// ToSnake converts a CamelCase string to snake_case.
// It inserts underscores at word boundaries and handles acronyms.
// Insert underscore before the last uppercase letter in a sequence when followed by a lowercase letter.
// For example, "HTTPSUrl" becomes "https_url".
func ToSnake(s string) string {
	runes := []rune(s)
	var sb strings.Builder
	for i, r := range runes {
		if i > 0 {
			prev := runes[i-1]
			next := rune(0)
			if i+1 < len(runes) {
				next = runes[i+1]
			}
			if unicode.IsLower(prev) && unicode.IsUpper(r) {
				sb.WriteByte('_')
			} else if unicode.IsUpper(prev) && unicode.IsUpper(r) && next != 0 && unicode.IsLower(next) {
				// Insert underscore before the last uppercase letter in a sequence when followed by a lowercase letter.
				// For example, 'HTTPSUrl' becomes 'https_url'.
				sb.WriteByte('_')
			}
		}
		sb.WriteRune(unicode.ToLower(r))
	}
	return sb.String()
}
