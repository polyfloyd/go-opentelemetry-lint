// Code generated by MockGen. DO NOT EDIT.
// Source: example.go

// Package lint is a generated GoMock package.
package lint

import (
	"context"
	"database/sql"
)

func MissingSpan(ctx context.Context, db *sql.DB) error {
	row := db.QueryRowContext(ctx, `SELECT * FROM sample_text`)
	return row.Err()
}
