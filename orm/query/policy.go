package query

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

const (
	WarningTenantFilterMissing     = "TENANT_FILTER_MISSING"
	WarningSoftDeleteFilterMissing = "SOFT_DELETE_FILTER_MISSING"
	WarningPIIColumnSelected       = "PII_COLUMN_SELECTED"
	WarningRequiredFilterMissing   = "REQUIRED_FILTER_MISSING"
)

// PolicyMode controls how policy violations are represented in a QueryPlan.
type PolicyMode string

const (
	PolicyModeWarn    PolicyMode = "warn"
	PolicyModeEnforce PolicyMode = "enforce"
	PolicyModeBlock   PolicyMode = "block"
)

// TablePolicy describes application-specific safety policy for a table.
type TablePolicy struct {
	Table                 string     `json:"table"`
	TenantColumn          string     `json:"tenant_column,omitempty"`
	TenantMode            PolicyMode `json:"tenant_mode,omitempty"`
	SoftDeleteColumn      string     `json:"soft_delete_column,omitempty"`
	SoftDeleteMode        PolicyMode `json:"soft_delete_mode,omitempty"`
	PIIColumns            []string   `json:"pii_columns,omitempty"`
	PIIMode               PolicyMode `json:"pii_mode,omitempty"`
	RequiredFilterColumns []string   `json:"required_filter_columns,omitempty"`
	RequiredFilterMode    PolicyMode `json:"required_filter_mode,omitempty"`
}

var policyRegistry = struct {
	sync.RWMutex
	byTable map[string]TablePolicy
}{byTable: make(map[string]TablePolicy)}

// RegisterTablePolicy registers or replaces a table policy.
func RegisterTablePolicy(policy TablePolicy) error {
	policy.Table = strings.TrimSpace(policy.Table)
	if policy.Table == "" {
		return fmt.Errorf("goquent: policy table is required")
	}
	policy = normalizeTablePolicy(policy)
	policyRegistry.Lock()
	defer policyRegistry.Unlock()
	policyRegistry.byTable[policy.Table] = cloneTablePolicy(policy)
	return nil
}

// PolicyForTable returns a registered policy for table.
func PolicyForTable(table string) (TablePolicy, bool) {
	policyRegistry.RLock()
	defer policyRegistry.RUnlock()
	policy, ok := policyRegistry.byTable[table]
	if !ok {
		return TablePolicy{}, false
	}
	return cloneTablePolicy(policy), true
}

// RegisteredTablePolicies returns all registered table policies in stable order.
func RegisteredTablePolicies() []TablePolicy {
	policyRegistry.RLock()
	defer policyRegistry.RUnlock()
	policies := make([]TablePolicy, 0, len(policyRegistry.byTable))
	for _, policy := range policyRegistry.byTable {
		policies = append(policies, cloneTablePolicy(policy))
	}
	sort.Slice(policies, func(i, j int) bool {
		return policies[i].Table < policies[j].Table
	})
	return policies
}

// ResetPolicyRegistry clears registered policies. Intended for tests.
func ResetPolicyRegistry() {
	policyRegistry.Lock()
	defer policyRegistry.Unlock()
	policyRegistry.byTable = make(map[string]TablePolicy)
}

func normalizeTablePolicy(policy TablePolicy) TablePolicy {
	policy.TenantColumn = strings.TrimSpace(policy.TenantColumn)
	policy.SoftDeleteColumn = strings.TrimSpace(policy.SoftDeleteColumn)
	policy.TenantMode = defaultPolicyMode(policy.TenantMode, PolicyModeEnforce)
	policy.SoftDeleteMode = defaultPolicyMode(policy.SoftDeleteMode, PolicyModeEnforce)
	policy.PIIMode = defaultPolicyMode(policy.PIIMode, PolicyModeWarn)
	policy.RequiredFilterMode = defaultPolicyMode(policy.RequiredFilterMode, PolicyModeEnforce)
	policy.PIIColumns = normalizeColumns(policy.PIIColumns)
	policy.RequiredFilterColumns = normalizeColumns(policy.RequiredFilterColumns)
	return policy
}

func defaultPolicyMode(mode, fallback PolicyMode) PolicyMode {
	switch mode {
	case PolicyModeWarn, PolicyModeEnforce, PolicyModeBlock:
		return mode
	default:
		return fallback
	}
}

func normalizeColumns(cols []string) []string {
	seen := make(map[string]struct{}, len(cols))
	out := make([]string, 0, len(cols))
	for _, col := range cols {
		col = strings.TrimSpace(col)
		if col == "" {
			continue
		}
		key := normalizeColumnName(col)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, col)
	}
	return out
}

func cloneTablePolicy(policy TablePolicy) TablePolicy {
	policy.PIIColumns = append([]string(nil), policy.PIIColumns...)
	policy.RequiredFilterColumns = append([]string(nil), policy.RequiredFilterColumns...)
	return policy
}

func checkPolicy(plan *QueryPlan, policy *TablePolicy) []Warning {
	if plan == nil || policy == nil || policy.Table == "" {
		return nil
	}
	if !planTouchesTable(plan, policy.Table) {
		return nil
	}

	var warnings []Warning
	if policy.TenantColumn != "" && policyAppliesToOperation(plan.Operation) && !hasPredicateColumn(plan, policy.TenantColumn) {
		warnings = append(warnings, policyWarning(
			WarningTenantFilterMissing,
			policyModeLevel(policy.TenantMode, RiskHigh),
			fmt.Sprintf("%s is tenant-scoped but %s filter is missing", policy.Table, policy.TenantColumn),
			"add a tenant filter before executing this query",
			false,
		))
	}
	for _, col := range policy.RequiredFilterColumns {
		if policyAppliesToOperation(plan.Operation) && !hasPredicateColumn(plan, col) {
			warnings = append(warnings, policyWarning(
				WarningRequiredFilterMissing,
				policyModeLevel(policy.RequiredFilterMode, RiskHigh),
				fmt.Sprintf("%s requires a filter on %s", policy.Table, col),
				"add the required filter before executing this query",
				false,
			))
		}
	}
	if policy.SoftDeleteColumn != "" && policyAppliesToOperation(plan.Operation) && shouldRequireSoftDeleteFilter(plan) && !hasPredicateColumn(plan, policy.SoftDeleteColumn) {
		warnings = append(warnings, policyWarning(
			WarningSoftDeleteFilterMissing,
			policyModeLevel(policy.SoftDeleteMode, RiskMedium),
			fmt.Sprintf("%s has soft delete policy but %s filter is missing", policy.Table, policy.SoftDeleteColumn),
			"use the default soft delete filter or explicitly call WithDeleted",
			true,
		))
	}
	if plan.Operation == OperationSelect {
		for _, col := range selectedPIIColumns(plan, policy.PIIColumns) {
			w := policyWarning(
				WarningPIIColumnSelected,
				policyModeLevel(policy.PIIMode, RiskMedium),
				fmt.Sprintf("PII column selected: %s.%s", policy.Table, col),
				"avoid selecting PII or include a narrow access reason",
				true,
			)
			w.RequiresReason = true
			if reason, ok := plan.Metadata["access_reason"].(string); ok && reason != "" {
				w.Evidence = append(w.Evidence, Evidence{Key: "access_reason", Value: reason})
			}
			warnings = append(warnings, w)
		}
	}
	return warnings
}

func policyWarning(code string, level RiskLevel, message, hint string, suppressible bool) Warning {
	return Warning{
		Code:         code,
		Level:        level,
		Message:      message,
		Hint:         hint,
		Suppressible: suppressible && level != RiskBlocked,
	}
}

func policyModeLevel(mode PolicyMode, fallback RiskLevel) RiskLevel {
	switch mode {
	case PolicyModeWarn:
		return fallback
	case PolicyModeEnforce:
		if compareRisk(fallback, RiskHigh) < 0 {
			return RiskHigh
		}
		return fallback
	case PolicyModeBlock:
		return RiskBlocked
	default:
		return fallback
	}
}

func policyAppliesToOperation(op OperationType) bool {
	return op == OperationSelect || op == OperationUpdate || op == OperationDelete
}

func planTouchesTable(plan *QueryPlan, table string) bool {
	for _, ref := range plan.Tables {
		if ref.Name == table {
			return true
		}
	}
	return false
}

func hasPredicateColumn(plan *QueryPlan, column string) bool {
	target := normalizeColumnName(column)
	for _, predicate := range plan.Predicates {
		if normalizeColumnName(predicate.Column) == target {
			return true
		}
		if normalizeColumnName(predicate.ValueColumn) == target {
			return true
		}
	}
	return false
}

func selectedPIIColumns(plan *QueryPlan, piiColumns []string) []string {
	if len(piiColumns) == 0 {
		return nil
	}
	selectedAll := false
	selected := make(map[string]struct{})
	for _, column := range plan.Columns {
		if strings.TrimSpace(column.Name) == "*" || strings.TrimSpace(column.Expression) == "*" {
			selectedAll = true
			continue
		}
		if column.Name != "" {
			selected[normalizeColumnName(column.Name)] = struct{}{}
		}
	}
	var out []string
	for _, pii := range piiColumns {
		if selectedAll {
			out = append(out, pii)
			continue
		}
		if _, ok := selected[normalizeColumnName(pii)]; ok {
			out = append(out, pii)
		}
	}
	return out
}

func shouldRequireSoftDeleteFilter(plan *QueryPlan) bool {
	if plan.Metadata != nil {
		if v, ok := plan.Metadata["soft_delete"].(string); ok && v == "with_deleted" {
			return false
		}
	}
	return true
}

func normalizeColumnName(column string) string {
	column = strings.TrimSpace(column)
	column = strings.Trim(column, "`\"")
	if idx := strings.LastIndex(column, "."); idx >= 0 {
		column = column[idx+1:]
	}
	column = strings.Trim(column, "`\"")
	return strings.ToLower(column)
}
