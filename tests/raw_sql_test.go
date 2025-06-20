package tests

import (
	"context"
	"errors"
	"testing"
)

func TestExecAndQueryRow(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	_, err := db.Exec("INSERT INTO users(name, age) VALUES(?, ?)", "greg", 55)
	if err != nil {
		t.Fatalf("exec raw: %v", err)
	}

	var name string
	if err := db.QueryRow("SELECT name FROM users WHERE name=?", "greg").Scan(&name); err != nil {
		t.Fatalf("queryrow: %v", err)
	}
	if name != "greg" {
		t.Errorf("expected greg, got %s", name)
	}

	rows, err := db.Query("SELECT name FROM users ORDER BY id ASC")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()
	var count int
	for rows.Next() {
		count++
	}
	if count == 0 {
		t.Error("expected rows from Query")
	}
}

func TestExecAndQueryRowContext(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	ctx := context.Background()
	_, err := db.ExecContext(ctx, "UPDATE users SET age=? WHERE name=?", 31, "alice")
	if err != nil {
		t.Fatalf("exec context: %v", err)
	}

	var age int
	if err := db.QueryRowContext(ctx, "SELECT age FROM users WHERE name=?", "alice").Scan(&age); err != nil {
		t.Fatalf("queryrow context: %v", err)
	}
	if age != 31 {
		t.Errorf("expected 31, got %d", age)
	}

	rows, err := db.QueryContext(ctx, "SELECT id FROM users")
	if err != nil {
		t.Fatalf("query context: %v", err)
	}
	rows.Close()
}

func TestCanceledContextErrors(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := db.ExecContext(ctx, "INSERT INTO users(name, age) VALUES('x',1)"); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled exec, got %v", err)
	}

	if _, err := db.QueryContext(ctx, "SELECT 1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled query, got %v", err)
	}

	if err := db.QueryRowContext(ctx, "SELECT 1").Scan(new(int)); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled queryrow, got %v", err)
	}
}
