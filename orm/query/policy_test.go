package query

import (
	"context"
	"errors"
	"strings"
	"testing"

	ormdriver "github.com/faciam-dev/goquent/orm/driver"
)

func registerUsersPolicy(t *testing.T, policy TablePolicy) {
	t.Helper()
	ResetPolicyRegistry()
	if policy.Table == "" {
		policy.Table = "users"
	}
	if err := RegisterTablePolicy(policy); err != nil {
		t.Fatalf("RegisterTablePolicy: %v", err)
	}
	t.Cleanup(ResetPolicyRegistry)
}

func newPolicyTestQuery(exec *recordingExec) *Query {
	return New(exec, "users", ormdriver.MySQLDialect{})
}

func TestTenantScopedPolicyWarnings(t *testing.T) {
	registerUsersPolicy(t, TablePolicy{TenantColumn: "tenant_id"})

	plan, err := newPolicyTestQuery(&recordingExec{}).
		Select("id").
		Limit(10).
		Plan(context.Background())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	if !warningCodeSet(plan.Warnings)[WarningTenantFilterMissing] {
		t.Fatalf("warnings=%#v", plan.Warnings)
	}
	if plan.RiskLevel != RiskHigh || !plan.RequiredApproval {
		t.Fatalf("risk=%s required=%v warnings=%#v", plan.RiskLevel, plan.RequiredApproval, plan.Warnings)
	}

	plan, err = newPolicyTestQuery(&recordingExec{}).
		Select("id").
		Where("tenant_id", 7).
		Limit(10).
		Plan(context.Background())
	if err != nil {
		t.Fatalf("Plan with tenant: %v", err)
	}
	if warningCodeSet(plan.Warnings)[WarningTenantFilterMissing] {
		t.Fatalf("unexpected tenant warning=%#v", plan.Warnings)
	}
}

func TestTablePolicyAppliesToAliasedTable(t *testing.T) {
	registerUsersPolicy(t, TablePolicy{TenantColumn: "tenant_id"})

	plan, err := New(&recordingExec{}, "users as u", ormdriver.MySQLDialect{}).
		Select("u.id").
		Limit(10).
		Plan(context.Background())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if !warningCodeSet(plan.Warnings)[WarningTenantFilterMissing] {
		t.Fatalf("expected tenant warning for aliased table, got %#v", plan.Warnings)
	}
}

func TestTenantScopedPolicyOnWrites(t *testing.T) {
	registerUsersPolicy(t, TablePolicy{TenantColumn: "tenant_id"})
	ctx := context.Background()

	updatePlan, err := newPolicyTestQuery(&recordingExec{}).
		Where("name", "alice").
		PlanUpdate(ctx, map[string]any{"age": 31})
	if err != nil {
		t.Fatalf("PlanUpdate: %v", err)
	}
	if !warningCodeSet(updatePlan.Warnings)[WarningTenantFilterMissing] {
		t.Fatalf("update warnings=%#v", updatePlan.Warnings)
	}

	deletePlan, err := newPolicyTestQuery(&recordingExec{}).
		Where("name", "alice").
		PlanDelete(ctx)
	if err != nil {
		t.Fatalf("PlanDelete: %v", err)
	}
	if !warningCodeSet(deletePlan.Warnings)[WarningTenantFilterMissing] {
		t.Fatalf("delete warnings=%#v", deletePlan.Warnings)
	}

	updatePlan, err = newPolicyTestQuery(&recordingExec{}).
		Where("tenant_id", 7).
		Where("name", "alice").
		PlanUpdate(ctx, map[string]any{"age": 31})
	if err != nil {
		t.Fatalf("PlanUpdate with tenant: %v", err)
	}
	if warningCodeSet(updatePlan.Warnings)[WarningTenantFilterMissing] {
		t.Fatalf("unexpected tenant warning=%#v", updatePlan.Warnings)
	}
}

func TestTenantBlockModeRejectsExecution(t *testing.T) {
	registerUsersPolicy(t, TablePolicy{TenantColumn: "tenant_id", TenantMode: PolicyModeBlock})

	exec := &recordingExec{}
	err := newPolicyTestQuery(exec).
		Select("id").
		Limit(10).
		GetMaps(&[]map[string]any{})
	if !errors.Is(err, ErrBlockedOperation) {
		t.Fatalf("GetMaps error=%v", err)
	}
	if exec.calls != 0 {
		t.Fatalf("blocked policy query executed database call count=%d", exec.calls)
	}
}

func TestSoftDeleteDefaultWithDeletedAndOnlyDeleted(t *testing.T) {
	registerUsersPolicy(t, TablePolicy{SoftDeleteColumn: "deleted_at"})

	plan, err := newPolicyTestQuery(&recordingExec{}).
		Select("id").
		Limit(10).
		Plan(context.Background())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if !strings.Contains(plan.SQL, "`deleted_at` IS NULL") {
		t.Fatalf("expected soft delete predicate in SQL: %q", plan.SQL)
	}
	if warningCodeSet(plan.Warnings)[WarningSoftDeleteFilterMissing] {
		t.Fatalf("unexpected soft delete warning: %#v", plan.Warnings)
	}

	plan, err = newPolicyTestQuery(&recordingExec{}).
		WithDeleted().
		Select("id").
		Limit(10).
		Plan(context.Background())
	if err != nil {
		t.Fatalf("Plan with deleted: %v", err)
	}
	if strings.Contains(plan.SQL, "`deleted_at`") {
		t.Fatalf("with deleted should not inject deleted_at predicate: %q", plan.SQL)
	}
	if plan.Metadata["soft_delete"] != "with_deleted" {
		t.Fatalf("metadata=%#v", plan.Metadata)
	}

	plan, err = newPolicyTestQuery(&recordingExec{}).
		OnlyDeleted().
		Select("id").
		Limit(10).
		Plan(context.Background())
	if err != nil {
		t.Fatalf("Plan only deleted: %v", err)
	}
	if !strings.Contains(plan.SQL, "`deleted_at` IS NOT NULL") {
		t.Fatalf("expected only deleted predicate in SQL: %q", plan.SQL)
	}
}

func TestPIIPolicyAndAccessReason(t *testing.T) {
	registerUsersPolicy(t, TablePolicy{
		TenantColumn: "tenant_id",
		PIIColumns:   []string{"email"},
	})

	plan, err := newPolicyTestQuery(&recordingExec{}).
		Select("id", "email").
		Where("tenant_id", 7).
		Limit(10).
		AccessReason("customer support lookup").
		Plan(context.Background())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if !warningCodeSet(plan.Warnings)[WarningPIIColumnSelected] {
		t.Fatalf("warnings=%#v", plan.Warnings)
	}
	if plan.Metadata["access_reason"] != "customer support lookup" {
		t.Fatalf("metadata=%#v", plan.Metadata)
	}
	var foundEvidence bool
	for _, warning := range plan.Warnings {
		if warning.Code != WarningPIIColumnSelected {
			continue
		}
		for _, evidence := range warning.Evidence {
			if evidence.Key == "access_reason" && evidence.Value == "customer support lookup" {
				foundEvidence = true
			}
		}
	}
	if !foundEvidence {
		t.Fatalf("missing access reason evidence: %#v", plan.Warnings)
	}

	_, err = newPolicyTestQuery(&recordingExec{}).
		AccessReason(" ").
		Plan(context.Background())
	if !errors.Is(err, ErrAccessReasonRequired) {
		t.Fatalf("Plan error=%v", err)
	}
}

func TestRequiredFilterPolicy(t *testing.T) {
	registerUsersPolicy(t, TablePolicy{RequiredFilterColumns: []string{"organization_id"}})

	plan, err := newPolicyTestQuery(&recordingExec{}).
		Select("id").
		Limit(10).
		Plan(context.Background())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if !warningCodeSet(plan.Warnings)[WarningRequiredFilterMissing] {
		t.Fatalf("warnings=%#v", plan.Warnings)
	}

	plan, err = newPolicyTestQuery(&recordingExec{}).
		Select("id").
		Where("organization_id", 42).
		Limit(10).
		Plan(context.Background())
	if err != nil {
		t.Fatalf("Plan with required filter: %v", err)
	}
	if warningCodeSet(plan.Warnings)[WarningRequiredFilterMissing] {
		t.Fatalf("unexpected required filter warning=%#v", plan.Warnings)
	}
}
