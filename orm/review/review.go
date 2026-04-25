package review

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/faciam-dev/goquent/orm/manifest"
	"github.com/faciam-dev/goquent/orm/migration"
	"github.com/faciam-dev/goquent/orm/query"
)

// Finding is a review finding emitted by goquent review.
type Finding struct {
	Code              string                  `json:"code"`
	Level             query.RiskLevel         `json:"level"`
	Message           string                  `json:"message"`
	Location          *query.SourceLocation   `json:"location,omitempty"`
	Hint              string                  `json:"hint,omitempty"`
	Evidence          []query.Evidence        `json:"evidence,omitempty"`
	AnalysisPrecision query.AnalysisPrecision `json:"analysis_precision"`
	Suppressed        bool                    `json:"suppressed"`
	Suppression       *query.Suppression      `json:"suppression,omitempty"`
}

// ReviewSummary aggregates findings for machine-readable output and CI.
type ReviewSummary struct {
	Total       int                     `json:"total"`
	Suppressed  int                     `json:"suppressed"`
	ByLevel     map[query.RiskLevel]int `json:"by_level,omitempty"`
	HighestRisk query.RiskLevel         `json:"highest_risk"`
}

// ManifestStatus is reserved for Phase 6 stale manifest integration.
type ManifestStatus struct {
	Fresh bool   `json:"fresh"`
	Path  string `json:"path,omitempty"`
}

// ReviewReport is the top-level report produced by goquent review.
type ReviewReport struct {
	Findings           []Finding       `json:"findings"`
	SuppressedFindings []Finding       `json:"suppressed_findings,omitempty"`
	Summary            ReviewSummary   `json:"summary"`
	ManifestStatus     *ManifestStatus `json:"manifest_status,omitempty"`
}

// Options controls review discovery and output behavior.
type Options struct {
	Paths                []string
	ShowSuppressed       bool
	ManifestPath         string
	RequireFreshManifest bool
}

// Run reviews all configured paths.
func Run(opts Options) (ReviewReport, error) {
	paths := opts.Paths
	if len(paths) == 0 {
		paths = []string{"."}
	}

	var report ReviewReport
	var errs []error
	if strings.TrimSpace(opts.ManifestPath) != "" {
		findings, status, err := reviewManifestFreshness(opts.ManifestPath)
		if err != nil {
			errs = append(errs, err)
		}
		if status != nil {
			report.ManifestStatus = status
		}
		report.Findings = append(report.Findings, findings...)
	}
	for _, path := range paths {
		files, err := discoverFiles(path)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, file := range files {
			findings, err := reviewFile(file)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			for _, finding := range findings {
				if finding.Suppressed {
					report.SuppressedFindings = append(report.SuppressedFindings, finding)
					if opts.ShowSuppressed {
						report.Findings = append(report.Findings, finding)
					}
					continue
				}
				report.Findings = append(report.Findings, finding)
			}
		}
	}
	report.Summary = summarize(report)
	return report, errors.Join(errs...)
}

func reviewManifestFreshness(path string) ([]Finding, *ManifestStatus, error) {
	m, err := manifest.Load(path)
	if err != nil {
		return nil, nil, err
	}
	status := &ManifestStatus{Fresh: true, Path: path}
	if m.Verification != nil {
		status.Fresh = m.Verification.Fresh
	}
	if m.Verification == nil || m.Verification.Fresh {
		return nil, status, nil
	}
	var evidence []query.Evidence
	for _, check := range m.Verification.Checks {
		if check.Status == "stale" {
			evidence = append(evidence, query.Evidence{Key: check.Name, Value: check.Message})
		}
	}
	return []Finding{{
		Code:              manifest.WarningStale,
		Level:             query.RiskHigh,
		Message:           "manifest does not match current schema, policy, generated code, or database fingerprint",
		Location:          &query.SourceLocation{File: path, Line: 1},
		Hint:              "regenerate the manifest or run goquent manifest verify against current inputs",
		Evidence:          evidence,
		AnalysisPrecision: query.AnalysisPrecise,
	}}, status, nil
}

func discoverFiles(root string) ([]string, error) {
	root = expandEllipsis(root)
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		if supportedFile(root) {
			return []string{root}, nil
		}
		return nil, nil
	}

	var files []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".codex", ".gocache", "vendor", "node_modules", "dist", "build", "coverage":
				return filepath.SkipDir
			}
			return nil
		}
		if supportedFile(path) {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func expandEllipsis(path string) string {
	path = strings.TrimSpace(path)
	switch path {
	case "", "...", "./...":
		return "."
	}
	if strings.HasSuffix(path, "/...") {
		root := strings.TrimSuffix(path, "/...")
		if root == "" {
			return "."
		}
		return root
	}
	return path
}

func supportedFile(path string) bool {
	switch filepath.Ext(path) {
	case ".go", ".sql", ".json":
		return true
	default:
		return false
	}
}

func reviewFile(path string) ([]Finding, error) {
	switch filepath.Ext(path) {
	case ".go":
		return reviewGoFile(path)
	case ".sql":
		return reviewSQLFile(path)
	case ".json":
		return reviewPlanJSONFile(path)
	default:
		return nil, nil
	}
}

func reviewSQLFile(path string) ([]Finding, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	sqlText := string(b)
	if looksLikeMigrationSQL(sqlText) {
		if plan, err := migration.PlanSQL(sqlText); err != nil {
			return nil, err
		} else if len(plan.Steps) > 0 {
			findings := warningsToFindings(plan.Warnings, plan.AnalysisPrecision, &query.SourceLocation{File: path, Line: 1})
			return applyFileSuppressions(path, findings)
		}
	}
	plan := query.NewRawPlan(sqlText)
	findings := findingsFromPlan(plan, query.AnalysisPrecise, &query.SourceLocation{File: path, Line: 1})
	return applyFileSuppressions(path, findings)
}

func looksLikeMigrationSQL(sqlText string) bool {
	upper := strings.ToUpper(sqlText)
	for _, token := range []string{"CREATE", "ALTER", "DROP", "RENAME", "GRANT", "REVOKE", "TRUNCATE"} {
		if containsSQLWord(upper, token) {
			return true
		}
	}
	return false
}

func reviewPlanJSONFile(path string) ([]Finding, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var plan query.QueryPlan
	if err := json.Unmarshal(b, &plan); err != nil {
		return nil, nil
	}
	if plan.Operation == "" || plan.SQL == "" {
		migrationFindings, ok, err := reviewMigrationPlanJSON(path, b)
		if err != nil || ok {
			return migrationFindings, err
		}
		return nil, nil
	}
	result := query.DefaultRiskEngine.CheckQuery(&plan)
	warnings := plan.Warnings
	if len(warnings) == 0 {
		warnings = result.Warnings
	}
	findings := warningsToFindings(warnings, query.AnalysisPrecise, &query.SourceLocation{File: path, Line: 1})
	for _, finding := range warningsToFindings(plan.SuppressedWarnings, query.AnalysisPrecise, &query.SourceLocation{File: path, Line: 1}) {
		finding.Suppressed = true
		findings = append(findings, finding)
	}
	return applyFileSuppressions(path, findings)
}

func reviewMigrationPlanJSON(path string, b []byte) ([]Finding, bool, error) {
	var plan migration.MigrationPlan
	if err := json.Unmarshal(b, &plan); err != nil {
		return nil, false, nil
	}
	if plan.SQL == "" && len(plan.Steps) == 0 && len(plan.Statements) == 0 {
		return nil, false, nil
	}
	if len(plan.Warnings) == 0 && plan.SQL != "" {
		planned, err := migration.PlanSQL(plan.SQL)
		if err != nil {
			return nil, true, err
		}
		plan.Warnings = planned.Warnings
		plan.AnalysisPrecision = planned.AnalysisPrecision
	}
	findings := warningsToFindings(plan.Warnings, plan.AnalysisPrecision, &query.SourceLocation{File: path, Line: 1})
	findings, err := applyFileSuppressions(path, findings)
	return findings, true, err
}

func findingsFromPlan(plan *query.QueryPlan, precision query.AnalysisPrecision, loc *query.SourceLocation) []Finding {
	if plan == nil {
		return nil
	}
	return warningsToFindings(plan.Warnings, precision, loc)
}

func warningsToFindings(warnings []query.Warning, precision query.AnalysisPrecision, loc *query.SourceLocation) []Finding {
	findings := make([]Finding, 0, len(warnings))
	for _, warning := range warnings {
		findingLoc := loc
		if warning.Location != nil {
			copied := *warning.Location
			if copied.File == "" && loc != nil {
				copied.File = loc.File
			}
			findingLoc = &copied
		}
		findings = append(findings, Finding{
			Code:              warning.Code,
			Level:             warning.Level,
			Message:           warning.Message,
			Location:          cloneLocation(findingLoc),
			Hint:              warning.Hint,
			Evidence:          append([]query.Evidence(nil), warning.Evidence...),
			AnalysisPrecision: precision,
		})
	}
	return findings
}

func cloneLocation(loc *query.SourceLocation) *query.SourceLocation {
	if loc == nil {
		return nil
	}
	copied := *loc
	return &copied
}

func applyFileSuppressions(path string, findings []Finding) ([]Finding, error) {
	if len(findings) == 0 {
		return findings, nil
	}
	suppressions, err := suppressionsForFile(path)
	if err != nil {
		return nil, err
	}
	if len(suppressions) == 0 {
		return findings, nil
	}

	var out []Finding
	now := time.Now().UTC()
	for _, finding := range findings {
		suppression, ok := findSuppressionForFinding(finding, suppressions)
		if !ok {
			out = append(out, finding)
			continue
		}
		if suppression.ExpiresAt != nil && !suppression.ExpiresAt.After(now) {
			out = append(out, finding)
			out = append(out, suppressionFinding(query.WarningSuppressionExpired, "suppression has expired", finding.Location))
			continue
		}
		if !findingSuppressible(finding.Code) {
			out = append(out, finding)
			out = append(out, suppressionFinding(query.WarningSuppressionNotAllowed, "finding is not suppressible", finding.Location))
			continue
		}
		finding.Suppressed = true
		finding.Suppression = &suppression
		out = append(out, finding)
	}
	return out, nil
}

func suppressionsForFile(path string) ([]query.Suppression, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(b), "\n")
	var suppressions []query.Suppression
	for i, line := range lines {
		suppression, ok, err := query.ParseInlineSuppression(line)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		lineNo := i + 1
		suppression.Location = &query.SourceLocation{File: path, Line: lineNo}
		suppressions = append(suppressions, suppression)
	}
	return suppressions, nil
}

func findSuppressionForFinding(finding Finding, suppressions []query.Suppression) (query.Suppression, bool) {
	for _, suppression := range suppressions {
		if suppression.Code != finding.Code {
			continue
		}
		if finding.Location == nil || suppression.Location == nil {
			return suppression, true
		}
		if suppression.Location.Line == finding.Location.Line || suppression.Location.Line+1 == finding.Location.Line {
			return suppression, true
		}
	}
	return query.Suppression{}, false
}

func findingSuppressible(code string) bool {
	switch code {
	case query.WarningUpdateWithoutWhere, query.WarningDeleteWithoutWhere, query.WarningDestructiveSQL,
		migration.WarningMigrationDropTable, migration.WarningMigrationDropColumn, migration.WarningMigrationTypeNarrowing,
		manifest.WarningStale:
		return false
	default:
		return true
	}
}

func suppressionFinding(code, message string, loc *query.SourceLocation) Finding {
	return Finding{
		Code:              code,
		Level:             query.RiskMedium,
		Message:           message,
		Location:          cloneLocation(loc),
		AnalysisPrecision: query.AnalysisPrecise,
	}
}

func summarize(report ReviewReport) ReviewSummary {
	summary := ReviewSummary{
		ByLevel:     make(map[query.RiskLevel]int),
		HighestRisk: query.RiskLow,
		Suppressed:  len(report.SuppressedFindings),
	}
	for _, finding := range report.Findings {
		if finding.Suppressed {
			continue
		}
		summary.Total++
		summary.ByLevel[finding.Level]++
		if compareRisk(finding.Level, summary.HighestRisk) > 0 {
			summary.HighestRisk = finding.Level
		}
	}
	return summary
}

// HasFindingsAtOrAbove reports whether report should fail CI at threshold.
func HasFindingsAtOrAbove(report ReviewReport, threshold query.RiskLevel) bool {
	for _, finding := range report.Findings {
		if finding.Suppressed {
			continue
		}
		if compareRisk(finding.Level, threshold) >= 0 {
			return true
		}
	}
	return false
}

// ParseRiskLevel parses a CLI threshold.
func ParseRiskLevel(s string) (query.RiskLevel, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "high":
		return query.RiskHigh, nil
	case "low":
		return query.RiskLow, nil
	case "medium":
		return query.RiskMedium, nil
	case "destructive":
		return query.RiskDestructive, nil
	case "blocked":
		return query.RiskBlocked, nil
	default:
		return "", fmt.Errorf("unknown risk level %q", s)
	}
}

func compareRisk(a, b query.RiskLevel) int {
	return riskRank(a) - riskRank(b)
}

func riskRank(level query.RiskLevel) int {
	switch level {
	case query.RiskLow, "":
		return 0
	case query.RiskMedium:
		return 1
	case query.RiskHigh:
		return 2
	case query.RiskDestructive:
		return 3
	case query.RiskBlocked:
		return 4
	default:
		return 0
	}
}
