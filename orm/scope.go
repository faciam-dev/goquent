package orm

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/faciam-dev/goquent/orm/query"
)

// Scope applies reusable query mutations for advanced read and write flows.
type Scope func(*query.Query) *query.Query

// ApplyScopes applies scopes to q in order. Nil scopes are ignored.
// If a scope returns nil, the current query is kept.
func ApplyScopes(q *query.Query, scopes ...Scope) *query.Query {
	for _, scope := range scopes {
		if scope == nil {
			continue
		}
		if next := scope(q); next != nil {
			q = next
		}
	}
	return q
}

// ComposeScopes bundles scopes into a single reusable scope.
func ComposeScopes(scopes ...Scope) Scope {
	return func(q *query.Query) *query.Query {
		return ApplyScopes(q, scopes...)
	}
}

func scopedQuery(base *query.Query, scopes ...Scope) (*query.Query, error) {
	if base == nil {
		return nil, fmt.Errorf("base query is nil")
	}
	return ApplyScopes(base, scopes...), nil
}

// SelectOneBy builds a scoped query and scans the first row into T.
func SelectOneBy[T any](ctx context.Context, db *DB, base *query.Query, scopes ...Scope) (T, error) {
	var zero T
	if db == nil {
		return zero, fmt.Errorf("db is nil")
	}
	q, err := scopedQuery(base, scopes...)
	if err != nil {
		return zero, err
	}
	plan, err := q.Plan(ctx)
	if err != nil {
		return zero, err
	}
	if err := query.EnsurePlanExecutable(plan); err != nil {
		return zero, err
	}
	return SelectOne[T](ctx, db.RequireRawApproval("goquent generated scoped query"), plan.SQL, plan.Params...)
}

// SelectAllBy builds a scoped query and scans all rows into []T.
func SelectAllBy[T any](ctx context.Context, db *DB, base *query.Query, scopes ...Scope) ([]T, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	q, err := scopedQuery(base, scopes...)
	if err != nil {
		return nil, err
	}
	plan, err := q.Plan(ctx)
	if err != nil {
		return nil, err
	}
	if err := query.EnsurePlanExecutable(plan); err != nil {
		return nil, err
	}
	return SelectAll[T](ctx, db.RequireRawApproval("goquent generated scoped query"), plan.SQL, plan.Params...)
}

// UpdateBy applies scopes to base and executes an UPDATE using the resulting query.
func UpdateBy(ctx context.Context, base *query.Query, data any, scopes ...Scope) (sql.Result, error) {
	q, err := scopedQuery(base, scopes...)
	if err != nil {
		return nil, err
	}
	return q.WithContext(ctx).Update(data)
}

// DeleteBy applies scopes to base and executes a DELETE using the resulting query.
func DeleteBy(ctx context.Context, base *query.Query, scopes ...Scope) (sql.Result, error) {
	q, err := scopedQuery(base, scopes...)
	if err != nil {
		return nil, err
	}
	return q.WithContext(ctx).Delete()
}
