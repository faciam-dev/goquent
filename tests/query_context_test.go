package tests

import (
	"context"
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
