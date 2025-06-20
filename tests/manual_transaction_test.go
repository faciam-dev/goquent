package tests

import (
	"database/sql"
	"testing"
)

func TestManualTxCommit(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if _, err := tx.Table("users").Insert(map[string]any{"name": "mallory", "age": 55}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	var row map[string]any
	if err := db.Table("users").Where("name", "mallory").FirstMap(&row); err != nil {
		t.Fatalf("select after commit: %v", err)
	}
	if row["age"] != int64(55) {
		t.Errorf("expected age 55, got %v", row["age"])
	}
}

func TestManualTxRollback(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if _, err := tx.Table("users").Insert(map[string]any{"name": "trent", "age": 44}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	var row map[string]any
	err = db.Table("users").Where("name", "trent").FirstMap(&row)
	if err != sql.ErrNoRows {
		t.Fatalf("expected no rows after rollback, got %v", err)
	}
}
