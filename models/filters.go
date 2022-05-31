package models

import (
	"fmt"
	"strings"
	"time"
)

var standards Filters

func init() {
	standards = Filters{
		UpdatedWithin{
			interval: "14 days",
			id:       "updated-within-fortnight",
			label:    "Updated within the last fortnight",
		},
		UpdatedWithin{
			interval: "1 day",
			id:       "updated-within-24hrs",
			label:    "Updated Within 24 hours",
		},
		CreatedWithin{
			interval: "14 days",
			id:       "created-within-fortnight",
			label:    "Created within the last fortnight",
		},
		CreatedWithin{
			interval: "1 day",
			id:       "created-within-24hrs",
			label:    "Created Within 24 hours",
		},
	}
}

func standardFilters() Filters {
	return standards
}

type HasProp struct {
	key   string
	value string
	id    string
	label string
}

func (this HasProp) query() string {
	return fmt.Sprintf("%s = %s", this.key, this.value)
}

func (this HasProp) Label() string {
	return this.label
}

func (this HasProp) ID() string {
	return this.id
}

type Custom struct {
	Key         string
	Values      []string
	CustomID    string
	CustomLabel string
}

func (this Custom) query() string {
	vals := []string{}
	for _, val := range this.Values {
		vals = append(vals, fmt.Sprintf("'%s'", val))
	}
	return fmt.Sprintf("%s::text = ANY (ARRAY[%s])", this.Key, strings.Join(vals, ", "))
}

func (this Custom) Label() string {
	return this.CustomLabel
}

func (this Custom) ID() string {
	return this.CustomID
}

type UpdatedWithin struct {
	interval string
	id       string
	label    string
}

func (this UpdatedWithin) query() string {
	return fmt.Sprintf("age(updated_at) < interval '%s'", this.interval)
}

func (this UpdatedWithin) Label() string {
	return this.label
}

func (this UpdatedWithin) ID() string {
	return this.id
}

type UpdatedAfter struct {
	TS    time.Time
	id    string
	label string
}

func (this UpdatedAfter) query() string {
	return fmt.Sprintf("updated_at > timestamptz '%s'", this.TS.Format(time.RFC3339))
}

func (this UpdatedAfter) Label() string {
	return fmt.Sprintf("From %s", this.TS.Format(time.RFC822))
}

func (this UpdatedAfter) ID() string {
	return fmt.Sprintf("filter-updated-after-%s", this.TS.Format("2006-01-02"))
}

type UpdatedBefore struct {
	TS    time.Time
	id    string
	label string
}

func (this UpdatedBefore) query() string {
	return fmt.Sprintf("updated_at < timestamptz '%s'", this.TS.Format(time.RFC3339))
}

func (this UpdatedBefore) Label() string {
	return fmt.Sprintf("To %s", this.TS.Format(time.RFC822))
}

func (this UpdatedBefore) ID() string {
	return fmt.Sprintf("filter-updated-before-%s", this.TS.Format("2006-01-02"))
}

type CreatedWithin struct {
	interval string
	id       string
	label    string
}

func (this CreatedWithin) query() string {
	return fmt.Sprintf("age(created_at) < interval '%s'", this.interval)
}

func (this CreatedWithin) Label() string {
	return this.label
}

func (this CreatedWithin) ID() string {
	return this.id
}

type CreatedAfter struct {
	TS    time.Time
	id    string
	label string
}

func (this CreatedAfter) query() string {
	return fmt.Sprintf("created_at > timestamptz '%s'", this.TS.Format(time.RFC3339))
}

func (this CreatedAfter) Label() string {
	return fmt.Sprintf("From %s", this.TS.Format(time.RFC822))
}

func (this CreatedAfter) ID() string {
	return fmt.Sprintf("filter-created-after-%s", this.TS.Format("2006-01-02"))
}

type CreatedBefore struct {
	TS    time.Time
	id    string
	label string
}

func (this CreatedBefore) query() string {
	return fmt.Sprintf("created_at < timestamptz '%s'", this.TS.Format(time.RFC3339))
}

func (this CreatedBefore) Label() string {
	return fmt.Sprintf("To %s", this.TS.Format(time.RFC822))
}

func (this CreatedBefore) ID() string {
	return fmt.Sprintf("filter-created-before-%s", this.TS.Format("2006-01-02"))
}
