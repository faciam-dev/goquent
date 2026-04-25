package migration

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/faciam-dev/goquent/orm/query"
)

// WriteJSON writes a machine-readable migration plan.
func WriteJSON(w io.Writer, plan *MigrationPlan) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(plan)
}

// WritePretty writes a human-readable migration plan.
func WritePretty(w io.Writer, plan *MigrationPlan) error {
	if plan == nil {
		_, err := fmt.Fprintln(w, "Migration Plan\n\nNo migration plan.")
		return err
	}
	if _, err := fmt.Fprintln(w, "Migration Plan"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "\nrisk: %s\nprecision: %s\n", plan.RiskLevel, plan.AnalysisPrecision); err != nil {
		return err
	}
	if plan.RequiredApproval {
		if _, err := fmt.Fprintln(w, "requires_approval: true"); err != nil {
			return err
		}
	}
	if len(plan.Steps) == 0 {
		_, err := fmt.Fprintln(w, "\nNo migration steps detected.")
		return err
	}

	for _, step := range plan.Steps {
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "[%s] %s", riskLabel(step.RiskLevel), step.Type); err != nil {
			return err
		}
		if step.Table != "" {
			if _, err := fmt.Fprintf(w, " table=%s", step.Table); err != nil {
				return err
			}
		}
		if step.Column != "" {
			if _, err := fmt.Fprintf(w, " column=%s", step.Column); err != nil {
				return err
			}
		}
		if step.Index != "" {
			if _, err := fmt.Fprintf(w, " index=%s", step.Index); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
		if step.Line > 0 {
			if _, err := fmt.Fprintf(w, "  line: %d\n", step.Line); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(w, "  precision: %s\n", step.AnalysisPrecision); err != nil {
			return err
		}
		for _, warning := range step.Warnings {
			if _, err := fmt.Fprintf(w, "  warning[%s]: %s - %s\n", warning.Level, warning.Code, warning.Message); err != nil {
				return err
			}
			if warning.Hint != "" {
				if _, err := fmt.Fprintf(w, "    hint: %s\n", warning.Hint); err != nil {
					return err
				}
			}
		}
		if len(step.Preflight) > 0 {
			if _, err := fmt.Fprintln(w, "  suggested_preflight:"); err != nil {
				return err
			}
			for _, item := range step.Preflight {
				if _, err := fmt.Fprintf(w, "    - %s\n", item); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func riskLabel(level query.RiskLevel) string {
	switch level {
	case query.RiskLow:
		return "Low"
	case query.RiskMedium:
		return "Medium"
	case query.RiskHigh:
		return "High"
	case query.RiskDestructive:
		return "Destructive"
	case query.RiskBlocked:
		return "Blocked"
	default:
		return string(level)
	}
}
