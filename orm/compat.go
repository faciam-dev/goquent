package orm

import "context"

// Deprecated: Use SelectOne instead.
func SelectStruct[T any](ctx context.Context, db *DB, q string, args ...any) (T, error) {
	return SelectOne[T](ctx, db, q, args...)
}

// Deprecated: Use SelectAll instead.
func SelectStructs[T any](ctx context.Context, db *DB, q string, args ...any) ([]T, error) {
	return SelectAll[T](ctx, db, q, args...)
}

// Deprecated: Use SelectOne with map type instead.
func (db *DB) SelectMap(ctx context.Context, q string, args ...any) (map[string]any, error) {
	return SelectOne[map[string]any](ctx, db, q, args...)
}

// Deprecated: Use SelectAll with map type instead.
func (db *DB) SelectMaps(ctx context.Context, q string, args ...any) ([]map[string]any, error) {
	return SelectAll[map[string]any](ctx, db, q, args...)
}
