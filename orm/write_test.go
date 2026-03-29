package orm

import (
	"context"
	"database/sql"
	"reflect"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/faciam-dev/goquent/orm/driver"
)

type captureExecutor struct {
	query string
	args  []any
}

func (e *captureExecutor) Query(string, ...any) (*sql.Rows, error) { return nil, nil }

func (e *captureExecutor) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, nil
}

func (e *captureExecutor) QueryRow(string, ...any) *sql.Row { return nil }

func (e *captureExecutor) QueryRowContext(context.Context, string, ...any) *sql.Row { return nil }

func (e *captureExecutor) Exec(string, ...any) (sql.Result, error) { return captureResult{}, nil }

func (e *captureExecutor) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	e.query = query
	e.args = append([]any(nil), args...)
	return captureResult{}, nil
}

type captureResult struct{}

func (captureResult) LastInsertId() (int64, error) { return 0, nil }

func (captureResult) RowsAffected() (int64, error) { return 1, nil }

func newCaptureWriteDB(d driver.Dialect) (*DB, *captureExecutor) {
	exec := &captureExecutor{}
	return &DB{
		drv:      &driver.Driver{Dialect: d},
		exec:     exec,
		scanOpts: ScanOptions{BoolPolicy: BoolCompat},
	}, exec
}

func newReturningMockDB(t *testing.T) (*DB, sqlmock.Sqlmock) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	return NewDB(sqlDB, driver.PostgresDialect{}), mock
}

type genericWriteUser struct {
	ID   int64  `db:"id,pk"`
	Name string `db:"name"`
	Age  int    `db:"age"`
}

func (genericWriteUser) TableName() string { return "users" }

func hasArg(args []any, want any) bool {
	for _, arg := range args {
		if reflect.DeepEqual(arg, want) {
			return true
		}
	}
	return false
}

func TestUpsertStructAlwaysIncludesPKColumn(t *testing.T) {
	db, exec := newCaptureWriteDB(driver.MySQLDialect{})

	_, err := Upsert(
		context.Background(),
		db,
		genericWriteUser{ID: 7, Name: "alice"},
		Columns("name"),
		Omit("id"),
		WherePK(),
	)
	if err != nil {
		t.Fatalf("upsert struct: %v", err)
	}

	if !strings.Contains(exec.query, "INSERT INTO `users`") {
		t.Fatalf("unexpected query: %s", exec.query)
	}
	if !strings.Contains(exec.query, "`id`") {
		t.Fatalf("expected pk column to stay in insert query, got: %s", exec.query)
	}
	if !hasArg(exec.args, int64(7)) {
		t.Fatalf("expected pk value in args, got: %#v", exec.args)
	}
}

func TestUpsertMapAlwaysIncludesPKColumn(t *testing.T) {
	db, exec := newCaptureWriteDB(driver.MySQLDialect{})

	_, err := Upsert(
		context.Background(),
		db,
		map[string]any{"id": int64(9), "name": "bob"},
		Table("users"),
		PK("id"),
		Columns("name"),
		Omit("id"),
		WherePK(),
	)
	if err != nil {
		t.Fatalf("upsert map: %v", err)
	}

	if !strings.Contains(exec.query, "INSERT INTO `users`") {
		t.Fatalf("unexpected query: %s", exec.query)
	}
	if !strings.Contains(exec.query, "`id`") {
		t.Fatalf("expected pk column to stay in insert query, got: %s", exec.query)
	}
	if !hasArg(exec.args, int64(9)) {
		t.Fatalf("expected pk value in args, got: %#v", exec.args)
	}
}

func TestUpdateStructUsesSetArgsBeforePKArgs(t *testing.T) {
	db, exec := newCaptureWriteDB(driver.MySQLDialect{})

	_, err := Update(
		context.Background(),
		db,
		genericWriteUser{ID: 3, Name: "alice"},
		Columns("name"),
		WherePK(),
	)
	if err != nil {
		t.Fatalf("update struct: %v", err)
	}

	if !strings.Contains(exec.query, "SET `name`=? WHERE `id`=?") {
		t.Fatalf("unexpected query: %s", exec.query)
	}
	if len(exec.args) != 2 || exec.args[0] != "alice" || exec.args[1] != int64(3) {
		t.Fatalf("unexpected arg order: %#v", exec.args)
	}
}

func TestUpdateMapUsesSetArgsBeforePKArgs(t *testing.T) {
	db, exec := newCaptureWriteDB(driver.MySQLDialect{})

	_, err := Update(
		context.Background(),
		db,
		map[string]any{"id": int64(4), "name": "bob"},
		Table("users"),
		PK("id"),
		Columns("name"),
		WherePK(),
	)
	if err != nil {
		t.Fatalf("update map: %v", err)
	}

	if !strings.Contains(exec.query, "SET `name`=? WHERE `id`=?") {
		t.Fatalf("unexpected query: %s", exec.query)
	}
	if len(exec.args) != 2 || exec.args[0] != "bob" || exec.args[1] != int64(4) {
		t.Fatalf("unexpected arg order: %#v", exec.args)
	}
}

func TestInsertReturningPostgresAddsClause(t *testing.T) {
	db, mock := newReturningMockDB(t)
	mock.ExpectQuery(`INSERT INTO "users".*RETURNING "id", "name"$`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "alice"))

	res, err := Insert(
		context.Background(),
		db,
		genericWriteUser{Name: "alice"},
		Columns("name"),
		Returning("id", "name"),
	)
	if err != nil {
		t.Fatalf("insert returning: %v", err)
	}
	if aff, err := res.RowsAffected(); err != nil || aff != 1 {
		t.Fatalf("expected rows affected 1, got %d err=%v", aff, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestUpdateReturningPostgresAddsClause(t *testing.T) {
	db, mock := newReturningMockDB(t)
	mock.ExpectQuery(`UPDATE "users" SET .* RETURNING "id", "name"$`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(3, "alice"))

	res, err := Update(
		context.Background(),
		db,
		genericWriteUser{ID: 3, Name: "alice"},
		Columns("name"),
		WherePK(),
		Returning("id", "name"),
	)
	if err != nil {
		t.Fatalf("update returning: %v", err)
	}
	if aff, err := res.RowsAffected(); err != nil || aff != 1 {
		t.Fatalf("expected rows affected 1, got %d err=%v", aff, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestUpsertReturningPostgresAddsClause(t *testing.T) {
	db, mock := newReturningMockDB(t)
	mock.ExpectQuery(`INSERT INTO "users".*ON CONFLICT \("id"\) DO UPDATE SET .* RETURNING "id", "name"$`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(5, "alice"))

	res, err := Upsert(
		context.Background(),
		db,
		genericWriteUser{ID: 5, Name: "alice"},
		WherePK(),
		Returning("id", "name"),
	)
	if err != nil {
		t.Fatalf("upsert returning: %v", err)
	}
	if aff, err := res.RowsAffected(); err != nil || aff != 1 {
		t.Fatalf("expected rows affected 1, got %d err=%v", aff, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
