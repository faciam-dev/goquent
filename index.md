# goquent Documentation

## 目次

- [Package API](./README.md)
- [Driver](./orm/driver/README.md)
- [Model](./orm/model/README.md)
- [Query](./orm/query/README.md)
- [Scanner](./orm/scanner/README.md)
- [Conversion](./orm/conv/README.md)

See the [QuickStart](../examples/quickstart/main.go) for a practical example.

## Generic CRUD Example

```go
u, _ := orm.SelectOne[User](ctx, db, "SELECT * FROM users WHERE id = ?", 1)
rows, _ := orm.SelectAll[map[string]any](ctx, db, "SELECT * FROM users")
_, _ = orm.Insert(ctx, db, User{Name: "sam", Age: 18})
```
