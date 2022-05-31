package models

type Filter interface {
	Label() string
	query() string
	ID() string
}

type Filters []Filter

func (filters Filters) ByID() map[string]Filter {
	ret := map[string]Filter{}
	for _, filter := range filters {
		ret[filter.ID()] = filter
	}
	return ret
}
