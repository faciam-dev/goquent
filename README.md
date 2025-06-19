# goquent ORM

This package provides a minimal ORM built on top of [goquent-query-builder](https://github.com/faciam-dev/goquent-query-builder).
It currently supports MySQL only.

## Usage
```go
import "goquent/orm/conv"
import "log"

orm, _ := orm.Open("root:password@tcp(localhost:3306)/testdb?parseTime=true")
user := new(User)
err := orm.Model(user).Where("id", 1).First(user)

var row map[string]any
err = orm.Table("users").Where("id", 1).FirstMap(&row)

// fetch a typed value from a map
id, err := conv.Value[uint64](row, "id")
if err != nil {
    log.Fatal(err)
}

var rows []map[string]any
err = orm.Table("users").Where("age", ">", 20).GetMaps(&rows)

// insert a record and get its auto-increment id
newID, err := orm.Table("users").InsertGetId(map[string]any{"name": "sam", "age": 18})
if err != nil {
    log.Fatal(err)
}
```

Transactions are handled via `Transaction`:
```go
err := orm.Transaction(func(tx orm.Tx) error {
    return tx.Table("users").Where("id", 1).First(&user)
})
```

## Benchmarks
Run benchmarks with `go test -bench . ./tests`.
Results on a GitHub Codespace (Go 1.23) show ~1.5x speedup over GORM for scanning operations.

## Extending to PostgreSQL
The driver package is designed with dialect abstractions. Implementing a `postgresDialect` and adding support in `driver.Open` would enable PostgreSQL usage.
