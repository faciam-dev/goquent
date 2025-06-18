package scanner

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
)

// Struct scans current row into dest struct using column mapping.
func Struct(dest any, rows *sql.Rows) error {
	cols, err := rows.Columns()
	if err != nil {
		return err
	}
	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return fmt.Errorf("dest must be non-nil pointer")
	}
	v = v.Elem()
	fields := make([]any, len(cols))
	for i := range fields {
		fields[i] = new(any)
	}
	if !rows.Next() {
		return sql.ErrNoRows
	}
	if err = rows.Scan(fields...); err != nil {
		return err
	}
	for i, col := range cols {
		f := v.FieldByNameFunc(func(name string) bool { return toSnake(name) == col })
		if f.IsValid() && f.CanSet() {
			val := reflect.ValueOf(fields[i]).Elem().Interface()
			if val != nil {
				fv := reflect.ValueOf(val)
				if fv.Type().ConvertibleTo(f.Type()) {
					f.Set(fv.Convert(f.Type()))
				} else {
					return fmt.Errorf("type mismatch for %s", col)
				}
			}
		}
	}
	return nil
}

// Map scans the current row into a map.
func Map(rows *sql.Rows) (map[string]any, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	vals := make([]any, len(cols))
	for i := range vals {
		vals[i] = new(any)
	}
	if !rows.Next() {
		return nil, sql.ErrNoRows
	}
	if err = rows.Scan(vals...); err != nil {
		return nil, err
	}
	m := make(map[string]any, len(cols))
	for i, c := range cols {
		m[c] = reflect.ValueOf(vals[i]).Elem().Interface()
	}
	return m, nil
}

// Maps scans all remaining rows into slice of maps.
func Maps(rows *sql.Rows) ([]map[string]any, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var list []map[string]any
	for rows.Next() {
		vals := make([]any, len(cols))
		for i := range vals {
			vals[i] = new(any)
		}
		if err := rows.Scan(vals...); err != nil {
			return nil, err
		}
		m := make(map[string]any, len(cols))
		for i, c := range cols {
			m[c] = reflect.ValueOf(vals[i]).Elem().Interface()
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

func toSnake(s string) string {
	var out []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			out = append(out, '_')
		}
		out = append(out, r)
	}
	return strings.ToLower(string(out))
}
