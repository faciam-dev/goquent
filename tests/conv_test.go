package tests

import (
	"testing"

	"goquent/orm/conv"
)

func TestValue(t *testing.T) {
	m := map[string]any{
		"id":   int64(1),
		"name": "bob",
	}
	id, err := conv.Value[uint64](m, "id")
	if err != nil {
		t.Fatalf("convert id: %v", err)
	}
	if id != 1 {
		t.Errorf("expected 1, got %d", id)
	}
	name, err := conv.Value[string](m, "name")
	if err != nil {
		t.Fatalf("convert name: %v", err)
	}
	if name != "bob" {
		t.Errorf("expected bob, got %s", name)
	}
}

func TestAs(t *testing.T) {
	v, err := conv.As[int](int64(5))
	if err != nil {
		t.Fatalf("as int: %v", err)
	}
	if v != 5 {
		t.Errorf("expected 5, got %d", v)
	}
}
