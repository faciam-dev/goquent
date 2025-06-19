package tests

import (
	"strings"
	"testing"
)

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

func TestWhereMonthDayYearTime(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	var row map[string]any
	if err := db.Table("users").WhereMonth("created_at", "=", "12").FirstMap(&row); err != nil {
		t.Fatalf("where month: %v", err)
	}
	if row["name"] != "alice" {
		t.Errorf("expected alice, got %v", row["name"])
	}

	if err := db.Table("users").WhereDay("created_at", "=", "31").FirstMap(&row); err != nil {
		t.Fatalf("where day: %v", err)
	}
	if row["name"] != "alice" {
		t.Errorf("expected alice, got %v", row["name"])
	}

	var rows []map[string]any
	if err := db.Table("users").WhereYear("created_at", "=", "2025").OrderBy("id", "asc").GetMaps(&rows); err != nil {
		t.Fatalf("where year: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(rows))
	}

	if err := db.Table("users").WhereTime("created_at", "=", "11:22:33").FirstMap(&row); err != nil {
		t.Fatalf("where time: %v", err)
	}
	if row["name"] != "alice" {
		t.Errorf("expected alice, got %v", row["name"])
	}
}

func TestWhereColumn(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	var rows []map[string]any
	err := db.Table("users").Join("profiles", "users.id", "=", "profiles.user_id").
		WhereColumn("users.id", "profiles.user_id").
		OrderBy("users.id", "asc").GetMaps(&rows)
	if err != nil {
		t.Fatalf("where column: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(rows))
	}
}

func TestInsertOrIgnore(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	_, err := db.Table("users").InsertOrIgnore([]map[string]any{{"id": 1, "name": "dup", "age": 99}})
	if err != nil {
		t.Fatalf("insert or ignore: %v", err)
	}
	var rows []map[string]any
	if err := db.Table("users").Where("id", 1).GetMaps(&rows); err != nil {
		t.Fatalf("select: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("expected 1 row, got %d", len(rows))
	}
}

func TestUpsert(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	_, err := db.Table("users").Upsert([]map[string]any{{"id": 1, "name": "alice", "age": 50}}, []string{"id"}, []string{"age"})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	var row map[string]any
	if err := db.Table("users").Where("id", 1).FirstMap(&row); err != nil {
		t.Fatalf("select: %v", err)
	}
	if row["age"] != int64(50) {
		t.Errorf("expected age 50, got %v", row["age"])
	}
}

func TestWhereAnyAllSQL(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	sqlStr, err := db.Table("users").WhereAny([]string{"name", "age"}, "LIKE", "%a%").RawSQL()
	if err != nil {
		t.Fatalf("raw sql any: %v", err)
	}
	if !strings.Contains(sqlStr, "OR") {
		t.Errorf("expected OR in sql, got %s", sqlStr)
	}

	sqlStr, err = db.Table("users").WhereAll([]string{"name", "age"}, "LIKE", "%a%").RawSQL()
	if err != nil {
		t.Fatalf("raw sql all: %v", err)
	}
	if !strings.Contains(sqlStr, "AND") {
		t.Errorf("expected AND in sql, got %s", sqlStr)
	}
}
