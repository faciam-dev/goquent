package tests

import (
	"database/sql"
	"net"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"goquent/orm"
)

type User struct {
	ID   int    `orm:"column=id,primaryKey"`
	Name string `orm:"column=name"`
	Age  int    `orm:"column=age"`
}

func setupDB(t testing.TB) *orm.DB {
	dsn := "root:password@tcp(localhost:3306)/testdb?parseTime=true"
	db, err := orm.Open(dsn)
	if err != nil {
		if _, ok := err.(*net.OpError); ok {
			t.Skip("mysql not available")
		}
		t.Fatalf("open: %v", err)
	}
	stdDB, _ := sql.Open("mysql", dsn)
	_, err = stdDB.Exec(`CREATE TABLE IF NOT EXISTS users (id INT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(64), age INT)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	_, err = stdDB.Exec("TRUNCATE TABLE users")
	if err != nil {
		t.Fatalf("truncate table: %v", err)
	}
	_, err = stdDB.Exec("INSERT INTO users(name, age) VALUES ('alice', 30), ('bob', 25)")
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	return db
}

func TestFirstMap(t *testing.T) {
	db := setupDB(t)
	defer db.Close()
	var row map[string]any
	if err := db.Table("users").Where("id", 1).FirstMap(&row); err != nil {
		t.Fatalf("first map: %v", err)
	}
	if row["name"] != "alice" {
		t.Errorf("expected alice, got %v", row["name"])
	}
}

func BenchmarkScannerMap(b *testing.B) {
	db := setupDB(b)
	defer db.Close()
	for i := 0; i < b.N; i++ {
		var row map[string]any
		if err := db.Table("users").Where("id", 1).FirstMap(&row); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkScannerStruct(b *testing.B) {
	db := setupDB(b)
	defer db.Close()
	for i := 0; i < b.N; i++ {
		var user User
		if err := db.Model(&User{}).Where("id", 1).First(&user); err != nil {
			b.Fatal(err)
		}
	}
}

func TestInsert(t *testing.T) {
	db := setupDB(t)
	defer db.Close()
	if _, err := db.Table("users").Insert(map[string]any{"name": "charlie", "age": 40}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	var row map[string]any
	if err := db.Table("users").Where("name", "charlie").FirstMap(&row); err != nil {
		t.Fatalf("select: %v", err)
	}
	if row["age"] != int64(40) {
		t.Errorf("expected age 40, got %v", row["age"])
	}
}

func TestUpdate(t *testing.T) {
	db := setupDB(t)
	defer db.Close()
	if _, err := db.Table("users").Where("name", "bob").Update(map[string]any{"age": 35}); err != nil {
		t.Fatalf("update: %v", err)
	}
	var row map[string]any
	if err := db.Table("users").Where("name", "bob").FirstMap(&row); err != nil {
		t.Fatalf("select: %v", err)
	}
	if row["age"] != int64(35) {
		t.Errorf("expected age 35, got %v", row["age"])
	}
}
