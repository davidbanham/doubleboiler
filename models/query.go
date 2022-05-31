package models

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
)

type Querier interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

type Query interface {
	Pagination() string
	Pages() []int
	Page() QueryPage
	Values() url.Values
	NextPage() url.Values
	PrevPage() url.Values
	ActiveFilters() Filters
	CustomFilters() Filters
}

type Paginatable interface {
	Pagination() string
}

type queryBase struct {
	Limit           int
	Skip            int
	DefaultPageSize int
	activeFilters   Filters
	customFilters   Filters
	input           url.Values
}

func (query queryBase) ActiveFilters() Filters {
	return query.activeFilters
}

func (query queryBase) CustomFilters() Filters {
	return query.customFilters
}

func (query *queryBase) FilterFromForm(form url.Values, availableFilters Filters, customFilters ...Filter) {
	availableFiltersByID := append(availableFilters, customFilters...).ByID()
	cfs := Filters{}
	for _, f := range customFilters {
		cfs = append(cfs, f)
	}
	customFiltersByID := cfs.ByID()
	for _, k := range form["filter"] {
		f, ok := availableFiltersByID[k]
		if ok {
			query.activeFilters = append(query.activeFilters, f)
		}
	}
	for _, k := range form["custom-filter"] {
		cf, ok := customFiltersByID[k]
		if ok {
			query.customFilters = append(query.customFilters, cf)
		}
	}
	form.Del("custom-filter")
	query.input = form
}

func (this queryBase) Values() url.Values {
	return url.Values{
		"limit": []string{strconv.Itoa(this.Limit)},
		"skip":  []string{strconv.Itoa(this.Skip)},
	}
}

func (this queryBase) NextPage() url.Values {
	vals := this.Values()
	vals.Set("skip", strconv.Itoa(this.Skip+this.Limit))
	return vals
}

func (this queryBase) PrevPage() url.Values {
	vals := this.Values()
	prev := this.Skip - this.Limit
	if prev < 0 {
		prev = 0
	}
	vals.Set("skip", strconv.Itoa(prev))
	return vals
}

func (this queryBase) Pages() []int {
	if this.Limit != 0 && this.Skip == 0 {
		return []int{1}
	} else if this.Limit == 0 {
		return []int{}
	}
	numPages := this.Skip / this.Limit
	pages := make([]int, numPages)
	i := 0
	for i < numPages {
		pages[i] = i + 1
		i++
	}
	return append(pages, i+1)
}

type QueryPage struct {
	Number int
	Values url.Values
}

func (this queryBase) Page() QueryPage {
	pageNum := (this.Skip / this.Limit) + 1
	return this.GivenPage(pageNum)
}

func (this queryBase) GivenPage(pageNum int) QueryPage {
	vals := this.Values()
	vals.Set("skip", strconv.Itoa((pageNum-1)*this.Limit))
	return QueryPage{
		Number: pageNum,
		Values: vals,
	}
}

func (this queryBase) Pagination() string {
	if this.Limit == 0 {
		return ""
	}
	return fmt.Sprintf(" LIMIT %d OFFSET %d", this.Limit, this.Skip)
}

type Gettable interface {
	// For example, url.Values
	Get(string) string
}

func (this *queryBase) Paginate(form Gettable) {
	limit := form.Get("limit")
	if limit == "" && this.DefaultPageSize > 0 {
		this.Limit = this.DefaultPageSize
		return
	} else {
		parsedLimit, err := strconv.Atoi(limit)
		if err != nil {
			return
		}
		this.Limit = parsedLimit
	}

	skip := form.Get("skip")
	if skip != "" {
		parsedSkip, err := strconv.Atoi(skip)
		if err != nil {
			return
		}
		this.Skip = parsedSkip
	}
}
