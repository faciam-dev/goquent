package manifest

import (
	"encoding/json"
	"fmt"
	"io"
)

// WriteJSON writes a stable manifest JSON document.
func WriteJSON(w io.Writer, m *Manifest) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(m)
}

// WritePretty writes a compact human-readable manifest summary.
func WritePretty(w io.Writer, m *Manifest) error {
	if m == nil {
		_, err := fmt.Fprintln(w, "Manifest\n\nNo manifest.")
		return err
	}
	if _, err := fmt.Fprintln(w, "Manifest"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "\nversion: %s\ngenerated_at: %s\n", m.Version, m.GeneratedAt.Format("2006-01-02T15:04:05Z07:00")); err != nil {
		return err
	}
	if m.Dialect != "" {
		if _, err := fmt.Fprintf(w, "dialect: %s\n", m.Dialect); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "schema_fingerprint: %s\npolicy_fingerprint: %s\n", m.SchemaFingerprint, m.PolicyFingerprint); err != nil {
		return err
	}
	if m.GeneratedCodeFingerprint != "" {
		if _, err := fmt.Fprintf(w, "generated_code_fingerprint: %s\n", m.GeneratedCodeFingerprint); err != nil {
			return err
		}
	}
	if m.DatabaseFingerprint != "" {
		if _, err := fmt.Fprintf(w, "database_fingerprint: %s\n", m.DatabaseFingerprint); err != nil {
			return err
		}
	}
	if m.Verification != nil {
		if _, err := fmt.Fprintf(w, "fresh: %t\n", m.Verification.Fresh); err != nil {
			return err
		}
	}
	if len(m.Tables) == 0 {
		_, err := fmt.Fprintln(w, "\nNo tables.")
		return err
	}
	for _, table := range m.Tables {
		if _, err := fmt.Fprintf(w, "\n%s", table.Name); err != nil {
			return err
		}
		if table.Model != "" {
			if _, err := fmt.Fprintf(w, " model=%s", table.Model); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
		for _, column := range table.Columns {
			if _, err := fmt.Fprintf(w, "  column: %s", column.Name); err != nil {
				return err
			}
			if column.Type != "" {
				if _, err := fmt.Fprintf(w, " type=%s", column.Type); err != nil {
					return err
				}
			}
			if column.Primary {
				if _, err := fmt.Fprint(w, " primary"); err != nil {
					return err
				}
			}
			if column.PII {
				if _, err := fmt.Fprint(w, " pii"); err != nil {
					return err
				}
			}
			if column.TenantScope {
				if _, err := fmt.Fprint(w, " tenant_scope"); err != nil {
					return err
				}
			}
			if column.SoftDelete {
				if _, err := fmt.Fprint(w, " soft_delete"); err != nil {
					return err
				}
			}
			if column.RequiredFilter {
				if _, err := fmt.Fprint(w, " required_filter"); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
		for _, policy := range table.Policies {
			if _, err := fmt.Fprintf(w, "  policy: %s column=%s mode=%s\n", policy.Type, policy.Column, policy.Mode); err != nil {
				return err
			}
		}
	}
	return nil
}

// WriteVerificationPretty writes freshness checks in a human-readable form.
func WriteVerificationPretty(w io.Writer, v Verification) error {
	if _, err := fmt.Fprintln(w, "Manifest Verification"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "\nfresh: %t\nchecked_at: %s\n", v.Fresh, v.CheckedAt.Format("2006-01-02T15:04:05Z07:00")); err != nil {
		return err
	}
	for _, check := range v.Checks {
		if _, err := fmt.Fprintf(w, "\n[%s] %s", check.Status, check.Name); err != nil {
			return err
		}
		if check.Message != "" {
			if _, err := fmt.Fprintf(w, " - %s", check.Message); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
		if check.Expected != "" || check.Actual != "" {
			if _, err := fmt.Fprintf(w, "  expected: %s\n  actual: %s\n", check.Expected, check.Actual); err != nil {
				return err
			}
		}
	}
	return nil
}

// WriteVerificationJSON writes a machine-readable freshness result.
func WriteVerificationJSON(w io.Writer, v Verification) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
