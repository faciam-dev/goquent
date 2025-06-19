package tests

import "testing"

func TestWhereNullAndNotNull(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	if _, err := db.Table("users").Insert(map[string]any{"name": "nulluser", "age": nil}); err != nil {
		t.Fatalf("insert null user: %v", err)
	}

	var row map[string]any
	if err := db.Table("users").WhereNull("age").FirstMap(&row); err != nil {
		t.Fatalf("where null: %v", err)
	}
	if row["name"] != "nulluser" {
		t.Errorf("expected nulluser, got %v", row["name"])
	}

	var rows []map[string]any
	if err := db.Table("users").WhereNotNull("age").OrderBy("id", "asc").GetMaps(&rows); err != nil {
		t.Fatalf("where not null: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(rows))
	}
}

func TestWhereBetween(t *testing.T) {
	db := setupDB(t)
	defer db.Close()
	var rows []map[string]any
	if err := db.Table("users").WhereBetween("age", 25, 30).OrderBy("age", "asc").GetMaps(&rows); err != nil {
		t.Fatalf("where between: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(rows))
	}
	if rows[0]["name"] != "bob" || rows[1]["name"] != "alice" {
		t.Errorf("unexpected rows: %v", rows)
	}
}

func TestWhereExists(t *testing.T) {
	db := setupDB(t)
	defer db.Close()
	sub := db.Table("profiles").SelectRaw("1").WhereRaw("profiles.user_id = users.id", map[string]any{})
	var rows []map[string]any
	if err := db.Table("users").WhereExists(sub).OrderBy("id", "asc").GetMaps(&rows); err != nil {
		t.Fatalf("where exists: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(rows))
	}
}

func TestUnionDistinct(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	// union identical queries should return distinct rows
	q1 := db.Table("users").Select("name")
	q2 := db.Table("users").Select("name")

	var rows []map[string]any
	if err := q1.Union(q2).OrderBy("name", "asc").GetMaps(&rows); err != nil {
		t.Fatalf("union distinct: %v", err)
	}
	if len(rows) != 2 || rows[0]["name"] != "alice" || rows[1]["name"] != "bob" {
		t.Errorf("unexpected union result: %v", rows)
	}
}

func TestUnionAllKeepsDuplicates(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	q1 := db.Table("users").Select("name")
	q2 := db.Table("users").Select("name")

	var rows []map[string]any
	if err := q1.UnionAll(q2).OrderBy("name", "asc").GetMaps(&rows); err != nil {
		t.Fatalf("union all: %v", err)
	}
	if len(rows) != 4 || rows[0]["name"] != "alice" || rows[1]["name"] != "alice" || rows[2]["name"] != "bob" || rows[3]["name"] != "bob" {
		t.Errorf("unexpected union all result: %v", rows)
	}
}
