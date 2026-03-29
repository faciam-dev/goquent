# Generic CRUD

The generic API is the small set of helpers around:

- `SelectOne[T]`
- `SelectAll[T]`
- `Insert[T]`
- `Update[T]`
- `Upsert[T]`

Use these helpers when you want a compact typed call and the operation is simple enough to fit their model. Use the query-builder API when you want to build SQL fluently with `db.Model(...).Where(...).Get(...)`, `db.Table(...).Where(...).FirstMap(...)`, joins, non-primary-key updates, or other builder features.

The snippets below assume a setup like this:

```go
type User struct {
    ID     int64  `db:"id,pk"`
    Name   string `db:"name"`
    Age    int    `db:"age"`
    Active bool   `db:"active"`
}

ctx := context.Background()
db, err := orm.OpenWithDriver(orm.MySQL, dsn)
```

## Overview

The read side of the generic API executes raw SQL that you provide and scans the result into a concrete Go type:

```go
user, err := orm.SelectOne[User](ctx, db, "SELECT id, name, age FROM users WHERE id = ?", 1)
rows, err := orm.SelectAll[map[string]any](ctx, db, "SELECT id, name FROM users ORDER BY id")
```

The query-builder API does the SQL construction for you:

```go
var user User
err := db.Model(&User{}).Where("id", 1).First(&user)

var row map[string]any
err = db.Table("users").Where("id", 1).FirstMap(&row)
```

The write side of the generic API builds simple `INSERT`, `UPDATE`, and `UPSERT` statements from a struct value or `map[string]any`:

```go
_, err := orm.Insert(ctx, db, User{Name: "sam", Age: 18})
_, err = orm.Update(ctx, db, User{ID: 1, Name: "sam"}, orm.Columns("name"), orm.WherePK())
_, err = orm.Upsert(ctx, db, User{ID: 1, Name: "sam", Age: 18}, orm.WherePK())
```

For anything more complex than "write this row to this table", prefer `db.Table(...).Where(...).Update(...)` or raw SQL.

## Read API

### `SelectOne[T]`

`SelectOne[T]` runs a query and scans the first row into `T`.

```go
u, err := orm.SelectOne[User](ctx, db, "SELECT id, name, age FROM users WHERE id = ?", 1)
```

If the query returns no rows, `SelectOne` returns `sql.ErrNoRows`.

```go
u, err := orm.SelectOne[User](ctx, db, "SELECT id, name, age FROM users WHERE id = ?", id)
if errors.Is(err, sql.ErrNoRows) {
    return
}
if err != nil {
    log.Fatal(err)
}
_ = u
```

### `SelectAll[T]`

`SelectAll[T]` runs a query and scans all rows into `[]T`.

```go
users, err := orm.SelectAll[User](ctx, db, "SELECT id, name, age FROM users ORDER BY id")
```

If the query returns no rows, `SelectAll` returns an empty slice and a `nil` error.

### Supported `T` shapes

The current implementation supports these destination shapes:

- A non-pointer struct type such as `User`
- Exactly `map[string]any`

The current implementation does not support, and this guide does not guarantee, these shapes:

- Pointer destinations such as `*User`
- Scalar destinations such as `int64`, `string`, or `bool`
- Slice types as `T`
- Other map shapes such as `map[string]string`

### Practical column matching

For struct destinations:

- `db:"column_name"` sets the column name explicitly.
- Without a tag, goquent uses the field name converted to `snake_case`.
- Matching first tries the exact column name.
- If that does not match, it tries a normalized match that lowercases the name and removes underscores.

In practice, these columns all match a field tagged or inferred as `schema_name`:

- `schema_name`
- `SchemaName`
- `SCHEMA_NAME`

Columns with no matching field are ignored. Fields with no matching column keep their zero value.

Struct field decoding is reflection-based. A field must either:

- be directly assignable or convertible from the driver value,
- implement `sql.Scanner` via its pointer type, or
- be `bool`, `sql.NullBool`, or `*bool`, which use goquent's bool scan policy.

For map destinations:

- keys are the column names returned by the database,
- values are stored as `any`,
- `[]byte` values are converted to `string`.

## Write API

### Supported input shapes

`Insert`, `Update`, and `Upsert` currently accept:

- A non-pointer struct value
- `map[string]any`

Pointer values such as `*User` are not supported by the current implementation.

### Struct-based writes

Struct-based writes use reflection metadata from the struct:

- The table name comes from `TableName() string` when the struct value implements it, otherwise from the struct type name in `snake_case` plus `s`.
- Column names come from the `db` tag or from the field name in `snake_case`.
- `db:"...,pk"` marks primary-key fields for `WherePK()`.
- `db:"...,readonly"` excludes a field from writes.
- `db:"...,omitempty"` skips zero values on insert, update, and upsert.

Example struct:

```go
type User struct {
    ID     int64  `db:"id,pk"`
    Name   string `db:"name"`
    Age    int    `db:"age"`
    Active bool   `db:"active"`
}
```

### Map-based writes

Map-based writes use the map keys as column names exactly as given. There is no table-name inference and no primary-key inference.

That means:

- `Table("...")` is required for all map writes.
- `PK("...")` is required for map `Update` and `Upsert` when you also use `WherePK()`.
- The map must include values for every column listed in `PK(...)`.

### `Insert[T]`

`Insert` builds a single-row `INSERT`.

Struct example:

```go
_, err := orm.Insert(ctx, db, User{Name: "sam", Age: 18, Active: true})
```

Map example:

```go
_, err := orm.Insert(ctx, db, map[string]any{
    "name":   "sam",
    "age":    18,
    "active": true,
}, orm.Table("users"))
```

### `Update[T]`

`Update` only works with `WherePK()`. There is no generic helper for arbitrary `WHERE` clauses.

For structs, `WherePK()` uses fields tagged with `db:"...,pk"`. If the struct has no primary-key metadata, `Update` returns an error.

For maps, `WherePK()` uses the columns named by `PK(...)`.

```go
_, err := orm.Update(
    ctx,
    db,
    User{ID: 1, Name: "alice", Active: true},
    orm.Columns("name", "active"),
    orm.WherePK(),
)
```

### `Upsert[T]`

`Upsert` also requires `WherePK()`.

- On MySQL it builds `INSERT ... ON DUPLICATE KEY UPDATE ...`.
- On PostgreSQL it builds `INSERT ... ON CONFLICT (...) DO UPDATE ...`.

If there are no non-primary-key columns left to update after filtering, the helper falls back to a no-op conflict action:

- MySQL: `INSERT IGNORE`
- PostgreSQL: `ON CONFLICT (...) DO NOTHING`

```go
_, err := orm.Upsert(
    ctx,
    db,
    User{ID: 1, Name: "alice", Age: 31},
    orm.WherePK(),
)
```

## Write options

### `Columns(...)`

`Columns` keeps only the listed columns.

```go
_, err := orm.Update(
    ctx,
    db,
    User{ID: 1, Name: "alice", Active: true},
    orm.Columns("name"),
    orm.WherePK(),
)
```

### `Omit(...)`

`Omit` removes columns from the write set.

```go
_, err := orm.Insert(
    ctx,
    db,
    User{Name: "sam", Age: 18, Active: true},
    orm.Omit("active"),
)
```

If you use both `Columns(...)` and `Omit(...)`, `Omit(...)` still removes the omitted columns.

### `WherePK()`

`WherePK()` is required for `Update` and `Upsert`.

- Struct writes: use fields tagged with `db:"...,pk"`.
- Map writes: use `PK(...)`.
- In practice you will usually combine it with `Columns(...)` or `Omit(...)` on struct updates.

```go
_, err := orm.Update(ctx, db, User{ID: 1, Name: "alice"}, orm.Columns("name"), orm.WherePK())
```

### `Returning(...)`

`Returning` appends a `RETURNING` clause only when the active dialect is PostgreSQL.

```go
_, err := orm.Update(
    ctx,
    db,
    User{ID: 1, Name: "alice"},
    orm.Columns("name"),
    orm.WherePK(),
    orm.Returning("id", "name"),
)
```

These helpers still return `sql.Result`. They do not expose returned row values directly.

### `Table(...)`

`Table` overrides the inferred table name for struct writes and is required for map writes.

```go
_, err := orm.Insert(
    ctx,
    db,
    map[string]any{"name": "sam"},
    orm.Table("users"),
)
```

### `PK(...)`

`PK` names the primary-key columns for map `Update` and `Upsert` when combined with `WherePK()`.

```go
_, err := orm.Update(
    ctx,
    db,
    map[string]any{"id": 1, "name": "alice"},
    orm.Table("users"),
    orm.PK("id"),
    orm.WherePK(),
)
```

`PK(...)` is for map writes. Struct writes use `db:"...,pk"` tags instead.

## Transactions

The generic helpers take `*orm.DB`. Inside a transaction callback, pass `tx.DB`.

### `db.Transaction(...)`

```go
err := db.Transaction(func(tx orm.Tx) error {
    _, err := orm.Update(
        ctx,
        tx.DB,
        User{ID: 1, Active: true},
        orm.Columns("active"),
        orm.WherePK(),
    )
    return err
})
```

### `db.TransactionContext(...)`

```go
err := db.TransactionContext(ctx, func(tx orm.Tx) error {
    user, err := orm.SelectOne[User](ctx, tx.DB, "SELECT id, name, age FROM users WHERE id = ?", 1)
    if err != nil {
        return err
    }
    _, err = orm.Update(ctx, tx.DB, User{ID: user.ID, Age: user.Age + 1}, orm.Columns("age"), orm.WherePK())
    return err
})
```

### Manual `Begin()` / `BeginTx(...)`

```go
tx, err := db.BeginTx(ctx, nil)
if err != nil {
    log.Fatal(err)
}
defer tx.Rollback()

if _, err := orm.Insert(ctx, tx.DB, User{Name: "sam", Age: 18}); err != nil {
    log.Fatal(err)
}

if err := tx.Commit(); err != nil {
    log.Fatal(err)
}
```

The same pattern works with `db.Begin()`.

## Dialect notes

- goquent ships with built-in `orm.MySQL` and `orm.Postgres` driver names.
- `SelectOne` and `SelectAll` execute the SQL string you pass in, so placeholder syntax must match your driver.
- `Insert`, `Update`, and `Upsert` use the configured dialect to quote identifiers and build placeholders.
- `Returning(...)` is PostgreSQL-only in the current implementation.
- Bool scanning follows the same compatibility rules as the rest of goquent. See [Boolean dialect compatibility](../../README.md#boolean-dialect-compatibility).

## Limitations and caveats

- Reads only support struct destinations and `map[string]any`. Pointer destinations are not supported.
- Writes only support non-pointer struct values and `map[string]any`.
- Generic writes only support primary-key-based `Update` and `Upsert` through `WherePK()`. For arbitrary predicates, use the query-builder API.
- Struct `Update` and `Upsert` depend on `db:"...,pk"` tags. Without them, `WherePK()` has no primary-key columns to use.
- Since generic writes take struct values, a `TableName() string` override must be available on the value type. A pointer-receiver-only `TableName` method is not picked up here.
- Map writes do not use struct metadata, so `readonly`, `omitempty`, and field tags do not apply.
- Mapping is reflection-based. Unmatched columns are ignored, missing columns leave zero values, and scan or type-conversion failures are returned as errors.
- There is no generic helper for `DELETE`.
- `Returning(...)` changes the generated SQL for PostgreSQL, but the API does not scan returned rows back into a value.

## Examples

### Select one struct

```go
user, err := orm.SelectOne[User](ctx, db, "SELECT id, name, age, active FROM users WHERE id = ?", 1)
if err != nil {
    log.Fatal(err)
}
_ = user
```

### Select many structs

```go
users, err := orm.SelectAll[User](ctx, db, "SELECT id, name, age, active FROM users WHERE active = ? ORDER BY id", true)
if err != nil {
    log.Fatal(err)
}
_ = users
```

### Select one `map[string]any`

```go
row, err := orm.SelectOne[map[string]any](ctx, db, "SELECT id, name FROM users WHERE id = ?", 1)
if err != nil {
    log.Fatal(err)
}
_ = row
```

### Insert a struct

```go
_, err := orm.Insert(ctx, db, User{Name: "sam", Age: 18, Active: true})
if err != nil {
    log.Fatal(err)
}
```

### Update selected columns on a struct

```go
_, err := orm.Update(
    ctx,
    db,
    User{ID: 1, Name: "alice", Active: true},
    orm.Columns("name", "active"),
    orm.WherePK(),
)
if err != nil {
    log.Fatal(err)
}
```

### Update a map with `Table(...)`, `PK(...)`, and `WherePK()`

```go
_, err := orm.Update(
    ctx,
    db,
    map[string]any{
        "id":     1,
        "name":   "alice",
        "active": true,
    },
    orm.Table("users"),
    orm.PK("id"),
    orm.Columns("name", "active"),
    orm.WherePK(),
)
if err != nil {
    log.Fatal(err)
}
```

### Upsert a struct

```go
_, err := orm.Upsert(
    ctx,
    db,
    User{ID: 1, Name: "alice", Age: 31, Active: true},
    orm.WherePK(),
)
if err != nil {
    log.Fatal(err)
}
```

### Use the generic API inside a transaction

```go
err := db.TransactionContext(ctx, func(tx orm.Tx) error {
    user, err := orm.SelectOne[User](ctx, tx.DB, "SELECT id, name, age, active FROM users WHERE id = ?", 1)
    if err != nil {
        return err
    }
    _, err = orm.Update(
        ctx,
        tx.DB,
        User{ID: user.ID, Active: !user.Active},
        orm.Columns("active"),
        orm.WherePK(),
    )
    return err
})
if err != nil {
    log.Fatal(err)
}
```
