# goquent ORM

This package provides a minimal ORM built on top of [goquent-query-builder](https://github.com/faciam-dev/goquent-query-builder).
It currently supports MySQL only.

## Usage
```go
import "github.com/faciam-dev/goquent/orm/conv"
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

var users []User
err = orm.Model(&User{}).Where("age", ">", 20).Get(&users)

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

Manual transaction control is also available:
```go
tx, err := orm.Begin()
if err != nil {
    log.Fatal(err)
}
if _, err = tx.Table("users").Insert(map[string]any{"name": "sam"}); err != nil {
    tx.Rollback()
    log.Fatal(err)
}
if err = tx.Commit(); err != nil {
    log.Fatal(err)
}
```

## Project Structure
The repository follows the Onion Architecture:

```
./cmd/        - Entry points
./internal/   - Application code
  ├── domain        - Business logic
  ├── usecase       - Application workflows
  ├── infrastructure - External implementations
  └── interface     - HTTP handlers or adapters
```

The `orm` directory contains the lightweight ORM used by the project.

## Development
1. Start MySQL 8 using Docker Compose:
   ```bash
   docker-compose up -d
   ```
2. Run tests:
   ```bash
   go test ./...
   ```
The tests automatically create the required tables.


## Benchmarks
Run benchmarks with `go test -bench . ./tests`.
Results on a GitHub Codespace (Go 1.23) show ~1.5x speedup over GORM for scanning operations.

## Extending to PostgreSQL
The driver package is designed with dialect abstractions. Implementing a `postgresDialect` and adding support in `driver.Open` would enable PostgreSQL usage.

## License
This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
