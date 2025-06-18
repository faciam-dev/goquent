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

// Take is an alias of Limit.
func (q *Query) Take(n int) *Query { return q.Limit(n) }

// Skip is an alias of Offset.
func (q *Query) Skip(n int) *Query { return q.Offset(n) }

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

// copyBuilderState duplicates where, join and order clauses from src to dst.
func copyBuilderState(src *qbapi.SelectQueryBuilder, dst *qbapi.UpdateQueryBuilder) {
	// copy where
	srcWb := src.GetWhereBuilder()
	dstWb := dst.GetWhereBuilder()
	_ = copyPtrField(dstWb, "query", reflect.ValueOf(srcWb).Elem().FieldByName("query").Pointer())

	// copy join
	srcJb := src.GetJoinBuilder()
	dstJb := dst.GetJoinBuilder()
	// deep copy joins to avoid sharing the slice between builders
	joinsVal := reflect.ValueOf(srcJb).Elem().FieldByName("Joins")
	newJoins := reflect.New(joinsVal.Elem().Type())
	newJoins.Elem().Set(joinsVal.Elem())
	if slice := joinsVal.Elem().FieldByName("Joins"); slice.IsValid() && !slice.IsNil() {
		cp := reflect.MakeSlice(slice.Type().Elem(), slice.Elem().Len(), slice.Elem().Len())
		reflect.Copy(cp, slice.Elem())
		newJoins.Elem().FieldByName("Joins").Set(cp.Addr())
	}
	if slice := joinsVal.Elem().FieldByName("JoinClauses"); slice.IsValid() && !slice.IsNil() {
		cp := reflect.MakeSlice(slice.Type().Elem(), slice.Elem().Len(), slice.Elem().Len())
		reflect.Copy(cp, slice.Elem())
		newJoins.Elem().FieldByName("JoinClauses").Set(cp.Addr())
	}
	if slice := joinsVal.Elem().FieldByName("LateralJoins"); slice.IsValid() && !slice.IsNil() {
		cp := reflect.MakeSlice(slice.Type().Elem(), slice.Elem().Len(), slice.Elem().Len())
		reflect.Copy(cp, slice.Elem())
		newJoins.Elem().FieldByName("LateralJoins").Set(cp.Addr())
	}
	_ = copyPtrField(dstJb, "Joins", newJoins.Pointer())

	// copy order
	srcOb := src.GetOrderByBuilder()
	dstOb := dst.GetOrderByBuilder()
	_ = copyPtrField(dstOb, "Order", reflect.ValueOf(srcOb).Elem().FieldByName("Order").Pointer())
}

func copyPtrField(target any, field string, ptr uintptr) error {
	v := reflect.ValueOf(target).Elem().FieldByName(field)
	if !v.IsValid() {
		return fmt.Errorf("field %q does not exist in target", field)
	}
	if !v.CanSet() {
		return fmt.Errorf("field %q is not settable", field)
	}
	p := unsafe.Pointer(v.UnsafeAddr())
	*(*uintptr)(p) = ptr
	return nil
}
