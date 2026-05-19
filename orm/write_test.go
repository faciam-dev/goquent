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

func TestInsertReturningTypedInfersColumns(t *testing.T) {
	db, mock := newReturningMockDB(t)
	mock.ExpectQuery(`INSERT INTO "users".*RETURNING "id", "name", "age"$`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "age"}).AddRow(1, "alice", 30))

	row, err := InsertReturning[genericWriteUser](
		context.Background(),
		db,
		genericWriteUser{Name: "alice"},
		Columns("name"),
	)
	if err != nil {
		t.Fatalf("insert returning typed: %v", err)
	}
	if row.ID != 1 || row.Name != "alice" || row.Age != 30 {
		t.Fatalf("unexpected row: %+v", row)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestInsertReturningMapUsesExplicitColumns(t *testing.T) {
	db, mock := newReturningMockDB(t)
	mock.ExpectQuery(`INSERT INTO "users".*RETURNING "id", "name"$`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "alice"))

	row, err := InsertReturning[map[string]any](
		context.Background(),
		db,
		map[string]any{"name": "alice"},
		Table("users"),
		Returning("id", "name"),
	)
	if err != nil {
		t.Fatalf("insert returning map: %v", err)
	}
	if row["id"] != int64(1) || row["name"] != "alice" {
		t.Fatalf("unexpected row: %+v", row)
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

func TestUpdateReturningTypedInfersColumns(t *testing.T) {
	db, mock := newReturningMockDB(t)
	mock.ExpectQuery(`UPDATE "users" SET .* RETURNING "id", "name", "age"$`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "age"}).AddRow(3, "alice", 31))

	row, err := UpdateReturning[genericWriteUser](
		context.Background(),
		db,
		genericWriteUser{ID: 3, Name: "alice"},
		Columns("name"),
		WherePK(),
	)
	if err != nil {
		t.Fatalf("update returning typed: %v", err)
	}
	if row.ID != 3 || row.Name != "alice" || row.Age != 31 {
		t.Fatalf("unexpected row: %+v", row)
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

func TestUpsertReturningTypedInfersColumns(t *testing.T) {
	db, mock := newReturningMockDB(t)
	mock.ExpectQuery(`INSERT INTO "users".*ON CONFLICT \("id"\) DO UPDATE SET .* RETURNING "id", "name", "age"$`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "age"}).AddRow(5, "alice", 32))

	row, err := UpsertReturning[genericWriteUser](
		context.Background(),
		db,
		genericWriteUser{ID: 5, Name: "alice"},
		WherePK(),
	)
	if err != nil {
		t.Fatalf("upsert returning typed: %v", err)
	}
	if row.ID != 5 || row.Name != "alice" || row.Age != 32 {
		t.Fatalf("unexpected row: %+v", row)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestUpsertPostgresConflictWhere(t *testing.T) {
	db, exec := newCaptureWriteDB(driver.PostgresDialect{})

	_, err := Upsert(
		context.Background(),
		db,
		map[string]any{
			"id":              "audit-1",
			"tenant_id":       "tenant-1",
			"idempotency_key": "idem-1",
			"payload_json":    "{}",
		},
		Table("ai_audit_logs"),
		ConflictColumns("tenant_id", "idempotency_key"),
		ConflictWhere("idempotency_key <> ''"),
	)
	if err != nil {
		t.Fatalf("upsert conflict where: %v", err)
	}
	if !strings.Contains(exec.query, `ON CONFLICT ("tenant_id", "idempotency_key") WHERE idempotency_key <> '' DO UPDATE SET`) {
		t.Fatalf("expected partial-index conflict target, got: %s", exec.query)
	}
	if strings.Contains(exec.query, `"tenant_id"=EXCLUDED."tenant_id"`) || strings.Contains(exec.query, `"idempotency_key"=EXCLUDED."idempotency_key"`) {
		t.Fatalf("conflict columns should not be updated: %s", exec.query)
	}
}

func TestUpsertPostgresUpdateColumnsSeparatesInsertAndUpdate(t *testing.T) {
	db, exec := newCaptureWriteDB(driver.PostgresDialect{})

	_, err := Upsert(
		context.Background(),
		db,
		map[string]any{
			"id":               "field-1",
			"tenant_id":        "tenant-1",
			"form_instance_id": "form-1",
			"field_key":        "weekly_hours",
			"value_text":       "40",
			"needs_update":     false,
		},
		Table("form_fields"),
		ConflictColumns("tenant_id", "form_instance_id", "field_key"),
		UpdateColumns("value_text", "needs_update"),
	)
	if err != nil {
		t.Fatalf("upsert update columns: %v", err)
	}
	if !strings.Contains(exec.query, `"id"`) {
		t.Fatalf("expected insert-only id column to be present, got: %s", exec.query)
	}
	if !strings.Contains(exec.query, `"value_text"=EXCLUDED."value_text"`) || !strings.Contains(exec.query, `"needs_update"=EXCLUDED."needs_update"`) {
		t.Fatalf("expected explicit update columns, got: %s", exec.query)
	}
	for _, col := range []string{`"id"=EXCLUDED."id"`, `"tenant_id"=EXCLUDED."tenant_id"`, `"form_instance_id"=EXCLUDED."form_instance_id"`, `"field_key"=EXCLUDED."field_key"`} {
		if strings.Contains(exec.query, col) {
			t.Fatalf("column %s should not be updated: %s", col, exec.query)
		}
	}
}

func TestUpsertConflictDoNothing(t *testing.T) {
	db, exec := newCaptureWriteDB(driver.PostgresDialect{})

	_, err := Upsert(
		context.Background(),
		db,
		map[string]any{
			"tenant_id":       "tenant-1",
			"idempotency_key": "idem-1",
			"payload_json":    "{}",
		},
		Table("submission_attempts"),
		ConflictColumns("tenant_id", "idempotency_key"),
		ConflictDoNothing(),
	)
	if err != nil {
		t.Fatalf("upsert do nothing: %v", err)
	}
	if !strings.Contains(exec.query, `ON CONFLICT ("tenant_id", "idempotency_key") DO NOTHING`) {
		t.Fatalf("expected DO NOTHING conflict action, got: %s", exec.query)
	}
	if strings.Contains(exec.query, "DO UPDATE") {
		t.Fatalf("expected no conflict update, got: %s", exec.query)
	}
}

func TestUpsertPostgresConflictTargetRaw(t *testing.T) {
	db, exec := newCaptureWriteDB(driver.PostgresDialect{})

	_, err := Upsert(
		context.Background(),
		db,
		map[string]any{
			"tenant_id":      "tenant-1",
			"target_node_id": nil,
			"payload_json":   "{}",
		},
		Table("citation_links"),
		ConflictTargetRaw(`("tenant_id", COALESCE("target_node_id", '')) WHERE "active"`),
		ConflictDoNothing(),
	)
	if err != nil {
		t.Fatalf("upsert raw conflict target: %v", err)
	}
	if !strings.Contains(exec.query, `ON CONFLICT ("tenant_id", COALESCE("target_node_id", '')) WHERE "active" DO NOTHING`) {
		t.Fatalf("expected raw expression conflict target, got: %s", exec.query)
	}
}

func TestInsertOnceReturningInsertedRow(t *testing.T) {
	db, mock := newReturningMockDB(t)
	mock.ExpectQuery(`INSERT INTO "users".*ON CONFLICT \("id"\) DO NOTHING RETURNING "id", "name", "age"$`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "age"}).AddRow(5, "alice", 32))

	row, inserted, err := InsertOnceReturning[genericWriteUser](
		context.Background(),
		db,
		genericWriteUser{ID: 5, Name: "alice", Age: 32},
		WherePK(),
	)
	if err != nil {
		t.Fatalf("insert once returning: %v", err)
	}
	if !inserted {
		t.Fatal("expected inserted=true")
	}
	if row.ID != 5 || row.Name != "alice" || row.Age != 32 {
		t.Fatalf("unexpected row: %+v", row)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestInsertOnceReturningExistingRow(t *testing.T) {
	db, mock := newReturningMockDB(t)
	mock.ExpectQuery(`INSERT INTO "users".*ON CONFLICT \("id"\) DO NOTHING RETURNING "id", "name", "age"$`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "age"}))
	mock.ExpectQuery(`SELECT "id", "name", "age" FROM "users" WHERE "id" = \$1`).
		WithArgs(int64(5)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "age"}).AddRow(5, "existing", 40))

	row, inserted, err := InsertOnceReturning[genericWriteUser](
		context.Background(),
		db,
		genericWriteUser{ID: 5, Name: "alice", Age: 32},
		WherePK(),
	)
	if err != nil {
		t.Fatalf("insert once returning existing: %v", err)
	}
	if inserted {
		t.Fatal("expected inserted=false")
	}
	if row.ID != 5 || row.Name != "existing" || row.Age != 40 {
		t.Fatalf("unexpected row: %+v", row)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestUpsertUpdateColumnsRequireInsertedColumn(t *testing.T) {
	db, _ := newCaptureWriteDB(driver.PostgresDialect{})

	_, err := Upsert(
		context.Background(),
		db,
		map[string]any{
			"tenant_id": "tenant-1",
			"field_key": "weekly_hours",
		},
		Table("form_fields"),
		ConflictColumns("tenant_id", "field_key"),
		UpdateColumns("value_text"),
	)
	if err == nil || !strings.Contains(err.Error(), "UpdateColumns requires inserted column value_text") {
		t.Fatalf("expected missing update column error, got: %v", err)
	}
}

func TestUpsertPostgresConflictConstraint(t *testing.T) {
	db, exec := newCaptureWriteDB(driver.PostgresDialect{})

	_, err := Upsert(
		context.Background(),
		db,
		map[string]any{"name": "alice", "age": 30},
		Table("users"),
		ConflictConstraint("users_name_key"),
	)
	if err != nil {
		t.Fatalf("upsert conflict constraint: %v", err)
	}
	if !strings.Contains(exec.query, `ON CONFLICT ON CONSTRAINT "users_name_key" DO UPDATE SET`) {
		t.Fatalf("expected named constraint conflict target, got: %s", exec.query)
	}
}
