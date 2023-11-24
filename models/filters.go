package models

import (
	scumfilter "github.com/davidbanham/scum/filter"
)

type Filter = scumfilter.Filter
type Filters = scumfilter.Filters
type UpdatedBetween = scumfilter.DateFilter
type CreatedBetween = scumfilter.DateFilter
type HasProp = scumfilter.HasProp
type HasPropOpts = scumfilter.HasPropOpts
type Custom = scumfilter.Custom
type FilterSet = scumfilter.FilterSet
type DateFilterOpts = scumfilter.DateFilterOpts

var standardFilters = scumfilter.CommonFilters
