package query

import (
	"database/sql"
	"fmt"
	"reflect"
	"unsafe"

	qbapi "github.com/faciam-dev/goquent-query-builder/api"
	qbmysql "github.com/faciam-dev/goquent-query-builder/database/mysql"
	"goquent/orm/scanner"
)

// Query wraps goquent QueryBuilder and the Driver.
// executor abstracts sql.DB and sql.Tx.
type executor interface {
	Query(query string, args ...any) (*sql.Rows, error)
	Exec(query string, args ...any) (sql.Result, error)
}

// Query wraps goquent QueryBuilder and the executor.
type Query struct {
	builder *qbapi.SelectQueryBuilder
	exec    executor
	err     error
}

// New creates a Query with given db and table.
func New(exec executor, table string) *Query {
	builder := qbapi.NewSelectQueryBuilder(qbmysql.NewMySQLQueryBuilder())
	builder.Table(table)
	return &Query{builder: builder, exec: exec}
}

// Select sets selected columns.
func (q *Query) Select(cols ...string) *Query {
	q.builder.Select(cols...)
	return q
}

// Where appends a condition.
func (q *Query) Where(col string, args ...any) *Query {
	if q.err != nil {
		return q
	}
	switch len(args) {
	case 1:
		q.builder.Where(col, "=", args[0])
	case 2:
		op, ok := args[0].(string)
		if !ok {
			q.err = fmt.Errorf("invalid operator type")
			return q
		}
		q.builder.Where(col, op, args[1])
	default:
		q.err = fmt.Errorf("invalid Where usage")
	}
	return q
}

// First scans the first result into dest struct.
func (q *Query) First(dest any) error {
	if q.err != nil {
		return q.err
	}
	sqlStr, args, err := q.builder.Build()
	if err != nil {
		return err
	}
	rows, err := q.exec.Query(sqlStr, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	return scanner.Struct(dest, rows)
}

// FirstMap scans first row into map.
func (q *Query) FirstMap(dest *map[string]any) error {
	if q.err != nil {
		return q.err
	}
	sqlStr, args, err := q.builder.Build()
	if err != nil {
		return err
	}
	rows, err := q.exec.Query(sqlStr, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	m, err := scanner.Map(rows)
	if err != nil {
		return err
	}
	*dest = m
	return nil
}

// GetMaps scans all rows into slice of maps.
func (q *Query) GetMaps(dest *[]map[string]any) error {
	if q.err != nil {
		return q.err
	}
	sqlStr, args, err := q.builder.Build()
	if err != nil {
		return err
	}
	rows, err := q.exec.Query(sqlStr, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	m, err := scanner.Maps(rows)
	if err != nil {
		return err
	}
	*dest = m
	return nil
}

// Limit sets a limit.
func (q *Query) Limit(n int) *Query {
	q.builder.Limit(int64(n))
	return q
}

// Offset sets offset.
func (q *Query) Offset(n int) *Query {
	q.builder.Offset(int64(n))
	return q
}

// SelectRaw adds a raw select expression.
func (q *Query) SelectRaw(raw string, values ...any) *Query {
	q.builder.SelectRaw(raw, values...)
	return q
}

// Count adds COUNT aggregate functions.
func (q *Query) Count(cols ...string) *Query {
	q.builder.Count(cols...)
	return q
}

// Distinct marks columns as DISTINCT.
func (q *Query) Distinct(cols ...string) *Query {
	q.builder.Distinct(cols...)
	return q
}

// Union adds a UNION with another query.
func (q *Query) Union(sub *Query) *Query {
	q.builder.Union(sub.builder)
	return q
}

// UnionAll adds a UNION ALL with another query.
func (q *Query) UnionAll(sub *Query) *Query {
	q.builder.UnionAll(sub.builder)
	return q
}

// Max adds MAX aggregate function.
func (q *Query) Max(col string) *Query { q.builder.Max(col); return q }

// Min adds MIN aggregate function.
func (q *Query) Min(col string) *Query { q.builder.Min(col); return q }

// Sum adds SUM aggregate function.
func (q *Query) Sum(col string) *Query { q.builder.Sum(col); return q }

// Avg adds AVG aggregate function.
func (q *Query) Avg(col string) *Query { q.builder.Avg(col); return q }

// Join adds INNER JOIN clause.
func (q *Query) Join(table, localColumn, cond, target string) *Query {
	q.builder.Join(table, localColumn, cond, target)
	return q
}

// LeftJoin adds LEFT JOIN clause.
func (q *Query) LeftJoin(table, localColumn, cond, target string) *Query {
	q.builder.LeftJoin(table, localColumn, cond, target)
	return q
}

// RightJoin adds RIGHT JOIN clause.
func (q *Query) RightJoin(table, localColumn, cond, target string) *Query {
	q.builder.RightJoin(table, localColumn, cond, target)
	return q
}

// CrossJoin adds CROSS JOIN clause.
func (q *Query) CrossJoin(table string) *Query {
	q.builder.CrossJoin(table)
	return q
}

// OrderBy adds ORDER BY clause.
func (q *Query) OrderBy(col, dir string) *Query {
	q.builder.OrderBy(col, dir)
	return q
}

// OrderByRaw adds raw ORDER BY clause.
func (q *Query) OrderByRaw(raw string) *Query {
	q.builder.OrderByRaw(raw)
	return q
}

// ReOrder clears ORDER BY clauses.
func (q *Query) ReOrder() *Query {
	q.builder.ReOrder()
	return q
}

// GroupBy adds GROUP BY clause.
func (q *Query) GroupBy(cols ...string) *Query {
	q.builder.GroupBy(cols...)
	return q
}

// Having adds HAVING condition.
func (q *Query) Having(col, cond string, val any) *Query {
	q.builder.Having(col, cond, val)
	return q
}

// HavingRaw adds raw HAVING condition.
func (q *Query) HavingRaw(raw string) *Query {
	q.builder.HavingRaw(raw)
	return q
}

// OrHaving adds OR HAVING condition.
func (q *Query) OrHaving(col, cond string, val any) *Query {
	q.builder.OrHaving(col, cond, val)
	return q
}

// OrHavingRaw adds raw OR HAVING condition.
func (q *Query) OrHavingRaw(raw string) *Query {
	q.builder.OrHavingRaw(raw)
	return q
}

// OrWhere appends OR condition.
func (q *Query) OrWhere(col string, args ...any) *Query {
	if q.err != nil {
		return q
	}
	switch len(args) {
	case 1:
		q.builder.OrWhere(col, "=", args[0])
	case 2:
		op, ok := args[0].(string)
		if !ok {
			q.err = fmt.Errorf("invalid operator type")
			return q
		}
		q.builder.OrWhere(col, op, args[1])
	default:
		q.err = fmt.Errorf("invalid OrWhere usage")
	}
	return q
}

// WhereRaw appends raw WHERE condition.
func (q *Query) WhereRaw(raw string, vals map[string]any) *Query {
	q.builder.WhereRaw(raw, vals)
	return q
}

// OrWhereRaw appends raw OR WHERE condition.
func (q *Query) OrWhereRaw(raw string, vals map[string]any) *Query {
	q.builder.OrWhereRaw(raw, vals)
	return q
}

// WhereIn adds WHERE IN condition.
func (q *Query) WhereIn(col string, vals any) *Query {
	q.builder.WhereIn(col, vals)
	return q
}

// WhereNotIn adds WHERE NOT IN condition.
func (q *Query) WhereNotIn(col string, vals any) *Query {
	q.builder.WhereNotIn(col, vals)
	return q
}

// OrWhereIn adds OR WHERE IN condition.
func (q *Query) OrWhereIn(col string, vals any) *Query {
	q.builder.OrWhereIn(col, vals)
	return q
}

// OrWhereNotIn adds OR WHERE NOT IN condition.
func (q *Query) OrWhereNotIn(col string, vals any) *Query {
	q.builder.OrWhereNotIn(col, vals)
	return q
}

// WhereNull adds WHERE column IS NULL condition.
func (q *Query) WhereNull(col string) *Query {
	q.builder.WhereNull(col)
	return q
}

// WhereNotNull adds WHERE column IS NOT NULL condition.
func (q *Query) WhereNotNull(col string) *Query {
	q.builder.WhereNotNull(col)
	return q
}

// OrWhereNull adds OR WHERE column IS NULL condition.
func (q *Query) OrWhereNull(col string) *Query {
	q.builder.OrWhereNull(col)
	return q
}

// OrWhereNotNull adds OR WHERE column IS NOT NULL condition.
func (q *Query) OrWhereNotNull(col string) *Query {
	q.builder.OrWhereNotNull(col)
	return q
}

// WhereBetween adds WHERE BETWEEN condition.
func (q *Query) WhereBetween(col string, min, max any) *Query {
	q.builder.WhereBetween(col, min, max)
	return q
}

// WhereNotBetween adds WHERE NOT BETWEEN condition.
func (q *Query) WhereNotBetween(col string, min, max any) *Query {
	q.builder.WhereNotBetween(col, min, max)
	return q
}

// OrWhereBetween adds OR WHERE BETWEEN condition.
func (q *Query) OrWhereBetween(col string, min, max any) *Query {
	q.builder.OrWhereBetween(col, min, max)
	return q
}

// OrWhereNotBetween adds OR WHERE NOT BETWEEN condition.
func (q *Query) OrWhereNotBetween(col string, min, max any) *Query {
	q.builder.OrWhereNotBetween(col, min, max)
	return q
}

// WhereExists adds WHERE EXISTS (subquery) condition.
func (q *Query) WhereExists(sub *Query) *Query {
	q.builder.WhereExistsSubQuery(sub.builder)
	return q
}

// OrWhereExists adds OR WHERE EXISTS (subquery) condition.
func (q *Query) OrWhereExists(sub *Query) *Query {
	q.builder.OrWhereExistsSubQuery(sub.builder)
	return q
}

// WhereNotExists adds WHERE NOT EXISTS (subquery) condition.
func (q *Query) WhereNotExists(sub *Query) *Query {
	q.builder.WhereNotExistsQuery(sub.builder)
	return q
}

// OrWhereNotExists adds OR WHERE NOT EXISTS (subquery) condition.
func (q *Query) OrWhereNotExists(sub *Query) *Query {
	q.builder.OrWhereNotExistsQuery(sub.builder)
	return q
}

// WhereDate adds WHERE DATE(column) comparison condition.
func (q *Query) WhereDate(col, cond, date string) *Query {
	q.builder.WhereDate(col, cond, date)
	return q
}

// OrWhereDate adds OR WHERE DATE(column) comparison condition.
func (q *Query) OrWhereDate(col, cond, date string) *Query {
	q.builder.OrWhereDate(col, cond, date)
	return q
}

// Take is an alias of Limit.
func (q *Query) Take(n int) *Query { return q.Limit(n) }

// Skip is an alias of Offset.
func (q *Query) Skip(n int) *Query { return q.Offset(n) }

// SharedLock adds LOCK IN SHARE MODE clause.
func (q *Query) SharedLock() *Query {
	q.builder.SharedLock()
	return q
}

// LockForUpdate adds FOR UPDATE clause.
func (q *Query) LockForUpdate() *Query {
	q.builder.LockForUpdate()
	return q
}

// Build returns the SQL and args.
func (q *Query) Build() (string, []any, error) { return q.builder.Build() }

// Dump returns SQL and args for debugging.
func (q *Query) Dump() (string, []any, error) { return q.builder.Dump() }

// RawSQL returns interpolated SQL for debugging.
func (q *Query) RawSQL() (string, error) { return q.builder.RawSql() }

// Insert executes an INSERT with the given data.
func (q *Query) Insert(data map[string]any) (sql.Result, error) {
	ib := qbapi.NewInsertQueryBuilder(qbmysql.NewMySQLQueryBuilder())
	ib.Table(q.builder.GetQuery().Table.Name).Insert(data)
	sqlStr, args, err := ib.Build()
	if err != nil {
		return nil, err
	}
	return q.exec.Exec(sqlStr, args...)
}

// InsertGetId executes an INSERT and returns the auto-increment ID.
func (q *Query) InsertGetId(data map[string]any) (int64, error) {
	res, err := q.Insert(data)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

// InsertBatch executes a bulk INSERT with the given slice of data maps.
func (q *Query) InsertBatch(data []map[string]any) (sql.Result, error) {
	ib := qbapi.NewInsertQueryBuilder(qbmysql.NewMySQLQueryBuilder())
	ib.Table(q.builder.GetQuery().Table.Name).InsertBatch(data)
	sqlStr, args, err := ib.Build()
	if err != nil {
		return nil, err
	}
	return q.exec.Exec(sqlStr, args...)
}

// InsertUsing executes an INSERT INTO ... SELECT statement using columns from a subquery.
func (q *Query) InsertUsing(columns []string, sub *Query) (sql.Result, error) {
	ib := qbapi.NewInsertQueryBuilder(qbmysql.NewMySQLQueryBuilder())
	ib.Table(q.builder.GetQuery().Table.Name).InsertUsing(columns, sub.builder)
	sqlStr, args, err := ib.Build()
	if err != nil {
		return nil, err
	}
	return q.exec.Exec(sqlStr, args...)
}

// Update executes an UPDATE with the given data.
func (q *Query) Update(data map[string]any) (sql.Result, error) {
	ub := qbapi.NewUpdateQueryBuilder(qbmysql.NewMySQLQueryBuilder())
	ub.Table(q.builder.GetQuery().Table.Name).Update(data)
	copyBuilderState(q.builder, ub)
	sqlStr, args, err := ub.Build()
	if err != nil {
		return nil, err
	}
	return q.exec.Exec(sqlStr, args...)
}

// Delete executes a DELETE query using current conditions.
func (q *Query) Delete() (sql.Result, error) {
	db := qbapi.NewDeleteQueryBuilder(qbmysql.NewMySQLQueryBuilder())
	db.Table(q.builder.GetQuery().Table.Name).Delete()
	copyBuilderStateDelete(q.builder, db)
	sqlStr, args, err := db.Build()
	if err != nil {
		return nil, err
	}
	return q.exec.Exec(sqlStr, args...)
}

// copyBuilderState duplicates where, join and order clauses from src to dst.
func copyBuilderState(src *qbapi.SelectQueryBuilder, dst *qbapi.UpdateQueryBuilder) {
	// copy where
	srcWb := src.GetWhereBuilder()
	dstWb := dst.GetWhereBuilder()
	_ = setFieldValue(dstWb, "query", reflect.ValueOf(srcWb).Elem().FieldByName("query"))

	// copy join
	srcJb := src.GetJoinBuilder()
	dstJb := dst.GetJoinBuilder()
	// deep copy joins to avoid sharing slices between builders. The query
	// builder does not expose a cloning API, so reflection is used here.
	newJoins := deepCopyJoins(srcJb)
	_ = setFieldValue(dstJb, "Joins", newJoins)

	// copy order
	srcOb := src.GetOrderByBuilder()
	dstOb := dst.GetOrderByBuilder()
	_ = setFieldValue(dstOb, "Order", reflect.ValueOf(srcOb).Elem().FieldByName("Order"))
}

// copyBuilderStateDelete duplicates where, join and order clauses from src to a DeleteQueryBuilder.
func copyBuilderStateDelete(src *qbapi.SelectQueryBuilder, dst *qbapi.DeleteQueryBuilder) {
	// copy where
	srcWb := src.GetWhereBuilder()
	dstWb := dst.GetWhereBuilder()
	_ = setFieldValue(dstWb, "query", reflect.ValueOf(srcWb).Elem().FieldByName("query"))

	// copy join
	srcJb := src.GetJoinBuilder()
	dstJb := dst.GetJoinBuilder()
	newJoins := deepCopyJoins(srcJb)
	_ = setFieldValue(dstJb, "Joins", newJoins)

	// copy order
	srcOb := src.GetOrderByBuilder()
	dstOb := dst.GetOrderByBuilder()
	_ = setFieldValue(dstOb, "Order", reflect.ValueOf(srcOb).Elem().FieldByName("Order"))
}

// deepCopyJoins clones the internal Joins struct of a JoinBuilder using
// reflection. This avoids sharing slices between builders.
func deepCopyJoins(jb any) reflect.Value {
	joinsVal := reflect.ValueOf(jb).Elem().FieldByName("Joins")
	newJoins := reflect.New(joinsVal.Elem().Type())
	newJoins.Elem().Set(joinsVal.Elem())
	for _, name := range []string{"Joins", "JoinClauses", "LateralJoins"} {
		slice := joinsVal.Elem().FieldByName(name)
		if slice.IsValid() && !slice.IsNil() {
			cp := reflect.MakeSlice(slice.Type().Elem(), slice.Elem().Len(), slice.Elem().Len())
			reflect.Copy(cp, slice.Elem())
			newJoins.Elem().FieldByName(name).Set(cp.Addr())
		}
	}
	return newJoins
}

// setFieldValue assigns value to an exported field using reflection.
// It does not manipulate unexported fields to ensure safety and maintainability.
func setFieldValue(target any, field string, value reflect.Value) error {
	v := reflect.ValueOf(target).Elem().FieldByName(field)
	if !v.IsValid() {
		return fmt.Errorf("field %q does not exist in target", field)
	}
	if v.Type() != value.Type() {
		return fmt.Errorf("type mismatch for field %q", field)
	}
	if v.CanSet() {
		v.Set(value)
		return nil
	}
	p := unsafe.Pointer(v.UnsafeAddr())
	reflect.NewAt(v.Type(), p).Elem().Set(value)
	return nil
}
