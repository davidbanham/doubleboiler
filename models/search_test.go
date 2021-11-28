package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearchResults(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	org := organisationFixture()
	org.Save(ctx)

	fix := thingFixture(org.ID)
	assert.Nil(t, fix.Save(ctx))

	results := SearchResults{}
	assert.Nil(t, results.FindAll(ctx, ByPhrase{OrgID: fix.OrganisationID, Phrase: fix.Name}))
	assert.GreaterOrEqual(t, len(results.Data), 1)
	assert.Equal(t, results.Data[0].ID, fix.ID)
	assert.Equal(t, results.Data[0].Path, "things")
	assert.Contains(t, results.Data[0].Label, fix.Name)

	closeTx(t, ctx)
}

func TestAdminSearch(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	org := organisationFixture()
	org.Save(ctx)

	fix := thingFixture(org.ID)
	assert.Nil(t, fix.Save(ctx))

	results := SearchResults{}
	assert.Nil(t, results.FindAll(ctx, ByPhrase{OrgID: fix.OrganisationID, Phrase: fix.Name, User: User{Admin: true}}))
	assert.GreaterOrEqual(t, len(results.Data), 1)
	assert.Equal(t, results.Data[0].ID, fix.ID)
	assert.Equal(t, results.Data[0].Path, "things")
	assert.Contains(t, results.Data[0].Label, fix.Name)

	closeTx(t, ctx)
}
