package models

import (
	"testing"

	bandname "github.com/davidbanham/bandname_go"
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
	assert.GreaterOrEqual(t, 1, len(results.Data))
	assert.Equal(t, fix.ID, results.Data[0].ID)
	assert.Equal(t, "things", results.Data[0].Path)
	assert.Contains(t, results.Data[0].Label, fix.Name)

	closeTx(t, ctx)
}

func TestEntityFilteredSearchResults(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	uniq := bandname.Bandname()

	org := organisationFixture()
	org.Name = uniq
	org.Save(ctx)

	fix := thingFixture(org.ID)
	fix.Name = uniq
	assert.Nil(t, fix.Save(ctx))

	results := SearchResults{}
	assert.Nil(t, results.FindAll(ctx, ByPhrase{OrgID: fix.OrganisationID, Phrase: uniq}))
	assert.Equal(t, 2, len(results.Data))

	found := false
	for _, result := range results.Data {
		if result.ID == fix.ID {
			found = true
		}
	}
	assert.True(t, found)

	filteredResults := SearchResults{}
	filter := map[string]bool{
		"Things": true,
	}
	assert.Nil(t, filteredResults.FindAll(ctx, ByPhrase{OrgID: fix.OrganisationID, Phrase: uniq, EntityFilter: filter}))
	assert.Equal(t, 1, len(filteredResults.Data))
	assert.Equal(t, fix.ID, filteredResults.Data[0].ID)

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
	assert.Equal(t, fix.ID, results.Data[0].ID)
	assert.Equal(t, "things", results.Data[0].Path)
	assert.Contains(t, results.Data[0].Label, fix.Name)

	closeTx(t, ctx)
}
