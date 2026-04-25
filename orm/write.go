package orm

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/faciam-dev/goquent/orm/driver"
	"github.com/faciam-dev/goquent/orm/model"
)

// WriteOpt configures write behavior.
type WriteOpt func(*writeOptions)

type writeOptions struct {
	cols      map[string]struct{}
	omit      map[string]struct{}
	wherePK   bool
	returning []string
	table     string
	pkCols    map[string]struct{}
}

// Columns limits write to specified columns.
func Columns(cols ...string) WriteOpt {
	return func(o *writeOptions) {
		if o.cols == nil {
			o.cols = make(map[string]struct{}, len(cols))
		}
		for _, c := range cols {
			o.cols[c] = struct{}{}
		}
	}
}

// Omit excludes specified columns.
func Omit(cols ...string) WriteOpt {
	return func(o *writeOptions) {
		if o.omit == nil {
			o.omit = make(map[string]struct{}, len(cols))
		}
		for _, c := range cols {
			o.omit[c] = struct{}{}
		}
	}
}

// WherePK uses primary key columns in WHERE clause.
func WherePK() WriteOpt { return func(o *writeOptions) { o.wherePK = true } }

// Returning specifies columns to return (Postgres only).
func Returning(cols ...string) WriteOpt { return func(o *writeOptions) { o.returning = cols } }

// Table sets table name (required for map writes).
func Table(name string) WriteOpt { return func(o *writeOptions) { o.table = name } }

// PK specifies primary key columns for map writes.
func PK(cols ...string) WriteOpt {
	return func(o *writeOptions) {
		if o.pkCols == nil {
			o.pkCols = make(map[string]struct{}, len(cols))
		}
		for _, c := range cols {
			o.pkCols[c] = struct{}{}
		}
	}
}

func applyWriteOpts(opts []WriteOpt) *writeOptions {
	o := &writeOptions{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

func (o *writeOptions) isPK(col string) bool {
	if o.pkCols == nil {
		return false
	}
	_, ok := o.pkCols[col]
	return ok
}

func quote(d driver.Dialect, ident string) string { return d.QuoteIdent(ident) }

func buildPlaceholders(d driver.Dialect, n int, start int) []string {
	ph := make([]string, n)
	for i := 0; i < n; i++ {
		ph[i] = d.Placeholder(start + i)
	}
	return ph
}

type returningResult struct {
	rowsAffected int64
}

func (r returningResult) LastInsertId() (int64, error) {
	return 0, fmt.Errorf("LastInsertId is not supported for RETURNING statements")
}

func (r returningResult) RowsAffected() (int64, error) {
	return r.rowsAffected, nil
}

func appendReturningClause(d driver.Dialect, sqlStr string, cols []string) (string, error) {
	if len(cols) == 0 {
		return sqlStr, nil
	}
	if _, ok := d.(driver.PostgresDialect); !ok {
		return "", fmt.Errorf("Returning is not supported on dialect: %T", d)
	}
	rc := make([]string, len(cols))
	for i, c := range cols {
		rc[i] = quote(d, c)
	}
	return sqlStr + " RETURNING " + strings.Join(rc, ", "), nil
}

func execReturningRows(ctx context.Context, db *DB, sqlStr string, args ...any) (sql.Result, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if ctx != nil {
		rows, err = db.exec.QueryContext(ctx, sqlStr, args...)
	} else {
		rows, err = db.exec.Query(sqlStr, args...)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	scanDst := make([]any, len(cols))
	values := make([]any, len(cols))
	for i := range scanDst {
		scanDst[i] = &values[i]
	}

	var count int64
	for rows.Next() {
		if len(scanDst) > 0 {
			if err := rows.Scan(scanDst...); err != nil {
				return nil, err
			}
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return returningResult{rowsAffected: count}, nil
}

// Insert inserts v into its table.
func Insert[T any](ctx context.Context, db *DB, v T, opts ...WriteOpt) (sql.Result, error) {
	o := applyWriteOpts(opts)
	val := reflect.ValueOf(v)
	typ := val.Type()
	var table string
	var cols []string
	var args []any

	if isMapStringInterface(typ) {
		if o.table == "" {
			return nil, fmt.Errorf("Table option required for map writes")
		}
		table = o.table
		iter := val.MapRange()
		for iter.Next() {
			col := iter.Key().String()
			if len(o.cols) > 0 {
				if _, ok := o.cols[col]; !ok {
					continue
				}
			}
			if _, ok := o.omit[col]; ok {
				continue
			}
			cols = append(cols, col)
			args = append(args, iter.Value().Interface())
		}
	} else if typ.Kind() == reflect.Struct {
		table = o.table
		if table == "" {
			table = model.TableName(v)
		}
		meta, err := getTypeMeta(typ)
		if err != nil {
			return nil, err
		}
		for _, fm := range meta.FieldsByName {
			if fm.Readonly {
				continue
			}
			if len(o.cols) > 0 {
				if _, ok := o.cols[fm.Col]; !ok {
					continue
				}
			}
			if _, ok := o.omit[fm.Col]; ok {
				continue
			}
			fv := val.FieldByIndex(fm.IndexPath)
			if fm.OmitEmpty && fv.IsZero() {
				continue
			}
			cols = append(cols, fm.Col)
			args = append(args, fv.Interface())
		}
	} else {
		return nil, fmt.Errorf("unsupported type %s", typ)
	}
	if len(cols) == 0 {
		return nil, fmt.Errorf("no columns to insert")
	}
	ph := buildPlaceholders(db.drv.Dialect, len(cols), 1)
	quotedCols := make([]string, len(cols))
	for i, c := range cols {
		quotedCols[i] = quote(db.drv.Dialect, c)
	}
	sqlStr := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", quote(db.drv.Dialect, table), strings.Join(quotedCols, ", "), strings.Join(ph, ", "))
	sqlStr, err := appendReturningClause(db.drv.Dialect, sqlStr, o.returning)
	if err != nil {
		return nil, err
	}
	if len(o.returning) > 0 {
		return execReturningRows(ctx, db, sqlStr, args...)
	}
	return db.execContextTrusted(ctx, sqlStr, args...)
}

// Update updates record v.
func Update[T any](ctx context.Context, db *DB, v T, opts ...WriteOpt) (sql.Result, error) {
	o := applyWriteOpts(opts)
	if !o.wherePK {
		return nil, fmt.Errorf("Update[T] without WherePK is not allowed")
	}
	val := reflect.ValueOf(v)
	typ := val.Type()
	var table string
	var setCols []string
	var setArgs []any
	var whereCols []string
	var whereArgs []any

	if isMapStringInterface(typ) {
		if o.table == "" {
			return nil, fmt.Errorf("Table option required for map writes")
		}
		if len(o.pkCols) == 0 {
			return nil, fmt.Errorf("WherePK for map writes requires PK columns via PK option")
		}
		table = o.table
		iter := val.MapRange()
		seen := make(map[string]bool)
		for iter.Next() {
			col := iter.Key().String()
			v := iter.Value()
			seen[col] = true
			if o.isPK(col) {
				whereCols = append(whereCols, col)
				whereArgs = append(whereArgs, v.Interface())
				continue
			}
			if len(o.cols) > 0 {
				if _, ok := o.cols[col]; !ok {
					continue
				}
			}
			if _, ok := o.omit[col]; ok {
				continue
			}
			setCols = append(setCols, col)
			setArgs = append(setArgs, v.Interface())
		}
		for pk := range o.pkCols {
			if !seen[pk] {
				return nil, fmt.Errorf("WherePK requires pk column %s", pk)
			}
		}
	} else if typ.Kind() == reflect.Struct {
		table = o.table
		if table == "" {
			table = model.TableName(v)
		}
		meta, err := getTypeMeta(typ)
		if err != nil {
			return nil, err
		}
		for _, fm := range meta.FieldsByName {
			fv := val.FieldByIndex(fm.IndexPath)
			if fm.PK {
				whereCols = append(whereCols, fm.Col)
				whereArgs = append(whereArgs, fv.Interface())
				continue
			}
			if fm.Readonly {
				continue
			}
			if len(o.cols) > 0 {
				if _, ok := o.cols[fm.Col]; !ok {
					continue
				}
			}
			if _, ok := o.omit[fm.Col]; ok {
				continue
			}
			if fm.OmitEmpty && fv.IsZero() {
				continue
			}
			setCols = append(setCols, fm.Col)
			setArgs = append(setArgs, fv.Interface())
		}
	} else {
		return nil, fmt.Errorf("unsupported type %s", typ)
	}
	if len(whereCols) == 0 {
		return nil, fmt.Errorf("WherePK requires pk values")
	}
	if len(setCols) == 0 {
		return nil, fmt.Errorf("no columns to update")
	}
	setParts := make([]string, len(setCols))
	for i, col := range setCols {
		setParts[i] = fmt.Sprintf("%s=%s", quote(db.drv.Dialect, col), db.drv.Dialect.Placeholder(i+1))
	}
	whereParts := make([]string, len(whereCols))
	for i, col := range whereCols {
		whereParts[i] = fmt.Sprintf("%s=%s", quote(db.drv.Dialect, col), db.drv.Dialect.Placeholder(len(setArgs)+i+1))
	}
	args := append(append([]any(nil), setArgs...), whereArgs...)
	sqlStr := fmt.Sprintf("UPDATE %s SET %s WHERE %s", quote(db.drv.Dialect, table), strings.Join(setParts, ", "), strings.Join(whereParts, " AND "))
	sqlStr, err := appendReturningClause(db.drv.Dialect, sqlStr, o.returning)
	if err != nil {
		return nil, err
	}
	if len(o.returning) > 0 {
		return execReturningRows(ctx, db, sqlStr, args...)
	}
	return db.execContextTrusted(ctx, sqlStr, args...)
}

// Upsert inserts or updates v using primary keys.
func Upsert[T any](ctx context.Context, db *DB, v T, opts ...WriteOpt) (sql.Result, error) {
	o := applyWriteOpts(opts)
	if !o.wherePK {
		return nil, fmt.Errorf("Upsert[T] without WherePK is not allowed")
	}
	val := reflect.ValueOf(v)
	typ := val.Type()
	var table string
	var cols []string
	var args []any
	var pkCols []string
	var updateCols []string

	if isMapStringInterface(typ) {
		if o.table == "" {
			return nil, fmt.Errorf("Table option required for map writes")
		}
		if len(o.pkCols) == 0 {
			return nil, fmt.Errorf("WherePK for map writes requires PK columns via PK option")
		}
		table = o.table
		iter := val.MapRange()
		seen := make(map[string]bool)
		for iter.Next() {
			col := iter.Key().String()
			fv := iter.Value().Interface()
			seen[col] = true
			if o.isPK(col) {
				pkCols = append(pkCols, col)
				cols = append(cols, col)
				args = append(args, fv)
				continue
			}
			if len(o.cols) > 0 {
				if _, ok := o.cols[col]; !ok {
					continue
				}
			}
			if _, ok := o.omit[col]; ok {
				continue
			}
			cols = append(cols, col)
			args = append(args, fv)
		}
		for pk := range o.pkCols {
			if !seen[pk] {
				return nil, fmt.Errorf("WherePK requires pk column %s", pk)
			}
		}
		for _, c := range cols {
			if !o.isPK(c) {
				updateCols = append(updateCols, c)
			}
		}
	} else if typ.Kind() == reflect.Struct {
		table = o.table
		if table == "" {
			table = model.TableName(v)
		}
		meta, err := getTypeMeta(typ)
		if err != nil {
			return nil, err
		}
		for _, fm := range meta.FieldsByName {
			fv := val.FieldByIndex(fm.IndexPath)
			if fm.PK {
				pkCols = append(pkCols, fm.Col)
				cols = append(cols, fm.Col)
				args = append(args, fv.Interface())
				continue
			}
			if fm.Readonly {
				continue
			}
			if len(o.cols) > 0 {
				if _, ok := o.cols[fm.Col]; !ok {
					continue
				}
			}
			if _, ok := o.omit[fm.Col]; ok {
				continue
			}
			if fm.OmitEmpty && fv.IsZero() {
				continue
			}
			cols = append(cols, fm.Col)
			args = append(args, fv.Interface())
		}
		for _, c := range cols {
			foundPK := false
			for _, pk := range pkCols {
				if c == pk {
					foundPK = true
					break
				}
			}
			if !foundPK {
				updateCols = append(updateCols, c)
			}
		}
	} else {
		return nil, fmt.Errorf("unsupported type %s", typ)
	}
	if len(pkCols) == 0 {
		return nil, fmt.Errorf("WherePK requires pk values")
	}
	ph := buildPlaceholders(db.drv.Dialect, len(cols), 1)
	quotedCols := make([]string, len(cols))
	for i, c := range cols {
		quotedCols[i] = quote(db.drv.Dialect, c)
	}
	sqlStr := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", quote(db.drv.Dialect, table), strings.Join(quotedCols, ", "), strings.Join(ph, ", "))
	switch db.drv.Dialect.(type) {
	case driver.MySQLDialect:
		if len(updateCols) > 0 {
			assigns := make([]string, len(updateCols))
			for i, c := range updateCols {
				assigns[i] = fmt.Sprintf("%s=VALUES(%s)", quote(db.drv.Dialect, c), quote(db.drv.Dialect, c))
			}
			sqlStr += " ON DUPLICATE KEY UPDATE " + strings.Join(assigns, ", ")
		} else {
			sqlStr = strings.Replace(sqlStr, "INSERT", "INSERT IGNORE", 1)
		}
	case driver.PostgresDialect:
		quotedPK := make([]string, len(pkCols))
		for i, c := range pkCols {
			quotedPK[i] = quote(db.drv.Dialect, c)
		}
		if len(updateCols) > 0 {
			assigns := make([]string, len(updateCols))
			for i, c := range updateCols {
				assigns[i] = fmt.Sprintf("%s=EXCLUDED.%s", quote(db.drv.Dialect, c), quote(db.drv.Dialect, c))
			}
			sqlStr += fmt.Sprintf(" ON CONFLICT (%s) DO UPDATE SET %s", strings.Join(quotedPK, ", "), strings.Join(assigns, ", "))
		} else {
			sqlStr += fmt.Sprintf(" ON CONFLICT (%s) DO NOTHING", strings.Join(quotedPK, ", "))
		}
	default:
		return nil, fmt.Errorf("upsert not supported on dialect: %T", db.drv.Dialect)
	}
	sqlStr, err := appendReturningClause(db.drv.Dialect, sqlStr, o.returning)
	if err != nil {
		return nil, err
	}
	if len(o.returning) > 0 {
		return execReturningRows(ctx, db, sqlStr, args...)
	}
	return db.execContextTrusted(ctx, sqlStr, args...)
}
