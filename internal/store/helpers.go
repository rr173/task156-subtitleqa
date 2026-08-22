package store

import "database/sql"

// sqlNullString maps an empty string to SQL NULL, preserving the distinction
// between "no speaker" and "empty speaker id" in the schema.
func sqlNullString(v string) sql.NullString {
	if v == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: v, Valid: true}
}
