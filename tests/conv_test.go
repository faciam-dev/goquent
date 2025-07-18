package tests

import (
	"testing"

	"github.com/faciam-dev/goquent/orm/conv"
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

func TestMapToStruct(t *testing.T) {
	type user struct {
		ID        int    `orm:"column=id"`
		Name      string `orm:"column=user_name"`
		Age       int
		CreatedAt string
	}

	tests := []struct {
		name string
		m    map[string]any
		want user
	}{
		{
			name: "basic",
			m:    map[string]any{"id": 1, "user_name": "alice", "age": 20, "created_at": "2025"},
			want: user{ID: 1, Name: "alice", Age: 20, CreatedAt: "2025"},
		},
		{
			name: "dotted and quoted keys",
			m:    map[string]any{"users.id": 2, "users.`user_name`": "bob"},
			want: user{ID: 2, Name: "bob"},
		},
	}

	for _, tt := range tests {
		var u user
		if err := conv.MapToStruct(tt.m, &u); err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		if u != tt.want {
			t.Errorf("%s: expected %+v, got %+v", tt.name, tt.want, u)
		}
	}
}

func TestMapToStructErrors(t *testing.T) {
	type user struct{ ID int }
	var u user
	if err := conv.MapToStruct(nil, &u); err == nil {
		t.Error("expected error for nil map")
	}
	if err := conv.MapToStruct(map[string]any{"id": 1}, u); err == nil {
		t.Error("expected error for non-pointer dest")
	}
}

func TestMapsToStructs(t *testing.T) {
	type user struct {
		ID   int `orm:"column=id"`
		Name string
	}
	src := []map[string]any{{"id": 1, "name": "alice"}, {"id": 2, "name": "bob"}}
	var users []user
	if err := conv.MapsToStructs(src, &users); err != nil {
		t.Fatalf("maps to structs: %v", err)
	}
	if len(users) != 2 || users[0].Name != "alice" || users[1].ID != 2 {
		t.Errorf("unexpected result: %+v", users)
	}
}
