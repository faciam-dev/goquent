package driver

import (
	"database/sql"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// Dialect defines the SQL dialect abstraction.
type Dialect interface {
	Placeholder(n int) string
	QuoteIdent(ident string) string
}

// MySQLDialect implements Dialect for MySQL.
type MySQLDialect struct{}

func (d MySQLDialect) Placeholder(_ int) string { return "?" }

func (d MySQLDialect) QuoteIdent(ident string) string { return "`" + ident + "`" }

// Driver wraps sql.DB with a dialect.
type Driver struct {
	DB      *sql.DB
	Dialect Dialect
}

// Open initializes the DB connection with pooling configuration.
func Open(dsn string, maxOpen, maxIdle int, lifetime time.Duration) (*Driver, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(lifetime)
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return &Driver{DB: db, Dialect: MySQLDialect{}}, nil
}

// Close closes the underlying DB.
func (d *Driver) Close() error { return d.DB.Close() }

// Tx wraps sql.Tx for transaction handling.
type Tx struct{ *sql.Tx }

// Transaction executes fn within a transaction.
func (d *Driver) Transaction(fn func(Tx) error) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}
	if err = fn(Tx{tx}); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return rbErr
		}
		return err
	}
	return tx.Commit()
}
