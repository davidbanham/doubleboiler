package models

import (
	"fmt"
	"net/url"
	"strconv"
)

type Pagination struct {
	Limit           int
	Skip            int
	DefaultPageSize int
}

func (this Pagination) Values() url.Values {
	return url.Values{
		"limit": []string{strconv.Itoa(this.Limit)},
		"skip":  []string{strconv.Itoa(this.Skip)},
	}
}

func (this Pagination) NextPage() url.Values {
	vals := this.Values()
	vals.Set("skip", strconv.Itoa(this.Skip+this.Limit))
	return vals
}

func (this Pagination) PrevPage() url.Values {
	vals := this.Values()
	prev := this.Skip - this.Limit
	if prev < 0 {
		prev = 0
	}
	vals.Set("skip", strconv.Itoa(prev))
	return vals
}

func (this Pagination) Pages() []int {
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

func (this Pagination) Page() QueryPage {
	pageNum := (this.Skip / this.Limit) + 1
	return this.GivenPage(pageNum)
}

func (this Pagination) GivenPage(pageNum int) QueryPage {
	vals := this.Values()
	vals.Set("skip", strconv.Itoa((pageNum-1)*this.Limit))
	return QueryPage{
		Number: pageNum,
		Values: vals,
	}
}

func (this Pagination) PaginationQuery() string {
	if this.Limit == 0 {
		return ""
	}
	return fmt.Sprintf(" LIMIT %d OFFSET %d", this.Limit, this.Skip)
}

type Gettable interface {
	// For example, url.Values
	Get(string) string
}

func (this *Pagination) Paginate(form Gettable) {
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
