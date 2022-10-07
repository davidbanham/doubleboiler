package models

import (
	"net/url"
	"strings"
)

type Filter interface {
	Label() string
	query() string
	ID() string
	IsCustom() bool
}

type filterBase struct{}

func (this filterBase) IsCustom() bool {
	return false
}

type Filters []Filter

func (filters Filters) ByID() map[string]Filter {
	ret := map[string]Filter{}
	for _, filter := range filters {
		ret[filter.ID()] = filter
	}
	return ret
}

func (filters Filters) Query() string {
	if len(filters) == 0 {
		return " WHERE true = true "
	}
	fragments := []string{}
	for _, filter := range filters {
		fragments = append(fragments, filter.query())
	}
	return " WHERE " + strings.Join(fragments, " AND ")
}

func (filters *Filters) FromForm(form url.Values, availableFilters Filters, customFilters ...Filter) url.Values {
	activeFilters := Filters{}

	availableFiltersByID := append(availableFilters, customFilters...).ByID()
	cfs := Filters{}
	for _, f := range customFilters {
		cfs = append(cfs, f)
	}
	customFiltersByID := cfs.ByID()
	for _, k := range form["filter"] {
		f, ok := availableFiltersByID[k]
		if ok {
			activeFilters = append(activeFilters, f)
		}
	}
	for _, k := range form["custom-filter"] {
		cf, ok := customFiltersByID[k]
		if ok {
			customFilters = append(customFilters, cf)
		}
	}
	form.Del("custom-filter")
	return form
}
