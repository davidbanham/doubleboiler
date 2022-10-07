package models

import (
	"context"
	"database/sql"
	"net/url"
)

type Querier interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

type Query interface {
}

type Paginatable interface {
	Pages() []int
	Page() QueryPage
	Values() url.Values
	NextPage() url.Values
	PrevPage() url.Values
}

type queryBase struct {
	activeFilters Filters
	customFilters Filters
	input         url.Values
}
