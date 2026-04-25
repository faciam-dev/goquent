package orm

import (
	"context"
	"database/sql"
	"testing"

	"github.com/faciam-dev/goquent/orm/driver"
)

type rawQueryRowExecutor struct {
	queryRows        []string
	queryRowsContext []string
}

func (e *rawQueryRowExecutor) Query(string, ...any) (*sql.Rows, error) { return nil, nil }

func (e *rawQueryRowExecutor) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, nil
}

func (e *rawQueryRowExecutor) QueryRow(q string, _ ...any) *sql.Row {
	e.queryRows = append(e.queryRows, q)
	return nil
}

func (e *rawQueryRowExecutor) QueryRowContext(_ context.Context, q string, _ ...any) *sql.Row {
	e.queryRowsContext = append(e.queryRowsContext, q)
	return nil
}

func (e *rawQueryRowExecutor) Exec(string, ...any) (sql.Result, error) { return nil, nil }

func (e *rawQueryRowExecutor) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	return nil, nil
}

func TestDeprecatedQueryRowDoesNotExecuteUnapprovedRawSQL(t *testing.T) {
	exec := &rawQueryRowExecutor{}
	db := newDB(&driver.Driver{Dialect: driver.MySQLDialect{}}, exec)

	db.QueryRow("DROP TABLE users")
	if len(exec.queryRows) != 0 {
		t.Fatalf("expected QueryRow not to execute caller SQL, got %#v", exec.queryRows)
	}
	if len(exec.queryRowsContext) != 1 || exec.queryRowsContext[0] != rawQueryRowRejectedSQL {
		t.Fatalf("expected rejected sentinel query, got %#v", exec.queryRowsContext)
	}
}

func TestDeprecatedQueryRowContextDoesNotExecuteUnapprovedRawSQL(t *testing.T) {
	exec := &rawQueryRowExecutor{}
	db := newDB(&driver.Driver{Dialect: driver.MySQLDialect{}}, exec)

	db.QueryRowContext(context.Background(), "SELECT * FROM users")
	if len(exec.queryRows) != 0 {
		t.Fatalf("expected QueryRowContext not to execute caller SQL, got %#v", exec.queryRows)
	}
	if len(exec.queryRowsContext) != 1 || exec.queryRowsContext[0] != rawQueryRowRejectedSQL {
		t.Fatalf("expected rejected sentinel query, got %#v", exec.queryRowsContext)
	}
}

func TestDeprecatedQueryRowExecutesApprovedRawSQL(t *testing.T) {
	exec := &rawQueryRowExecutor{}
	db := newDB(&driver.Driver{Dialect: driver.MySQLDialect{}}, exec).
		RequireRawApproval("operator reviewed raw single-row query")

	db.QueryRow("SELECT id FROM users WHERE id = ?", 1)
	if len(exec.queryRows) != 1 || exec.queryRows[0] != "SELECT id FROM users WHERE id = ?" {
		t.Fatalf("expected approved caller SQL, got %#v", exec.queryRows)
	}
	if len(exec.queryRowsContext) != 0 {
		t.Fatalf("expected no rejected sentinel query, got %#v", exec.queryRowsContext)
	}
}
