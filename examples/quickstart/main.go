package main

import (
	"context"
	"log"

	"github.com/faciam-dev/goquent/orm"
)

type User struct {
	ID   int64
	Name string
	Age  int
}

func main() {
	// Scan existing users older than 20.
	db, err := orm.OpenWithDriver(orm.MySQL, "root:password@tcp(localhost:3306)/testdb?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var users []User
	if err := db.Model(&User{}).Where("age", ">", 20).Get(&users); err != nil {
		log.Fatal(err)
	}

	// Apply an update to mark them active.
	for _, u := range users {
		if _, err := db.Table("users").Where("id", u.ID).Update(map[string]any{"active": true}); err != nil {
			log.Fatal(err)
		}
	}

	// Migrate some data into archived_users.
	sub := db.Table("users").Select("id", "name")
	if _, err := db.Table("archived_users").InsertUsing([]string{"id", "name"}, sub); err != nil {
		log.Fatal(err)
	}

	// Optional transaction example.
	ctx := context.Background()
	if err := db.TransactionContext(ctx, func(tx orm.Tx) error {
		_, err := tx.Table("logs").Insert(map[string]any{"msg": "migrated"})
		return err
	}); err != nil {
		log.Fatal(err)
	}
}
