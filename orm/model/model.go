package model

import (
	"reflect"
	"strings"
	"sync"
	"unicode"
)

// fieldInfo holds mapping metadata.
type fieldInfo struct {
	name  string
	index []int
}

// Map of struct type -> column mappings with concurrency safety.
var cache = struct {
	sync.RWMutex
	m map[reflect.Type][]fieldInfo
}{m: make(map[reflect.Type][]fieldInfo)}

// Columns returns column info for struct type.
func Columns(t reflect.Type) []fieldInfo {
	cache.RLock()
	fi, ok := cache.m[t]
	cache.RUnlock()
	if ok {
		return fi
	}
	var res []fieldInfo
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		col := ""
		if tag := f.Tag.Get("db"); tag != "" && tag != "-" {
			col = tag
		} else if tag := f.Tag.Get("orm"); tag != "" {
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
	cache.Lock()
	defer cache.Unlock()
	cache.m[t] = res
	return res
}

func toSnake(s string) string {
	runes := []rune(s)
	var sb strings.Builder
	for i, r := range runes {
		if i > 0 {
			prev := runes[i-1]
			next := rune(0)
			if i+1 < len(runes) {
				next = runes[i+1]
			}
			if unicode.IsLower(prev) && unicode.IsUpper(r) {
				sb.WriteByte('_')
			} else if unicode.IsUpper(prev) && unicode.IsUpper(r) && next != 0 && unicode.IsLower(next) {
				sb.WriteByte('_')
			}
		}
		sb.WriteRune(unicode.ToLower(r))
	}
	return sb.String()
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
