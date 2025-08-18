package tests

import (
	"database/sql"
	"net"
	"testing"

	"github.com/faciam-dev/goquent/orm"
	_ "github.com/lib/pq"
)

type PgUser struct {
	ID   int    `orm:"column=id,primaryKey"`
	Name string `orm:"column=name"`
	Age  int    `orm:"column=age"`
}

func setupPgDB(t testing.TB) *orm.DB {
	dsn := "postgres://postgres:password@localhost/testdb?sslmode=disable"
	db, err := orm.OpenWithDriver(orm.Postgres, dsn)
	if err != nil {
		if _, ok := err.(*net.OpError); ok {
			t.Skip("postgres not available")
		}
		t.Fatalf("open: %v", err)
	}
	stdDB, _ := sql.Open("postgres", dsn)
	_, err = stdDB.Exec(`CREATE TABLE IF NOT EXISTS users (
        id SERIAL PRIMARY KEY,
        name TEXT,
        age INT,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    )`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	_, err = stdDB.Exec("TRUNCATE TABLE users")
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return db
}

func TestPostgresInsertSelect(t *testing.T) {
	db := setupPgDB(t)
	defer db.Close()
	if _, err := db.Table("users").Insert(map[string]any{"name": "pg", "age": 10}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	var row map[string]any
	if err := db.Table("users").Where("name", "pg").FirstMap(&row); err != nil {
		t.Fatalf("select: %v", err)
	}
	if row["age"] != int64(10) {
		t.Errorf("expected age 10, got %v", row["age"])
	}
}

func TestPostgresInsertGetId(t *testing.T) {
	db := setupPgDB(t)
	defer db.Close()
	id, err := db.Table("users").InsertGetId(map[string]any{"name": "pg2", "age": 11})
	if err != nil {
		t.Fatalf("insert get id: %v", err)
	}
	if id != 1 {
		t.Errorf("expected id 1, got %d", id)
	}
}

func TestPostgresInsertGetIdCustomPrimaryKey(t *testing.T) {
	db := setupPgDB(t)
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS items (
               item_id SERIAL PRIMARY KEY,
               name TEXT
       )`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	defer db.Exec("DROP TABLE items")
	id, err := db.Table("items").PrimaryKey("item_id").InsertGetId(map[string]any{"name": "foo"})
	if err != nil {
		t.Fatalf("insert get id: %v", err)
	}
	if id != 1 {
		t.Errorf("expected id 1, got %d", id)
	}
}
