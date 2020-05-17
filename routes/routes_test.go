package routes

import (
	"context"
	"doubleboiler/models"
	m "doubleboiler/models"
	"testing"

	bn "github.com/davidbanham/bandname_go"
)

func init() {
	Init()
}

func bandname() string {
	return bn.Bandname()
}

func getCtx(t *testing.T) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "organisations", models.Organisations{})
	ctx = context.WithValue(ctx, "organisation_users", models.OrganisationUsers{})
	return ctx
}

func contextifyOrgAdmin(ctx context.Context, org models.Organisation) context.Context {
	ctx = context.WithValue(ctx, "organisation_users", m.OrganisationUsers{
		models.OrganisationUser{
			OrganisationID: org.ID, Roles: models.Roles{
				"admin": true,
			},
		},
	})
	ctx = context.WithValue(ctx, "organisations", m.Organisations{org})
	ctx = context.WithValue(ctx, "target_org", org.ID)
	return ctx
}

func closeTx(t *testing.T, ctx context.Context) {
	return
}
