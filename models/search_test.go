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

	fix := someThingFixture(org.ID)
	assert.Nil(t, fix.Save(ctx))

	results := SearchResults{}
	assert.Nil(t, results.FindAll(ctx,
		Criteria{
			Query: ByPhrase{
				OrgID:  fix.OrganisationID,
				Phrase: fix.Name,
				User:   User{Admin: true},
			},
		}))
	assert.GreaterOrEqual(t, len(results.Data), 1)
	assert.Equal(t, fix.ID, results.Data[0].ID)
	assert.Equal(t, "some_things", results.Data[0].Path)
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

	fix := someThingFixture(org.ID)
	fix.Name = uniq
	assert.Nil(t, fix.Save(ctx))

	results := SearchResults{}
	assert.Nil(t, results.FindAll(ctx, Criteria{
		Query: ByPhrase{
			OrgID:  fix.OrganisationID,
			Phrase: uniq,
			User:   User{Admin: true},
		},
	}))
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
		"SomeThings": true,
	}
	assert.Nil(t, filteredResults.FindAll(ctx, Criteria{
		Query: ByPhrase{
			OrgID:        fix.OrganisationID,
			Phrase:       uniq,
			EntityFilter: filter,
			User:         User{Admin: true},
		},
	}))
	assert.Equal(t, 1, len(filteredResults.Data))
	assert.Equal(t, fix.ID, filteredResults.Data[0].ID)

	closeTx(t, ctx)
}

func TestAdminSearch(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	org := organisationFixture()
	org.Save(ctx)

	fix := someThingFixture(org.ID)
	assert.Nil(t, fix.Save(ctx))

	results := SearchResults{}
	assert.Nil(t, results.FindAll(ctx, Criteria{
		Query: ByPhrase{
			OrgID:  fix.OrganisationID,
			Phrase: fix.Name,
			User:   User{Admin: true},
		},
	}))
	assert.GreaterOrEqual(t, len(results.Data), 1)
	assert.Equal(t, fix.ID, results.Data[0].ID)
	assert.Equal(t, "some_things", results.Data[0].Path)
	assert.Contains(t, results.Data[0].Label, fix.Name)

	closeTx(t, ctx)
}

func TestRequiredRoleSearchResults(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	uniq := bandname.Bandname()

	org := organisationFixture()
	org.Name = uniq
	org.Save(ctx)

	fix := someThingFixture(org.ID)
	fix.Name = uniq
	assert.Nil(t, fix.Save(ctx))

	results := SearchResults{}
	assert.Nil(t, results.FindAll(ctx, Criteria{
		Query: ByPhrase{
			OrgID:  fix.OrganisationID,
			Phrase: uniq,
			Roles: Roles{
				Role{
					Name: "dummy",
				},
			},
		},
	}))
	assert.Equal(t, 0, len(results.Data))

	adminResults := SearchResults{}
	assert.Nil(t, adminResults.FindAll(ctx, Criteria{
		Query: ByPhrase{
			OrgID:  fix.OrganisationID,
			Phrase: uniq,
			Roles: Roles{
				ValidRoles["admin"],
			},
		},
	}))
	assert.Equal(t, 2, len(adminResults.Data))

	found := false
	for _, result := range adminResults.Data {
		if result.ID == fix.ID {
			found = true
		}
	}
	assert.True(t, found)

	closeTx(t, ctx)
}
