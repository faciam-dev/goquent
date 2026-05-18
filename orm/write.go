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
	cols               map[string]struct{}
	omit               map[string]struct{}
	wherePK            bool
	returning          []string
	table              string
	pkCols             map[string]struct{}
	conflictCols       []string
	conflictWhere      string
	conflictConstraint string
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

// ConflictColumns sets the conflict target columns for Upsert.
func ConflictColumns(cols ...string) WriteOpt {
	return func(o *writeOptions) { o.conflictCols = append([]string(nil), cols...) }
}

// ConflictWhere adds a Postgres partial-index predicate to the conflict target.
func ConflictWhere(predicate string) WriteOpt {
	return func(o *writeOptions) { o.conflictWhere = predicate }
}

// ConflictConstraint sets a Postgres named constraint as the conflict target.
func ConflictConstraint(name string) WriteOpt {
	return func(o *writeOptions) { o.conflictConstraint = name }
}

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

func (o *writeOptions) hasConflictTarget() bool {
	return len(o.conflictCols) > 0 || strings.TrimSpace(o.conflictConstraint) != ""
}

func (o *writeOptions) isConflictColumn(col string) bool {
	for _, c := range o.conflictCols {
		if c == col {
			return true
		}
	}
	return false
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

func queryReturningOne[T any](ctx context.Context, db *DB, sqlStr string, args ...any) (T, error) {
	var zero T
	rows, err := db.queryContextTrusted(ctx, sqlStr, args...)
	if err != nil {
		return zero, err
	}
	defer rows.Close()
	return scanRowsOne[T](db, rows)
}

func ensureReturningColumns[T any](o *writeOptions) error {
	if len(o.returning) > 0 {
		return nil
	}
	cols, err := returningColumnsForQuery[T]()
	if err != nil {
		return err
	}
	o.returning = cols
	return nil
}

func returningColumnsForQuery[T any]() ([]string, error) {
	var zero T
	typ := reflect.TypeOf(zero)
	if typ == nil {
		return nil, fmt.Errorf("Returning columns are required for untyped return values")
	}
	if isMapStringInterface(typ) {
		return nil, fmt.Errorf("Returning columns are required for map return values")
	}
	cols, err := structColumnNames(typ)
	if err != nil {
		return nil, err
	}
	if len(cols) == 0 {
		return nil, fmt.Errorf("no columns to return")
	}
	return cols, nil
}

func execWriteStatement(ctx context.Context, db *DB, sqlStr string, args []any, returning bool) (sql.Result, error) {
	if returning {
		return execReturningRows(ctx, db, sqlStr, args...)
	}
	return db.execContextTrusted(ctx, sqlStr, args...)
}

func validateWriteRawSQLFragment(raw string) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fmt.Errorf("goquent: raw SQL fragment is required")
	}
	if strings.ContainsAny(trimmed, ";\x00") ||
		strings.Contains(trimmed, "--") ||
		strings.Contains(trimmed, "/*") ||
		strings.Contains(trimmed, "*/") {
		return fmt.Errorf("goquent: raw SQL fragment contains a statement separator or comment")
	}
	upper := strings.ToUpper(trimmed)
	for _, token := range []string{"ALTER", "CREATE", "DELETE", "DROP", "GRANT", "INSERT", "REVOKE", "TRUNCATE", "UPDATE"} {
		if containsSQLWord(upper, token) {
			return fmt.Errorf("goquent: raw SQL fragment contains disallowed SQL token %q", token)
		}
	}
	return nil
}

// Insert inserts v into its table.
func Insert[T any](ctx context.Context, db *DB, v T, opts ...WriteOpt) (sql.Result, error) {
	o := applyWriteOpts(opts)
	sqlStr, args, err := buildInsertStatement(db, v, o)
	if err != nil {
		return nil, err
	}
	return execWriteStatement(ctx, db, sqlStr, args, len(o.returning) > 0)
}

// InsertReturning inserts v and scans the Postgres RETURNING row into T.
func InsertReturning[T any, V any](ctx context.Context, db *DB, v V, opts ...WriteOpt) (T, error) {
	var zero T
	o := applyWriteOpts(opts)
	if err := ensureReturningColumns[T](o); err != nil {
		return zero, err
	}
	sqlStr, args, err := buildInsertStatement(db, v, o)
	if err != nil {
		return zero, err
	}
	return queryReturningOne[T](ctx, db, sqlStr, args...)
}

func buildInsertStatement(db *DB, v any, o *writeOptions) (string, []any, error) {
	val := reflect.ValueOf(v)
	typ := val.Type()
	var table string
	var cols []string
	var args []any

	if isMapStringInterface(typ) {
		if o.table == "" {
			return "", nil, fmt.Errorf("Table option required for map writes")
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
			return "", nil, err
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
		return "", nil, fmt.Errorf("unsupported type %s", typ)
	}
	if len(cols) == 0 {
		return "", nil, fmt.Errorf("no columns to insert")
	}
	ph := buildPlaceholders(db.drv.Dialect, len(cols), 1)
	quotedCols := make([]string, len(cols))
	for i, c := range cols {
		quotedCols[i] = quote(db.drv.Dialect, c)
	}
	sqlStr := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", quote(db.drv.Dialect, table), strings.Join(quotedCols, ", "), strings.Join(ph, ", "))
	sqlStr, err := appendReturningClause(db.drv.Dialect, sqlStr, o.returning)
	if err != nil {
		return "", nil, err
	}
	return sqlStr, args, nil
}

// Update updates record v.
func Update[T any](ctx context.Context, db *DB, v T, opts ...WriteOpt) (sql.Result, error) {
	o := applyWriteOpts(opts)
	sqlStr, args, err := buildUpdateStatement(db, v, o)
	if err != nil {
		return nil, err
	}
	return execWriteStatement(ctx, db, sqlStr, args, len(o.returning) > 0)
}

// UpdateReturning updates v and scans the Postgres RETURNING row into T.
func UpdateReturning[T any, V any](ctx context.Context, db *DB, v V, opts ...WriteOpt) (T, error) {
	var zero T
	o := applyWriteOpts(opts)
	if err := ensureReturningColumns[T](o); err != nil {
		return zero, err
	}
	sqlStr, args, err := buildUpdateStatement(db, v, o)
	if err != nil {
		return zero, err
	}
	return queryReturningOne[T](ctx, db, sqlStr, args...)
}

func buildUpdateStatement(db *DB, v any, o *writeOptions) (string, []any, error) {
	if !o.wherePK {
		return "", nil, fmt.Errorf("Update[T] without WherePK is not allowed")
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
			return "", nil, fmt.Errorf("Table option required for map writes")
		}
		if len(o.pkCols) == 0 {
			return "", nil, fmt.Errorf("WherePK for map writes requires PK columns via PK option")
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
				return "", nil, fmt.Errorf("WherePK requires pk column %s", pk)
			}
		}
	} else if typ.Kind() == reflect.Struct {
		table = o.table
		if table == "" {
			table = model.TableName(v)
		}
		meta, err := getTypeMeta(typ)
		if err != nil {
			return "", nil, err
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
		return "", nil, fmt.Errorf("unsupported type %s", typ)
	}
	if len(whereCols) == 0 {
		return "", nil, fmt.Errorf("WherePK requires pk values")
	}
	if len(setCols) == 0 {
		return "", nil, fmt.Errorf("no columns to update")
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
		return "", nil, err
	}
	return sqlStr, args, nil
}

// Upsert inserts or updates v using primary keys.
func Upsert[T any](ctx context.Context, db *DB, v T, opts ...WriteOpt) (sql.Result, error) {
	o := applyWriteOpts(opts)
	sqlStr, args, err := buildUpsertStatement(db, v, o)
	if err != nil {
		return nil, err
	}
	return execWriteStatement(ctx, db, sqlStr, args, len(o.returning) > 0)
}

// UpsertReturning upserts v and scans the Postgres RETURNING row into T.
func UpsertReturning[T any, V any](ctx context.Context, db *DB, v V, opts ...WriteOpt) (T, error) {
	var zero T
	o := applyWriteOpts(opts)
	if err := ensureReturningColumns[T](o); err != nil {
		return zero, err
	}
	sqlStr, args, err := buildUpsertStatement(db, v, o)
	if err != nil {
		return zero, err
	}
	return queryReturningOne[T](ctx, db, sqlStr, args...)
}

func buildUpsertStatement(db *DB, v any, o *writeOptions) (string, []any, error) {
	if !o.wherePK && !o.hasConflictTarget() {
		return "", nil, fmt.Errorf("Upsert[T] requires WherePK, ConflictColumns, or ConflictConstraint")
	}
	val := reflect.ValueOf(v)
	typ := val.Type()
	var table string
	var cols []string
	var args []any
	var pkCols []string

	if isMapStringInterface(typ) {
		if o.table == "" {
			return "", nil, fmt.Errorf("Table option required for map writes")
		}
		if o.wherePK && len(o.pkCols) == 0 {
			return "", nil, fmt.Errorf("WherePK for map writes requires PK columns via PK option")
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
			if o.isConflictColumn(col) {
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
		if o.wherePK {
			for pk := range o.pkCols {
				if !seen[pk] {
					return "", nil, fmt.Errorf("WherePK requires pk column %s", pk)
				}
			}
		}
	} else if typ.Kind() == reflect.Struct {
		table = o.table
		if table == "" {
			table = model.TableName(v)
		}
		meta, err := getTypeMeta(typ)
		if err != nil {
			return "", nil, err
		}
		for _, fm := range meta.FieldsByName {
			fv := val.FieldByIndex(fm.IndexPath)
			if fm.PK {
				pkCols = append(pkCols, fm.Col)
				cols = append(cols, fm.Col)
				args = append(args, fv.Interface())
				continue
			}
			if o.isConflictColumn(fm.Col) {
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
	} else {
		return "", nil, fmt.Errorf("unsupported type %s", typ)
	}
	if o.wherePK && len(pkCols) == 0 {
		return "", nil, fmt.Errorf("WherePK requires pk values")
	}
	if len(cols) == 0 {
		return "", nil, fmt.Errorf("no columns to insert")
	}
	if err := ensureConflictColumnsPresent(o.conflictCols, cols); err != nil {
		return "", nil, err
	}
	ph := buildPlaceholders(db.drv.Dialect, len(cols), 1)
	quotedCols := make([]string, len(cols))
	for i, c := range cols {
		quotedCols[i] = quote(db.drv.Dialect, c)
	}
	sqlStr := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", quote(db.drv.Dialect, table), strings.Join(quotedCols, ", "), strings.Join(ph, ", "))
	targetCols := conflictTargetColumns(o, pkCols)
	updateCols := upsertUpdateColumns(cols, targetCols)
	switch db.drv.Dialect.(type) {
	case driver.MySQLDialect:
		if strings.TrimSpace(o.conflictWhere) != "" || strings.TrimSpace(o.conflictConstraint) != "" {
			return "", nil, fmt.Errorf("ConflictWhere and ConflictConstraint are not supported on dialect: %T", db.drv.Dialect)
		}
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
		target, err := postgresConflictTarget(db.drv.Dialect, targetCols, o)
		if err != nil {
			return "", nil, err
		}
		if len(updateCols) > 0 {
			assigns := make([]string, len(updateCols))
			for i, c := range updateCols {
				assigns[i] = fmt.Sprintf("%s=EXCLUDED.%s", quote(db.drv.Dialect, c), quote(db.drv.Dialect, c))
			}
			sqlStr += fmt.Sprintf(" ON CONFLICT %s DO UPDATE SET %s", target, strings.Join(assigns, ", "))
		} else {
			sqlStr += fmt.Sprintf(" ON CONFLICT %s DO NOTHING", target)
		}
	default:
		return "", nil, fmt.Errorf("upsert not supported on dialect: %T", db.drv.Dialect)
	}
	sqlStr, err := appendReturningClause(db.drv.Dialect, sqlStr, o.returning)
	if err != nil {
		return "", nil, err
	}
	return sqlStr, args, nil
}

func conflictTargetColumns(o *writeOptions, pkCols []string) []string {
	if len(o.conflictCols) > 0 {
		return append([]string(nil), o.conflictCols...)
	}
	return append([]string(nil), pkCols...)
}

func upsertUpdateColumns(cols []string, targetCols []string) []string {
	target := make(map[string]struct{}, len(targetCols))
	for _, col := range targetCols {
		target[col] = struct{}{}
	}
	updateCols := make([]string, 0, len(cols))
	for _, col := range cols {
		if _, ok := target[col]; ok {
			continue
		}
		updateCols = append(updateCols, col)
	}
	return updateCols
}

func ensureConflictColumnsPresent(targetCols []string, cols []string) error {
	if len(targetCols) == 0 {
		return nil
	}
	present := make(map[string]struct{}, len(cols))
	for _, col := range cols {
		present[col] = struct{}{}
	}
	for _, col := range targetCols {
		if _, ok := present[col]; !ok {
			return fmt.Errorf("ConflictColumns requires column %s", col)
		}
	}
	return nil
}

func postgresConflictTarget(d driver.Dialect, cols []string, o *writeOptions) (string, error) {
	constraint := strings.TrimSpace(o.conflictConstraint)
	predicate := strings.TrimSpace(o.conflictWhere)
	if constraint != "" {
		if len(o.conflictCols) > 0 {
			return "", fmt.Errorf("ConflictConstraint cannot be combined with ConflictColumns")
		}
		if predicate != "" {
			return "", fmt.Errorf("ConflictConstraint cannot be combined with ConflictWhere")
		}
		return "ON CONSTRAINT " + quote(d, constraint), nil
	}
	if len(cols) == 0 {
		return "", fmt.Errorf("Postgres upsert requires ConflictColumns or WherePK primary key columns")
	}
	quoted := make([]string, len(cols))
	for i, col := range cols {
		quoted[i] = quote(d, col)
	}
	target := "(" + strings.Join(quoted, ", ") + ")"
	if predicate != "" {
		if err := validateWriteRawSQLFragment(predicate); err != nil {
			return "", err
		}
		target += " WHERE " + predicate
	}
	return target, nil
}

func containsSQLWord(upperSQL, token string) bool {
	for i := 0; i+len(token) <= len(upperSQL); i++ {
		if upperSQL[i:i+len(token)] != token {
			continue
		}
		beforeOK := i == 0 || !isSQLWordByte(upperSQL[i-1])
		after := i + len(token)
		afterOK := after >= len(upperSQL) || !isSQLWordByte(upperSQL[after])
		if beforeOK && afterOK {
			return true
		}
	}
	return false
}

func isSQLWordByte(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}
