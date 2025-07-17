package tests

import (
	"database/sql"
	"net"
	"testing"

	"github.com/faciam-dev/goquent/orm"
	mysql "github.com/go-sql-driver/mysql"
)

func setupCustomDB(t testing.TB, drvName string) *orm.DB {
	dsn := "root:password@tcp(localhost:3306)/testdb?parseTime=true"
	db, err := orm.OpenWithDriver(drvName, dsn)
	if err != nil {
		if _, ok := err.(*net.OpError); ok {
			t.Skip("mysql not available")
		}
		t.Fatalf("open: %v", err)
	}
	stdDB, _ := sql.Open("mysql", dsn)
	_, err = stdDB.Exec(`CREATE TABLE IF NOT EXISTS users (
        id INT AUTO_INCREMENT PRIMARY KEY,
        name VARCHAR(64),
        age INT,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    )`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	_, err = stdDB.Exec("TRUNCATE TABLE users")
	if err != nil {
		t.Fatalf("truncate table: %v", err)
	}
	_, err = stdDB.Exec("INSERT INTO users(name, age) VALUES ('cdrv', 1)")
	if err != nil {
		t.Fatalf("insert users: %v", err)
	}
	return db
}

func TestOpenWithRegisteredDriver(t *testing.T) {
	orm.RegisterDriver("mysql-custom", &mysql.MySQLDriver{})
	db := setupCustomDB(t, "mysql-custom")
	defer db.Close()

	var row map[string]any
	if err := db.Table("users").Where("name", "cdrv").FirstMap(&row); err != nil {
		t.Fatalf("select: %v", err)
	}
	if row["age"] != int64(1) {
		t.Errorf("expected age 1, got %v", row["age"])
	}
}
