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
type ByPhrase struct {
	Phrase       string
	OrgID        string
	User         User
	Roles        Roles
	EntityFilter map[string]bool
	queryBase
}
type ByEntityID struct {
	EntityID string
	queryBase
}
type ByIDs struct {
	IDs []string
	queryBase
}
