package orm

import (
	"context"
	"database/sql"
	sqldriver "database/sql/driver"
	"fmt"
	"strings"
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
	drv      *driver.Driver
	exec     executor
	scanOpts ScanOptions
}

// Option configures DB at creation.
type Option func(*DB)

// WithBoolScanPolicy sets the bool scanning policy.
func WithBoolScanPolicy(p BoolScanPolicy) Option {
	return func(db *DB) { db.scanOpts.BoolPolicy = p }
}

// SQLDB returns the underlying *sql.DB.
func (db *DB) SQLDB() *sql.DB {
	if db.drv == nil {
		return nil
	}
	return db.drv.DB
}

// Database driver names.
const (
	MySQL    = "mysql"
	Postgres = "postgres"
)

func defaultDialect(name string) driver.Dialect {
	if strings.Contains(strings.ToLower(name), "postgres") {
		return driver.PostgresDialect{}
	}
	return driver.MySQLDialect{}
}

func newDB(d *driver.Driver, exec executor, opts ...Option) *DB {
	db := &DB{drv: d, exec: exec, scanOpts: ScanOptions{BoolPolicy: BoolCompat}}
	for _, o := range opts {
		o(db)
	}
	return db
}

// NewDB wraps an existing sql.DB with a dialect into DB.
func NewDB(sqlDB *sql.DB, dialect driver.Dialect, opts ...Option) *DB {
	d := &driver.Driver{DB: sqlDB, Dialect: dialect}
	return newDB(d, sqlDB, opts...)
}

// Open opens a MySQL database with default pooling. Deprecated: use
// OpenWithDriver to specify a driver explicitly.
func Open(dsn string) (*DB, error) {
	return OpenWithDriverOptions(MySQL, dsn)
}

// OpenWithDriver opens a database with default pooling for the given driver.
func OpenWithDriver(driverName, dsn string) (*DB, error) {
	return OpenWithDriverOptions(driverName, dsn)
}

// OpenWithDriverOptions opens a database with options for the given driver.
func OpenWithDriverOptions(driverName, dsn string, opts ...Option) (*DB, error) {
	if drv, ok := GetDriver(driverName); ok {
		dc, ok := drv.(sqldriver.DriverContext)
		if !ok {
			return nil, fmt.Errorf("driver %q does not implement DriverContext", driverName)
		}
		connector, err := dc.OpenConnector(dsn)
		if err != nil {
			return nil, fmt.Errorf("failed to create connector: %w", err)
		}
		sqlDB := sql.OpenDB(connector)
		sqlDB.SetMaxOpenConns(10)
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetConnMaxLifetime(time.Hour)
		if err = sqlDB.Ping(); err != nil {
			return nil, err
		}
		dialect, ok := getDialect(driverName)
		if !ok {
			dialect = defaultDialect(driverName)
		}
		d := &driver.Driver{DB: sqlDB, Dialect: dialect}
		return newDB(d, sqlDB, opts...), nil
	}

	drv, err := driver.Open(driverName, dsn, 10, 10, time.Hour)
	if err != nil {
		return nil, err
	}
	return newDB(drv, drv.DB, opts...), nil
}

// Close closes underlying DB.
func (db *DB) Close() error { return db.drv.Close() }

// newTransactionDB wraps a sql.Tx in a DB instance bound to the same driver.
func (db *DB) newTransactionDB(tx *sql.Tx) *DB {
	return &DB{drv: db.drv, exec: tx, scanOpts: db.scanOpts}
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

// TransactionContext executes fn in a transaction using ctx.
func (db *DB) TransactionContext(ctx context.Context, fn func(tx Tx) error) error {
	return db.drv.TransactionContext(ctx, func(t driver.Tx) error {
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

// BeginTx starts a transaction using ctx and returns the Tx.
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error) {
	t, err := db.drv.BeginTx(ctx, opts)
	if err != nil {
		return Tx{}, err
	}
	txDB := db.newTransactionDB(t.Tx)
	return Tx{DB: txDB, Tx: t}, nil
}

// Model creates a query for the struct table.
func (db *DB) Model(v any) *query.Query {
	return query.New(db.exec, model.TableName(v), db.drv.Dialect)
}

// Table creates a query for table name.
func (db *DB) Table(name string) *query.Query {
	return query.New(db.exec, name, db.drv.Dialect)
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
