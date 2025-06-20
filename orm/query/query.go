package query

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	qbapi "github.com/faciam-dev/goquent-query-builder/api"
	qbmysql "github.com/faciam-dev/goquent-query-builder/database/mysql"
	"github.com/faciam-dev/goquent/orm/scanner"
)

// Query wraps goquent QueryBuilder and the Driver.
// executor abstracts sql.DB and sql.Tx.
type executor interface {
	// Query executes a statement returning multiple rows.
	Query(query string, args ...any) (*sql.Rows, error)
	// QueryContext is the context-aware form of Query.
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	// QueryRow executes a single-row query.
	QueryRow(query string, args ...any) *sql.Row
	// QueryRowContext executes QueryRow with a context.
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	// Exec runs a statement that does not return rows.
	Exec(query string, args ...any) (sql.Result, error)
	// ExecContext runs Exec with a context.
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// Query wraps goquent QueryBuilder and the executor.
type Query struct {
	builder *qbapi.SelectQueryBuilder
	exec    executor
	ctx     context.Context
	err     error
}

// New creates a Query with given db and table.
func New(exec executor, table string) *Query {
	builder := qbapi.NewSelectQueryBuilder(qbmysql.NewMySQLQueryBuilder())
	builder.Table(table)
	return &Query{builder: builder, exec: exec}
}

// WithContext sets ctx on the query for context-aware execution.
func (q *Query) WithContext(ctx context.Context) *Query {
	q.ctx = ctx
	return q
}

// queryRows executes Query or QueryContext based on whether ctx is set.
func (q *Query) queryRows(sqlStr string, args ...any) (*sql.Rows, error) {
	if q.ctx != nil {
		return q.exec.QueryContext(q.ctx, sqlStr, args...)
	}
	return q.exec.Query(sqlStr, args...)
}

// execStmt executes Exec or ExecContext depending on ctx.
func (q *Query) execStmt(sqlStr string, args ...any) (sql.Result, error) {
	if q.ctx != nil {
		return q.exec.ExecContext(q.ctx, sqlStr, args...)
	}
	return q.exec.Exec(sqlStr, args...)
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
	rows, err := q.queryRows(sqlStr, args...)
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
	rows, err := q.queryRows(sqlStr, args...)
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
	rows, err := q.queryRows(sqlStr, args...)
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

// Get scans all rows into the slice pointed to by dest.
func (q *Query) Get(dest any) error {
	if q.err != nil {
		return q.err
	}
	sqlStr, args, err := q.builder.Build()
	if err != nil {
		return err
	}
	rows, err := q.queryRows(sqlStr, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	return scanner.Structs(dest, rows)
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

// JoinQuery adds a JOIN with additional ON/WHERE clauses defined in the callback.
func (q *Query) JoinQuery(table string, fn func(b *qbapi.JoinClauseQueryBuilder)) *Query {
	q.builder.JoinQuery(table, func(b *qbapi.JoinClauseQueryBuilder) { fn(b) })
	return q
}

// LeftJoinQuery adds a LEFT JOIN with additional clauses defined in the callback.
func (q *Query) LeftJoinQuery(table string, fn func(b *qbapi.JoinClauseQueryBuilder)) *Query {
	q.builder.LeftJoinQuery(table, func(b *qbapi.JoinClauseQueryBuilder) { fn(b) })
	return q
}

// RightJoinQuery adds a RIGHT JOIN with additional clauses defined in the callback.
func (q *Query) RightJoinQuery(table string, fn func(b *qbapi.JoinClauseQueryBuilder)) *Query {
	q.builder.RightJoinQuery(table, func(b *qbapi.JoinClauseQueryBuilder) { fn(b) })
	return q
}

// JoinSubQuery joins a subquery with alias and join condition.
func (q *Query) JoinSubQuery(sub *Query, alias, my, condition, target string) *Query {
	q.builder.JoinSubQuery(sub.builder, alias, my, condition, target)
	return q
}

// LeftJoinSubQuery performs a LEFT JOIN using a subquery.
func (q *Query) LeftJoinSubQuery(sub *Query, alias, my, condition, target string) *Query {
	q.builder.LeftJoinSubQuery(sub.builder, alias, my, condition, target)
	return q
}

// RightJoinSubQuery performs a RIGHT JOIN using a subquery.
func (q *Query) RightJoinSubQuery(sub *Query, alias, my, condition, target string) *Query {
	q.builder.RightJoinSubQuery(sub.builder, alias, my, condition, target)
	return q
}

// JoinLateral performs a LATERAL JOIN using a subquery.
func (q *Query) JoinLateral(sub *Query, alias string) *Query {
	q.builder.JoinLateral(sub.builder, alias)
	return q
}

// LeftJoinLateral performs a LEFT LATERAL JOIN using a subquery.
func (q *Query) LeftJoinLateral(sub *Query, alias string) *Query {
	q.builder.LeftJoinLateral(sub.builder, alias)
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

// SafeWhereRaw appends a raw WHERE condition ensuring a values map is always used.
func (q *Query) SafeWhereRaw(raw string, vals map[string]any) *Query {
	q.builder.SafeWhereRaw(raw, vals)
	return q
}

// SafeOrWhereRaw appends a raw OR WHERE condition ensuring a values map is used.
func (q *Query) SafeOrWhereRaw(raw string, vals map[string]any) *Query {
	q.builder.SafeOrWhereRaw(raw, vals)
	return q
}

// WhereGroup groups conditions with parentheses using AND logic.
func (q *Query) WhereGroup(fn func(g *Query)) *Query {
	q.builder.WhereGroup(func(b *qbapi.WhereSelectQueryBuilder) {
		grp := &Query{builder: q.builder, exec: q.exec, ctx: q.ctx}
		_ = setFieldValue(&grp.builder.WhereQueryBuilder, "builder", reflect.ValueOf(b.GetBuilder()))
		fn(grp)
	})
	return q
}

// OrWhereGroup groups conditions with parentheses using OR logic.
func (q *Query) OrWhereGroup(fn func(g *Query)) *Query {
	q.builder.OrWhereGroup(func(b *qbapi.WhereSelectQueryBuilder) {
		grp := &Query{builder: q.builder, exec: q.exec, ctx: q.ctx}
		_ = setFieldValue(&grp.builder.WhereQueryBuilder, "builder", reflect.ValueOf(b.GetBuilder()))
		fn(grp)
	})
	return q
}

// WhereNot groups conditions inside NOT (...).
func (q *Query) WhereNot(fn func(g *Query)) *Query {
	q.builder.WhereNot(func(b *qbapi.WhereSelectQueryBuilder) {
		grp := &Query{builder: q.builder, exec: q.exec, ctx: q.ctx}
		_ = setFieldValue(&grp.builder.WhereQueryBuilder, "builder", reflect.ValueOf(b.GetBuilder()))
		fn(grp)
	})
	return q
}

// OrWhereNot groups conditions inside OR NOT (...).
func (q *Query) OrWhereNot(fn func(g *Query)) *Query {
	q.builder.OrWhereNot(func(b *qbapi.WhereSelectQueryBuilder) {
		grp := &Query{builder: q.builder, exec: q.exec, ctx: q.ctx}
		_ = setFieldValue(&grp.builder.WhereQueryBuilder, "builder", reflect.ValueOf(b.GetBuilder()))
		fn(grp)
	})
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

// WhereInSubQuery adds WHERE IN (subquery) condition.
func (q *Query) WhereInSubQuery(col string, sub *Query) *Query {
	q.builder.WhereInSubQuery(col, sub.builder)
	return q
}

// WhereNotInSubQuery adds WHERE NOT IN (subquery) condition.
func (q *Query) WhereNotInSubQuery(col string, sub *Query) *Query {
	q.builder.WhereNotInSubQuery(col, sub.builder)
	return q
}

// OrWhereInSubQuery adds OR WHERE IN (subquery) condition.
func (q *Query) OrWhereInSubQuery(col string, sub *Query) *Query {
	q.builder.OrWhereInSubQuery(col, sub.builder)
	return q
}

// OrWhereNotInSubQuery adds OR WHERE NOT IN (subquery) condition.
func (q *Query) OrWhereNotInSubQuery(col string, sub *Query) *Query {
	q.builder.OrWhereNotInSubQuery(col, sub.builder)
	return q
}

// WhereAny adds grouped OR conditions across columns.
func (q *Query) WhereAny(cols []string, cond string, val any) *Query {
	q.builder.WhereAny(cols, cond, val)
	return q
}

// WhereAll adds grouped AND conditions across columns.
func (q *Query) WhereAll(cols []string, cond string, val any) *Query {
	q.builder.WhereAll(cols, cond, val)
	return q
}

// WhereColumn adds WHERE column operator column condition.
func (q *Query) WhereColumn(col string, args ...string) *Query {
	var op, other string
	switch len(args) {
	case 1:
		op = "="
		other = args[0]
	case 2:
		op = args[0]
		other = args[1]
	default:
		q.err = fmt.Errorf("invalid WhereColumn usage")
		return q
	}
	columnsPair := []string{col, other}
	q.builder.WhereColumn(columnsPair, col, op, other)
	return q
}

// OrWhereColumn adds OR WHERE column operator column condition.
func (q *Query) OrWhereColumn(col string, args ...string) *Query {
	var op, other string
	switch len(args) {
	case 1:
		op = "="
		other = args[0]
	case 2:
		op = args[0]
		other = args[1]
	default:
		q.err = fmt.Errorf("invalid OrWhereColumn usage")
		return q
	}
	columnsPair := []string{col, other}
	q.builder.OrWhereColumn(columnsPair, col, op, other)
	return q
}

// WhereColumns adds multiple column comparison conditions joined by AND.
func (q *Query) WhereColumns(columns [][]string) *Query {
	all, err := gatherColumns(columns)
	if err != nil {
		q.err = err
		return q
	}
	q.builder.WhereColumns(all, columns)
	return q
}

// OrWhereColumns adds multiple column comparison conditions joined by OR.
func (q *Query) OrWhereColumns(columns [][]string) *Query {
	all, err := gatherColumns(columns)
	if err != nil {
		q.err = err
		return q
	}
	q.builder.OrWhereColumns(all, columns)
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

// WhereBetweenColumns adds WHERE col BETWEEN minCol AND maxCol using columns.
func (q *Query) WhereBetweenColumns(col, minCol, maxCol string) *Query {
	cols := []string{col, minCol, maxCol}
	q.builder.WhereBetweenColumns(cols, col, minCol, maxCol)
	return q
}

// OrWhereBetweenColumns adds OR WHERE col BETWEEN minCol AND maxCol using columns.
func (q *Query) OrWhereBetweenColumns(col, minCol, maxCol string) *Query {
	cols := []string{col, minCol, maxCol}
	q.builder.OrWhereBetweenColumns(cols, col, minCol, maxCol)
	return q
}

// WhereNotBetweenColumns adds WHERE col NOT BETWEEN minCol AND maxCol using columns.
func (q *Query) WhereNotBetweenColumns(col, minCol, maxCol string) *Query {
	cols := []string{col, minCol, maxCol}
	q.builder.WhereNotBetweenColumns(cols, col, minCol, maxCol)
	return q
}

// OrWhereNotBetweenColumns adds OR WHERE col NOT BETWEEN minCol AND maxCol using columns.
func (q *Query) OrWhereNotBetweenColumns(col, minCol, maxCol string) *Query {
	cols := []string{col, minCol, maxCol}
	q.builder.OrWhereNotBetweenColumns(cols, col, minCol, maxCol)
	return q
}

// WhereFullText adds full-text search condition.
func (q *Query) WhereFullText(cols []string, search string, opts map[string]any) *Query {
	q.builder.WhereFullText(cols, search, opts)
	return q
}

// OrWhereFullText adds OR full-text search condition.
func (q *Query) OrWhereFullText(cols []string, search string, opts map[string]any) *Query {
	q.builder.OrWhereFullText(cols, search, opts)
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

// WhereTime adds WHERE TIME(column) comparison condition.
func (q *Query) WhereTime(col, cond, time string) *Query {
	q.builder.WhereTime(col, cond, time)
	return q
}

// OrWhereTime adds OR WHERE TIME(column) comparison condition.
func (q *Query) OrWhereTime(col, cond, time string) *Query {
	q.builder.OrWhereTime(col, cond, time)
	return q
}

// WhereDay adds WHERE DAY(column) comparison condition.
func (q *Query) WhereDay(col, cond, day string) *Query {
	q.builder.WhereDay(col, cond, day)
	return q
}

// OrWhereDay adds OR WHERE DAY(column) comparison condition.
func (q *Query) OrWhereDay(col, cond, day string) *Query {
	q.builder.OrWhereDay(col, cond, day)
	return q
}

// WhereMonth adds WHERE MONTH(column) comparison condition.
func (q *Query) WhereMonth(col, cond, month string) *Query {
	q.builder.WhereMonth(col, cond, month)
	return q
}

// OrWhereMonth adds OR WHERE MONTH(column) comparison condition.
func (q *Query) OrWhereMonth(col, cond, month string) *Query {
	q.builder.OrWhereMonth(col, cond, month)
	return q
}

// WhereYear adds WHERE YEAR(column) comparison condition.
func (q *Query) WhereYear(col, cond, year string) *Query {
	q.builder.WhereYear(col, cond, year)
	return q
}

// OrWhereYear adds OR WHERE YEAR(column) comparison condition.
func (q *Query) OrWhereYear(col, cond, year string) *Query {
	q.builder.OrWhereYear(col, cond, year)
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
	return q.execStmt(sqlStr, args...)
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
	return q.execStmt(sqlStr, args...)
}

// InsertOrIgnore executes an INSERT IGNORE.
func (q *Query) InsertOrIgnore(data []map[string]any) (sql.Result, error) {
	ib := qbapi.NewInsertQueryBuilder(qbmysql.NewMySQLQueryBuilder())
	ib.Table(q.builder.GetQuery().Table.Name).InsertOrIgnore(data)
	sqlStr, args, err := ib.Build()
	if err != nil {
		return nil, err
	}
	return q.execStmt(sqlStr, args...)
}

// Upsert executes an UPSERT using ON DUPLICATE KEY UPDATE.
func (q *Query) Upsert(data []map[string]any, unique []string, updateCols []string) (sql.Result, error) {
	ib := qbapi.NewInsertQueryBuilder(qbmysql.NewMySQLQueryBuilder())
	ib.Table(q.builder.GetQuery().Table.Name).Upsert(data, unique, updateCols)
	sqlStr, args, err := ib.Build()
	if err != nil {
		return nil, err
	}
	return q.execStmt(sqlStr, args...)
}

// UpdateOrInsert performs UPDATE or INSERT based on condition.
func (q *Query) UpdateOrInsert(cond map[string]any, values map[string]any) (sql.Result, error) {
	ib := qbapi.NewInsertQueryBuilder(qbmysql.NewMySQLQueryBuilder())
	ib.Table(q.builder.GetQuery().Table.Name).UpdateOrInsert(cond, values)
	sqlStr, args, err := ib.Build()
	if err != nil {
		return nil, err
	}
	return q.execStmt(sqlStr, args...)
}

// InsertUsing executes an INSERT INTO ... SELECT statement using columns from a subquery.
func (q *Query) InsertUsing(columns []string, sub *Query) (sql.Result, error) {
	ib := qbapi.NewInsertQueryBuilder(qbmysql.NewMySQLQueryBuilder())
	ib.Table(q.builder.GetQuery().Table.Name).InsertUsing(columns, sub.builder)
	sqlStr, args, err := ib.Build()
	if err != nil {
		return nil, err
	}
	return q.execStmt(sqlStr, args...)
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
	return q.execStmt(sqlStr, args...)
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
	return q.execStmt(sqlStr, args...)
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

// deepCopyJoins clones the Joins value from a JoinBuilder using reflection.
// Each field of Joins is a pointer to a slice, so we copy the underlying
// slices to ensure the destination builder can modify them independently.
func deepCopyJoins(jb any) reflect.Value {
	joinsVal := reflect.ValueOf(jb).Elem().FieldByName("Joins")
	newJoins := reflect.New(joinsVal.Elem().Type())
	newJoins.Elem().Set(joinsVal.Elem())
	for _, name := range []string{"Joins", "JoinClauses", "LateralJoins"} {
		slice := joinsVal.Elem().FieldByName(name)
		if slice.IsValid() && !slice.IsNil() {
			sliceType := slice.Type().Elem()
			newSlice := reflect.MakeSlice(sliceType, slice.Elem().Len(), slice.Elem().Len())
			reflect.Copy(newSlice, slice.Elem())
			newSlicePtr := reflect.New(sliceType)
			newSlicePtr.Elem().Set(newSlice)
			newJoins.Elem().FieldByName(name).Set(newSlicePtr)
		}
	}
	return newJoins
}

// setFieldValue assigns value to an exported field using reflection.
// If the target field is unexported or cannot be set, it returns an error.
func setFieldValue(target any, field string, value reflect.Value) error {
	v := reflect.ValueOf(target).Elem().FieldByName(field)
	if !v.IsValid() {
		return fmt.Errorf("field %q does not exist in target", field)
	}
	if v.Type() != value.Type() {
		return fmt.Errorf("type mismatch for field %q", field)
	}
	if !v.CanSet() {
		return fmt.Errorf("cannot set field %q", field)
	}
	v.Set(value)
	return nil
}

// gatherColumns extracts unique column names from column comparison slices.
// Each slice must have length 2 (column, otherColumn) or 3 (column, operator, otherColumn).
// Returns an error if any slice has an unexpected length.
func gatherColumns(cols [][]string) ([]string, error) {
	set := make(map[string]struct{})
	for i, c := range cols {
		switch len(c) {
		case 2:
			set[c[0]] = struct{}{}
			set[c[1]] = struct{}{}
		case 3:
			set[c[0]] = struct{}{}
			set[c[2]] = struct{}{}
		default:
			return nil, fmt.Errorf("invalid column slice at index %d", i)
		}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	return out, nil
}
