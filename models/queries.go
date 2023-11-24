package models

import (
	scumquery "github.com/davidbanham/scum/query"
)

type All = scumquery.All
type ByOrg = scumquery.ByOrg
type ByUser = scumquery.ByUser
type ByIDs = scumquery.ByIDs

type OrganisationsContainingUser struct {
	ID string
}

type ByEntityID struct {
	EntityID string
}
type ByItem struct {
	ID string
}
type ByGroup struct {
	ID string
}
