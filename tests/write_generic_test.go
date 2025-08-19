package tests

import (
	"context"
	"testing"

	"github.com/faciam-dev/goquent/orm"
)

func TestInsertStructGeneric(t *testing.T) {
	db := setupDB(t)
	defer db.Close()
	ctx := context.Background()
	u := User{Name: "carol", Age: 33}
	if _, err := orm.Insert(ctx, db, u); err != nil {
		t.Fatalf("insert struct: %v", err)
	}
	var cnt int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE name = ?", u.Name).Scan(&cnt); err != nil {
		t.Fatalf("count: %v", err)
	}
	if cnt != 1 {
		t.Errorf("expected 1, got %d", cnt)
	}
}

func TestInsertMapGeneric(t *testing.T) {
	db := setupDB(t)
	defer db.Close()
	ctx := context.Background()
	m := map[string]any{"name": "mapg", "age": 25}
	if _, err := orm.Insert(ctx, db, m, orm.Table("users")); err != nil {
		t.Fatalf("insert map: %v", err)
	}
	var cnt int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE name = ?", "mapg").Scan(&cnt); err != nil {
		t.Fatalf("count: %v", err)
	}
	if cnt != 1 {
		t.Errorf("expected 1, got %d", cnt)
	}
}

func TestUpdateStructWherePK(t *testing.T) {
	db := setupDB(t)
	defer db.Close()
	ctx := context.Background()
	u := User{ID: 1, Name: "alice2"}
	if _, err := orm.Update(ctx, db, u, orm.Columns("name"), orm.WherePK()); err != nil {
		t.Fatalf("update: %v", err)
	}
	var name string
	if err := db.QueryRowContext(ctx, "SELECT name FROM users WHERE id = 1").Scan(&name); err != nil {
		t.Fatalf("select: %v", err)
	}
	if name != "alice2" {
		t.Errorf("expected alice2, got %s", name)
	}
}

func TestUpdateStructNoWherePK(t *testing.T) {
	db := setupDB(t)
	defer db.Close()
	ctx := context.Background()
	u := User{ID: 1, Name: "alice3"}
	if _, err := orm.Update(ctx, db, u, orm.Columns("name")); err == nil {
		t.Fatalf("expected error without WherePK")
	}
}

func TestUpsertStruct(t *testing.T) {
	db := setupDB(t)
	defer db.Close()
	ctx := context.Background()
	// update existing
	u := User{ID: 2, Name: "bob2"}
	if _, err := orm.Upsert(ctx, db, u, orm.WherePK()); err != nil {
		t.Fatalf("upsert update: %v", err)
	}
	var name string
	if err := db.QueryRowContext(ctx, "SELECT name FROM users WHERE id = 2").Scan(&name); err != nil {
		t.Fatalf("select: %v", err)
	}
	if name != "bob2" {
		t.Errorf("expected bob2, got %s", name)
	}
	// insert new
	u2 := User{ID: 10, Name: "newg"}
	if _, err := orm.Upsert(ctx, db, u2, orm.WherePK()); err != nil {
		t.Fatalf("upsert insert: %v", err)
	}
	if err := db.QueryRowContext(ctx, "SELECT name FROM users WHERE id = 10").Scan(&name); err != nil {
		t.Fatalf("select2: %v", err)
	}
	if name != "newg" {
		t.Errorf("expected newg, got %s", name)
	}
}

func TestUpdateMapWherePK(t *testing.T) {
	db := setupDB(t)
	defer db.Close()
	ctx := context.Background()
	m := map[string]any{"id": 1, "name": "alice4"}
	if _, err := orm.Update(ctx, db, m, orm.Table("users"), orm.PK("id"), orm.Columns("name"), orm.WherePK()); err != nil {
		t.Fatalf("update map: %v", err)
	}
	var name string
	if err := db.QueryRowContext(ctx, "SELECT name FROM users WHERE id = 1").Scan(&name); err != nil {
		t.Fatalf("select: %v", err)
	}
	if name != "alice4" {
		t.Errorf("expected alice4, got %s", name)
	}
}

func TestUpdateMapMissingPKOpt(t *testing.T) {
	db := setupDB(t)
	defer db.Close()
	ctx := context.Background()
	m := map[string]any{"id": 1, "name": "alice5"}
	if _, err := orm.Update(ctx, db, m, orm.Table("users"), orm.WherePK()); err == nil {
		t.Fatalf("expected error without PK option")
	}
}

func TestUpsertMap(t *testing.T) {
	db := setupDB(t)
	defer db.Close()
	ctx := context.Background()
	// update existing
	m := map[string]any{"id": 2, "name": "bob3"}
	if _, err := orm.Upsert(ctx, db, m, orm.Table("users"), orm.PK("id"), orm.WherePK()); err != nil {
		t.Fatalf("upsert update: %v", err)
	}
	var name string
	if err := db.QueryRowContext(ctx, "SELECT name FROM users WHERE id = 2").Scan(&name); err != nil {
		t.Fatalf("select: %v", err)
	}
	if name != "bob3" {
		t.Errorf("expected bob3, got %s", name)
	}
	// insert new
	m2 := map[string]any{"id": 11, "name": "mapnew"}
	if _, err := orm.Upsert(ctx, db, m2, orm.Table("users"), orm.PK("id"), orm.WherePK()); err != nil {
		t.Fatalf("upsert insert: %v", err)
	}
	if err := db.QueryRowContext(ctx, "SELECT name FROM users WHERE id = 11").Scan(&name); err != nil {
		t.Fatalf("select2: %v", err)
	}
	if name != "mapnew" {
		t.Errorf("expected mapnew, got %s", name)
	}
}

func TestUpsertStructNoUpdate(t *testing.T) {
	db := setupDB(t)
	defer db.Close()
	ctx := context.Background()
	u := User{ID: 2}
	if _, err := orm.Upsert(ctx, db, u, orm.WherePK()); err != nil {
		t.Fatalf("upsert no update: %v", err)
	}
	var name string
	if err := db.QueryRowContext(ctx, "SELECT name FROM users WHERE id = 2").Scan(&name); err != nil {
		t.Fatalf("select: %v", err)
	}
	if name != "bob" {
		t.Errorf("expected bob, got %s", name)
	}
}
