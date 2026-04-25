package query

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"

	ormdriver "github.com/faciam-dev/goquent/orm/driver"
)

type recordingExec struct {
	calls int
}

func (e *recordingExec) Query(string, ...any) (*sql.Rows, error) {
	e.calls++
	return nil, errors.New("unexpected query")
}

func (e *recordingExec) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	e.calls++
	return nil, errors.New("unexpected query context")
}

func (e *recordingExec) QueryRow(string, ...any) *sql.Row {
	e.calls++
	return &sql.Row{}
}

func (e *recordingExec) QueryRowContext(context.Context, string, ...any) *sql.Row {
	e.calls++
	return &sql.Row{}
}

func (e *recordingExec) Exec(string, ...any) (sql.Result, error) {
	e.calls++
	return driver.RowsAffected(0), errors.New("unexpected exec")
}

func (e *recordingExec) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	e.calls++
	return driver.RowsAffected(0), errors.New("unexpected exec context")
}

func newPlanTestQuery(exec *recordingExec) *Query {
	return New(exec, "users", ormdriver.MySQLDialect{})
}

func TestSelectPlanSnapshot(t *testing.T) {
	exec := &recordingExec{}
	plan, err := newPlanTestQuery(exec).
		Select("id", "name").
		Where("id", 10).
		OrderBy("id", "asc").
		Limit(1).
		Plan(context.Background())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	if exec.calls != 0 {
		t.Fatalf("Plan executed database call count=%d", exec.calls)
	}
	if plan.Operation != OperationSelect {
		t.Fatalf("operation=%s", plan.Operation)
	}
	if plan.SQL != "SELECT `id`, `name` FROM `users` WHERE `id` = ? ORDER BY `id` ASC LIMIT 1" {
		t.Fatalf("sql=%q", plan.SQL)
	}
	if len(plan.Params) != 1 || plan.Params[0] != 10 {
		t.Fatalf("params=%#v", plan.Params)
	}
	if len(plan.Tables) != 1 || plan.Tables[0].Name != "users" {
		t.Fatalf("tables=%#v", plan.Tables)
	}
	if len(plan.Columns) != 2 || plan.Columns[0].Name != "id" || plan.Columns[1].Name != "name" {
		t.Fatalf("columns=%#v", plan.Columns)
	}
	if len(plan.Predicates) != 1 || plan.Predicates[0].Column != "id" || plan.Predicates[0].Operator != "=" {
		t.Fatalf("predicates=%#v", plan.Predicates)
	}
	if plan.Limit == nil || *plan.Limit != 1 {
		t.Fatalf("limit=%v", plan.Limit)
	}
}

func TestWritePlanSnapshots(t *testing.T) {
	ctx := context.Background()

	t.Run("insert", func(t *testing.T) {
		exec := &recordingExec{}
		plan, err := newPlanTestQuery(exec).PlanInsert(ctx, map[string]any{"name": "alice", "age": 30})
		if err != nil {
			t.Fatalf("PlanInsert: %v", err)
		}
		if exec.calls != 0 {
			t.Fatalf("PlanInsert executed database call count=%d", exec.calls)
		}
		if plan.Operation != OperationInsert {
			t.Fatalf("operation=%s", plan.Operation)
		}
		if plan.SQL != "INSERT INTO `users` (`age`, `name`) VALUES (?, ?)" {
			t.Fatalf("sql=%q", plan.SQL)
		}
		if len(plan.Params) != 2 || plan.Params[0] != 30 || plan.Params[1] != "alice" {
			t.Fatalf("params=%#v", plan.Params)
		}
		if len(plan.Columns) != 2 || plan.Columns[0].Name != "age" || plan.Columns[1].Name != "name" {
			t.Fatalf("columns=%#v", plan.Columns)
		}
	})

	t.Run("update", func(t *testing.T) {
		exec := &recordingExec{}
		plan, err := newPlanTestQuery(exec).
			Where("id", 10).
			PlanUpdate(ctx, map[string]any{"name": "alice"})
		if err != nil {
			t.Fatalf("PlanUpdate: %v", err)
		}
		if exec.calls != 0 {
			t.Fatalf("PlanUpdate executed database call count=%d", exec.calls)
		}
		if plan.Operation != OperationUpdate {
			t.Fatalf("operation=%s", plan.Operation)
		}
		if plan.SQL != "UPDATE `users` SET `name` = ? WHERE `id` = ?" {
			t.Fatalf("sql=%q", plan.SQL)
		}
		if len(plan.Params) != 2 || plan.Params[0] != "alice" || plan.Params[1] != 10 {
			t.Fatalf("params=%#v", plan.Params)
		}
		if len(plan.Predicates) != 1 || plan.Predicates[0].Column != "id" {
			t.Fatalf("predicates=%#v", plan.Predicates)
		}
	})

	t.Run("delete", func(t *testing.T) {
		exec := &recordingExec{}
		plan, err := newPlanTestQuery(exec).
			Where("id", 10).
			PlanDelete(ctx)
		if err != nil {
			t.Fatalf("PlanDelete: %v", err)
		}
		if exec.calls != 0 {
			t.Fatalf("PlanDelete executed database call count=%d", exec.calls)
		}
		if plan.Operation != OperationDelete {
			t.Fatalf("operation=%s", plan.Operation)
		}
		if plan.SQL != "DELETE FROM `users` WHERE `id` = ?" {
			t.Fatalf("sql=%q", plan.SQL)
		}
		if len(plan.Params) != 1 || plan.Params[0] != 10 {
			t.Fatalf("params=%#v", plan.Params)
		}
	})
}

func TestRawPlanSnapshot(t *testing.T) {
	plan := NewRawPlan("DELETE FROM users WHERE id = ?", 10)
	if plan.Operation != OperationRaw {
		t.Fatalf("operation=%s", plan.Operation)
	}
	if plan.RiskLevel != RiskHigh {
		t.Fatalf("risk=%s", plan.RiskLevel)
	}
	if len(plan.Warnings) != 1 || plan.Warnings[0].Code != WarningRawSQLUsed {
		t.Fatalf("warnings=%#v", plan.Warnings)
	}
	if len(plan.Params) != 1 || plan.Params[0] != 10 {
		t.Fatalf("params=%#v", plan.Params)
	}
	if _, err := plan.ToJSON(); err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	if !strings.Contains(plan.String(), "RAW_SQL_USED") {
		t.Fatalf("String()=%q", plan.String())
	}
}

func TestPlanParamsOrderingAndInjectionRegression(t *testing.T) {
	injected := "alice' OR 1=1 --"
	plan, err := newPlanTestQuery(&recordingExec{}).
		Where("age", ">", 20).
		Where("name", injected).
		Plan(context.Background())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Params) != 2 || plan.Params[0] != 20 || plan.Params[1] != injected {
		t.Fatalf("params=%#v", plan.Params)
	}
	if strings.Contains(plan.SQL, injected) {
		t.Fatalf("SQL contains injected value: %q", plan.SQL)
	}
	if got := strings.Count(plan.SQL, "?"); got != 2 {
		t.Fatalf("placeholder count=%d sql=%q", got, plan.SQL)
	}
}
