package model

import (
	"reflect"
	"strings"
)

// fieldInfo holds mapping metadata.
type fieldInfo struct {
	name  string
	index []int
}

// Map of struct type -> column mappings.
var cache = map[reflect.Type][]fieldInfo{}

// Columns returns column info for struct type.
func Columns(t reflect.Type) []fieldInfo {
	if fi, ok := cache[t]; ok {
		return fi
	}
	var res []fieldInfo
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("orm")
		col := ""
		if tag != "" {
			parts := strings.Split(tag, ",")
			for _, p := range parts {
				kv := strings.SplitN(p, "=", 2)
				if kv[0] == "column" && len(kv) > 1 {
					col = kv[1]
				}
			}
		}
		if col == "" {
			col = toSnake(f.Name)
		}
		res = append(res, fieldInfo{name: col, index: f.Index})
	}
	cache[t] = res
	return res
}

func toSnake(s string) string {
	var sb strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			sb.WriteByte('_')
		}
		sb.WriteRune(r)
	}
	return strings.ToLower(sb.String())
}

// TableName returns default table name for struct value.
type tableNamer interface{ TableName() string }

// TableName returns the table name for the given value.
func TableName(v any) string {
	if tn, ok := v.(tableNamer); ok {
		return tn.TableName()
	}
	t := reflect.Indirect(reflect.ValueOf(v)).Type()
	return toSnake(t.Name()) + "s"
}
