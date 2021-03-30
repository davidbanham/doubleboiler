package models

type All struct{ queryBase }
type OrganisationsContainingUser struct {
	ID string
	queryBase
}
type ByOrg struct {
	ID string
	queryBase
}
type ByCol struct{ queryBase }
type ByUser struct {
	ID string
	queryBase
}
