package conv

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"
)

// As converts v to the desired type T using reflection.
func As[T any](v any) (T, error) {
	var zero T
	if v == nil {
		return zero, fmt.Errorf("value is nil")
	}
	rv := reflect.ValueOf(v)
	rt := reflect.TypeOf(zero)
	if !rv.Type().ConvertibleTo(rt) {
		return zero, fmt.Errorf("cannot convert %T to %T", v, zero)
	}
	return rv.Convert(rt).Interface().(T), nil
}

// Value returns the given key from m converted to T.
func Value[T any](m map[string]any, key string) (T, error) {
	val, ok := m[key]
	if !ok {
		var zero T
		return zero, fmt.Errorf("key %q not found", key)
	}
	return As[T](val)
}

// MapToStruct copies values from map m to the struct pointed to by dest.
// Keys are matched to struct fields using orm tags or snake_case names.
func MapToStruct(m map[string]any, dest any) error {
	if m == nil {
		return fmt.Errorf("map is nil")
	}
	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return fmt.Errorf("dest must be non-nil pointer to struct")
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("dest must point to struct")
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		col := sf.Tag.Get("db")
		if col == "" || col == "-" {
			col = parseTag(sf.Tag.Get("orm"))
		}
		if col == "" {
			col = toSnake(sf.Name)
		}
		if val, ok := findValue(m, col); ok && val != nil {
			fv := reflect.ValueOf(val)
			f := v.Field(i)
			if fv.Type().ConvertibleTo(f.Type()) {
				f.Set(fv.Convert(f.Type()))
			} else {
				return fmt.Errorf("cannot convert %s to field %s", fv.Type(), sf.Name)
			}
		}
	}
	return nil
}

// MapsToStructs converts a slice of maps to a slice of structs.
func MapsToStructs(src []map[string]any, dest any) error {
	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return fmt.Errorf("dest must be non-nil pointer to slice")
	}
	v = v.Elem()
	if v.Kind() != reflect.Slice {
		return fmt.Errorf("dest must point to slice")
	}
	elemType := v.Type().Elem()
	for _, m := range src {
		elemPtr := reflect.New(elemType)
		if err := MapToStruct(m, elemPtr.Interface()); err != nil {
			return err
		}
		v.Set(reflect.Append(v, elemPtr.Elem()))
	}
	return nil
}

func findValue(m map[string]any, name string) (any, bool) {
	if val, ok := m[name]; ok {
		return val, true
	}
	for k, val := range m {
		if normalizeKey(k) == name {
			return val, true
		}
	}
	return nil, false
}

func normalizeKey(k string) string {
	if idx := strings.LastIndex(k, "."); idx >= 0 {
		k = k[idx+1:]
	}
	return strings.Trim(k, "`")
}

func parseTag(tag string) string {
	for _, part := range strings.Split(tag, ",") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 && kv[0] == "column" {
			return kv[1]
		}
	}
	return ""
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
