# goquent Documentation

## Index

- [ORM package API](./orm/README.md)
- [Generic CRUD guide](./orm/generic-crud.md)
- [Driver](./orm/driver/README.md)
- [Model](./orm/model/README.md)
- [Query](./orm/query/README.md)
- [Scanner](./orm/scanner/README.md)
- [Conversion](./orm/conv/README.md)

See the [QuickStart](../examples/quickstart/main.go) for a practical example.

## Generic CRUD Example

```go
type User struct {
	ID     int64  `db:"id,pk"`
	Name   string `db:"name"`
	Age    int    `db:"age"`
	Active bool   `db:"active"`
}

func inactiveAdults() orm.Scope {
	return func(q *query.Query) *query.Query {
		return q.Where("age", ">", 20).
			Where("active", false).
			OrderBy("id", "asc")
	}
}

ctx := context.Background()
user, _ := orm.SelectOne[User](ctx, db, "SELECT id, name, age, active FROM users WHERE id = ?", 1)
users, _ := orm.SelectAllBy[User](ctx, db, db.Model(&User{}), inactiveAdults())
_, _ = orm.Update(ctx, db, User{ID: user.ID, Active: true}, orm.Columns("active"), orm.WherePK())
_, _ = orm.UpdateBy(ctx, db.Table("users"), map[string]any{"active": true}, inactiveAdults())
_, _ = orm.DeleteBy(ctx, db.Table("users"), func(q *query.Query) *query.Query {
	return q.Where("age", "<", 13)
})
_ = users
```
