package orm

import "database/sql"

// NullString returns a valid sql.NullString.
func NullString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: true}
}

// NullStringPtr returns NULL when value is nil and a valid sql.NullString otherwise.
func NullStringPtr(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	return NullString(*value)
}

// NullStringEmpty returns NULL when value is empty and a valid sql.NullString otherwise.
func NullStringEmpty(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}
	return NullString(value)
}
