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
	Query(query string, args ...any) (*sql.Rows, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	Exec(query string, args ...any) (sql.Result, error)
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

// Tx represents a transaction-scoped DB wrapper.
type Tx struct{ *DB }

// Transaction executes fn in a transaction.
func (db *DB) Transaction(fn func(tx Tx) error) error {
	return db.drv.Transaction(func(t driver.Tx) error {
		txDB := &DB{drv: db.drv, exec: t.Tx}
		return fn(Tx{txDB})
	})
}

// Model creates a query for the struct table.
func (db *DB) Model(v any) *query.Query {
	return query.New(db.exec, model.TableName(v))
}

// Table creates a query for table name.
func (db *DB) Table(name string) *query.Query {
	return query.New(db.exec, name)
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
