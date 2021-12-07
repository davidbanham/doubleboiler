package routes

import (
	"context"
	"database/sql"
	"doubleboiler/config"
	"doubleboiler/models"
	"fmt"
	"testing"

	bn "github.com/davidbanham/bandname_go"
	"github.com/stretchr/testify/assert"
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
	tx, err := config.Db.BeginTx(ctx, nil)
	assert.Nil(t, err)
	return context.WithValue(ctx, "tx", tx)
}

func contextifyOrgAdmin(ctx context.Context, org models.Organisation) context.Context {
	ctx = context.WithValue(ctx, "organisation_users", models.OrganisationUsers{
		Data: []models.OrganisationUser{
			models.OrganisationUser{
				OrganisationID: org.ID,
				Roles: models.Roles{
					models.ValidRoles["admin"],
				},
			},
		},
	})
	ctx = context.WithValue(ctx, "organisations", models.Organisations{
		Data: []models.Organisation{
			org,
		},
	})
	ctx = context.WithValue(ctx, "target_org", org.ID)
	return ctx
}

func closeTx(t *testing.T, ctx context.Context) {
	db := ctx.Value("tx")
	switch v := db.(type) {
	case *sql.Tx:
		t.Log("Rolling back")
		v.Rollback()
	case models.Querier:
		t.Log("No tx")
		return
	default:
		t.Log(fmt.Errorf("Unknown db type"))
		t.FailNow()
	}
}
