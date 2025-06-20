package tests

import (
	"context"
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
}
