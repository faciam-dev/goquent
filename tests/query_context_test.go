package tests

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

func TestQueryWithContext(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	ctx := context.Background()
	var row map[string]any
	if err := db.Table("users").WithContext(ctx).Where("id", 1).FirstMap(&row); err != nil {
		t.Fatalf("first map with context: %v", err)
	}
	if row["name"] != "alice" {
		t.Errorf("expected alice, got %v", row["name"])
	}
}

func TestQueryWithCanceledContext(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var row map[string]any
	if err := db.Table("users").WithContext(ctx).Where("id", 1).FirstMap(&row); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled context, got %v", err)
	}
}

func TestInsertWithContext(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	ctx := context.Background()
	_, err := db.Table("users").WithContext(ctx).Insert(map[string]any{"name": "ctx_insert", "age": 20})
	if err != nil {
		t.Fatalf("insert with context: %v", err)
	}
	var row map[string]any
	if err := db.Table("users").Where("name", "ctx_insert").FirstMap(&row); err != nil {
		t.Fatalf("select inserted: %v", err)
	}
	if row["age"] != int64(20) {
		t.Errorf("expected age 20, got %v", row["age"])
	}
}

func TestInsertWithCanceledContext(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := db.Table("users").WithContext(ctx).Insert(map[string]any{"name": "ctx_cancel", "age": 99})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled context, got %v", err)
	}
}

func TestUpdateWithContext(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	ctx := context.Background()
	_, err := db.Table("users").WithContext(ctx).Where("name", "alice").Update(map[string]any{"age": 31})
	if err != nil {
		t.Fatalf("update with context: %v", err)
	}
	var row map[string]any
	if err := db.Table("users").Where("name", "alice").FirstMap(&row); err != nil {
		t.Fatalf("select after update: %v", err)
	}
	if row["age"] != int64(31) {
		t.Errorf("expected age 31, got %v", row["age"])
	}
}

func TestUpdateWithCanceledContext(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := db.Table("users").WithContext(ctx).Where("name", "alice").Update(map[string]any{"age": 40})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled context, got %v", err)
	}
}

func TestDeleteWithContext(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	ctx := context.Background()
	_, err := db.Table("users").WithContext(ctx).Where("name", "bob").Delete()
	if err != nil {
		t.Fatalf("delete with context: %v", err)
	}
	var row map[string]any
	err = db.Table("users").Where("name", "bob").FirstMap(&row)
	if err != sql.ErrNoRows {
		t.Fatalf("expected ErrNoRows, got %v", err)
	}
}

func TestDeleteWithCanceledContext(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := db.Table("users").WithContext(ctx).Where("name", "alice").Delete()
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled context, got %v", err)
	}
}
