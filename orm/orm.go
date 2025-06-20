package orm

import (
	"context"
	"database/sql"
	"time"

	"github.com/faciam-dev/goquent/orm/driver"
	"github.com/faciam-dev/goquent/orm/model"
	"github.com/faciam-dev/goquent/orm/query"
)

// executor abstracts sql.DB and sql.Tx.
type executor interface {
	// Query runs a SQL statement returning multiple rows.
	Query(query string, args ...any) (*sql.Rows, error)
	// QueryContext is the context-aware version of Query.
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	// QueryRow executes a query expected to return at most one row.
	QueryRow(query string, args ...any) *sql.Row
	// QueryRowContext executes a single-row query with context.
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	// Exec runs a SQL statement that doesn't return rows.
	Exec(query string, args ...any) (sql.Result, error)
	// ExecContext runs Exec with a context.
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// DB provides main ORM interface.
type DB struct {
	drv  *driver.Driver
	exec executor
}

// Open opens a MySQL database with default pooling.
func Open(dsn string) (*DB, error) {
	drv, err := driver.Open(dsn, 10, 10, time.Hour)
	if err != nil {
		return nil, err
	}
	return &DB{drv: drv, exec: drv.DB}, nil
}

// Close closes underlying DB.
func (db *DB) Close() error { return db.drv.Close() }

// newTransactionDB wraps a sql.Tx in a DB instance bound to the same driver.
func (db *DB) newTransactionDB(tx *sql.Tx) *DB {
	return &DB{drv: db.drv, exec: tx}
}

// Tx represents a transaction-scoped DB wrapper.
type Tx struct {
	*DB
	driver.Tx
}

// Transaction executes fn in a transaction.
func (db *DB) Transaction(fn func(tx Tx) error) error {
	return db.drv.Transaction(func(t driver.Tx) error {
		txDB := db.newTransactionDB(t.Tx)
		return fn(Tx{DB: txDB, Tx: t})
	})
}

// Begin starts a transaction for manual control.
func (db *DB) Begin() (Tx, error) {
	t, err := db.drv.Begin()
	if err != nil {
		return Tx{}, err
	}
	txDB := db.newTransactionDB(t.Tx)
	return Tx{DB: txDB, Tx: t}, nil
}

// Model creates a query for the struct table.
func (db *DB) Model(v any) *query.Query {
	return query.New(db.exec, model.TableName(v))
}

// Table creates a query for table name.
func (db *DB) Table(name string) *query.Query {
	return query.New(db.exec, name)
}

// Query runs a raw SQL query returning multiple rows.
func (db *DB) Query(q string, args ...any) (*sql.Rows, error) {
	return db.exec.Query(q, args...)
}

// QueryContext runs Query with a context.
func (db *DB) QueryContext(ctx context.Context, q string, args ...any) (*sql.Rows, error) {
	return db.exec.QueryContext(ctx, q, args...)
}

// Exec executes a raw SQL statement.
func (db *DB) Exec(query string, args ...any) (sql.Result, error) {
	return db.exec.Exec(query, args...)
}

// ExecContext executes a raw SQL statement with a context.
func (db *DB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return db.exec.ExecContext(ctx, query, args...)
}

// QueryRow executes a query that is expected to return at most one row.
func (db *DB) QueryRow(query string, args ...any) *sql.Row {
	return db.exec.QueryRow(query, args...)
}

// QueryRowContext executes a query with context returning at most one row.
func (db *DB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return db.exec.QueryRowContext(ctx, query, args...)
}
