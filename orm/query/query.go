package query

import (
	"database/sql"

	qbapi "github.com/faciam-dev/goquent-query-builder/api"
	qbmysql "github.com/faciam-dev/goquent-query-builder/database/mysql"
	"goquent/orm/scanner"
)

// Query wraps goquent QueryBuilder and the Driver.
// executor abstracts sql.DB and sql.Tx.
type executor interface {
	Query(query string, args ...any) (*sql.Rows, error)
}

// Query wraps goquent QueryBuilder and the executor.
type Query struct {
	builder *qbapi.SelectQueryBuilder
	exec    executor
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
	switch len(args) {
	case 1:
		q.builder.Where(col, "=", args[0])
	case 2:
		q.builder.Where(col, args[0].(string), args[1])
	default:
		// invalid usage
	}
	return q
}

// First scans the first result into dest struct.
func (q *Query) First(dest any) error {
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
