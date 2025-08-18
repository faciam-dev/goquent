# goquent ORM
[![Docs](https://img.shields.io/badge/docs-API-blue.svg)](https://faciam-dev.github.io/goquent/)

This package provides a minimal ORM built on top of [goquent-query-builder](https://github.com/faciam-dev/goquent-query-builder).
It supports MySQL and PostgreSQL.

## Usage
```go
import (
       "github.com/faciam-dev/goquent/orm"
       "github.com/faciam-dev/goquent/orm/conv"
       "log"
)

db, _ := orm.OpenWithDriver(orm.MySQL, "root:password@tcp(localhost:3306)/testdb?parseTime=true")
// PostgreSQL example
// db, _ := orm.OpenWithDriver(orm.Postgres, "postgres://user:pass@localhost/testdb?sslmode=disable")
user := new(User)
err := db.Model(user).Where("id", 1).First(user)

var row map[string]any
err = db.Table("users").Where("id", 1).FirstMap(&row)

// fetch a typed value from a map
id, err := conv.Value[uint64](row, "id")
if err != nil {
    log.Fatal(err)
}

var rows []map[string]any
err = db.Table("users").Where("age", ">", 20).GetMaps(&rows)

var users []User
err = db.Model(&User{}).Where("age", ">", 20).Get(&users)

// insert a record and get its auto-increment id (uses RETURNING on PostgreSQL)
newID, err := db.Table("users").InsertGetId(map[string]any{"name": "sam", "age": 18})
if err != nil {
    log.Fatal(err)
}
```

Transactions are handled via `Transaction`:
```go
err := db.Transaction(func(tx orm.Tx) error {
    return tx.Table("users").Where("id", 1).First(&user)
})
```

Context-aware transactions are also available:
```go
ctx := context.Background()
err := db.TransactionContext(ctx, func(tx orm.Tx) error {
    return tx.Table("users").Where("id", 1).First(&user)
})
```

Manual transaction control is also available:
```go
ctx := context.Background()
tx, err := db.BeginTx(ctx, nil)
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

### Column comparisons
Values passed to `Where` are always treated as literals. To compare one column
against another, use `WhereColumn`:

```go
err := db.Table("profiles").
    WhereColumn("profiles.user_id", "users.id").
    Where("profiles.bio", "=", "go developer").
    FirstMap(&row)
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

## PostgreSQL Support
The driver now includes a `PostgresDialect`. Use `orm.OpenWithDriver(orm.Postgres, dsn)` with a valid PostgreSQL DSN to connect.

### Custom Drivers
Register a driver and optionally its SQL dialect so the ORM can infer quoting rules:

```go
orm.RegisterDriverWithDialect("mysql-custom", &mysql.MySQLDriver{}, driver.MySQLDialect{})
db, err := orm.OpenWithDriver("mysql-custom", dsn)
```

## License
This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
