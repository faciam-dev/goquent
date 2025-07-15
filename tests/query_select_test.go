package tests

import "testing"

func TestSelectColumns(t *testing.T) {
	db := setupDB(t)
	defer db.Close()
	var row map[string]any
	if err := db.Table("users").Select("name").Where("id", 1).FirstMap(&row); err != nil {
		t.Fatalf("select columns: %v", err)
	}
	if len(row) != 1 {
		t.Errorf("expected 1 column, got %d", len(row))
	}
	if row["name"] != "alice" {
		t.Errorf("expected alice, got %v", row["name"])
	}
}

func TestDistinct(t *testing.T) {
	db := setupDB(t)
	defer db.Close()
	_, err := db.Table("users").Insert(map[string]any{"name": "carol", "age": 30})
	if err != nil {
		t.Fatalf("insert duplicate: %v", err)
	}
	var rows []map[string]any
	if err := db.Table("users").Distinct("age").OrderBy("age", "asc").GetMaps(&rows); err != nil {
		t.Fatalf("distinct: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(rows))
	}
	if rows[0]["age"] != int64(25) || rows[1]["age"] != int64(30) {
		t.Errorf("unexpected ages: %v", rows)
	}
}

func TestLimitOffset(t *testing.T) {
	db := setupDB(t)
	defer db.Close()
	var rows []map[string]any
	if err := db.Table("users").OrderBy("id", "asc").Offset(1).Limit(1).GetMaps(&rows); err != nil {
		t.Fatalf("limit offset: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("expected 1 row, got %d", len(rows))
	}
	if rows[0]["name"] != "bob" {
		t.Errorf("expected bob, got %v", rows[0]["name"])
	}
}

func TestJoinSelect(t *testing.T) {
	db := setupDB(t)
	defer db.Close()
	var row map[string]any
	err := db.Table("users").Join("profiles", "users.id", "=", "profiles.user_id").Select("users.name", "profiles.bio").Where("profiles.bio", "like", "%go%").FirstMap(&row)
	if err != nil {
		t.Fatalf("join select: %v", err)
	}
	if row["name"] != "alice" || row["bio"] != "go developer" {
		t.Errorf("unexpected row: %v", row)
	}
}

func TestCount(t *testing.T) {
	db := setupDB(t)
	defer db.Close()
	c, err := db.Table("users").Count()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if c != 2 {
		t.Errorf("expected count 2, got %d", c)
	}
}
