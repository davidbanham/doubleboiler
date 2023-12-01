package models

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func init() {
	modelsUnderTest = append(modelsUnderTest, someThingFix())
	modelCollectionsUnderTest = append(modelCollectionsUnderTest, someThingsFix())
}

func someThingFixture(organisationID string) (u SomeThing) {
	u.New(randString(), randString(), organisationID)
	return
}

func (SomeThing) blank() model {
	return &SomeThing{}
}

func (someThing *SomeThing) nullDynamicValues() {
	someThing.CreatedAt = time.Time{}
	someThing.UpdatedAt = time.Time{}
	someThing.Revision = ""
}

func (SomeThing) tablename() string {
	return "some_things"
}

func (someThing SomeThing) id() string {
	return someThing.ID
}

func (SomeThings) tablename() string {
	return "some_things"
}

func (SomeThings) blank() models {
	return &SomeThings{}
}

func someThingsFix() modelCollectionFixture {
	org := organisationFixture()
	return modelCollectionFixture{
		deps: []model{&org},
		collection: &SomeThings{
			Data: []SomeThing{
				someThingFixture(org.ID),
				someThingFixture(org.ID),
			},
		},
	}
}

func someThingFix() []model {
	org := organisationFixture()
	fix := someThingFixture(org.ID)
	return []model{
		&org,
		&fix,
	}
}

func (this SomeThings) data() []model {
	ret := []model{}
	for _, m := range this.Data {
		ret = append(ret, &m)
	}
	return ret
}

func TestSomeThingRevisionCollision(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	org := organisationFixture()
	org.Save(ctx)

	fix := someThingFixture(org.ID)
	assert.Nil(t, fix.Save(ctx))
	fix.Revision = "yeahnah"
	assert.Equal(t, ErrWrongRev, fix.Save(ctx))

	closeTx(t, ctx)
}

func TestSomeThingRevisionChange(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	org := organisationFixture()
	org.Save(ctx)

	fix := someThingFixture(org.ID)
	assert.Nil(t, fix.Save(ctx))
	firstRev := fix.Revision
	firstTS := fix.UpdatedAt
	assert.Nil(t, fix.Save(ctx))
	assert.NotEqual(t, firstRev, fix.Revision)
	found := SomeThing{}
	assert.Nil(t, found.FindByID(ctx, fix.ID))
	assert.NotEqual(t, firstTS, found.UpdatedAt)
	assert.True(t, found.UpdatedAt.After(firstTS))

	closeTx(t, ctx)
}

func TestSomeThingSoftDelete(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	org := organisationFixture()
	org.Save(ctx)

	fix := someThingFixture(org.ID)
	assert.Nil(t, fix.Save(ctx))

	fix2 := someThingFixture(org.ID)
	assert.Nil(t, fix2.Save(ctx))

	found := SomeThings{}
	assert.Nil(t, found.FindAll(ctx, Criteria{Query: &ByOrg{ID: org.ID}}))
	assert.Equal(t, 2, len(found.Data))

	fix2.SoftDeleted = true
	assert.Nil(t, fix2.Save(ctx))

	foundAfter := SomeThings{}
	assert.Nil(t, foundAfter.FindAll(ctx, Criteria{Query: &ByOrg{ID: org.ID}}))
	assert.Equal(t, 1, len(foundAfter.Data))
	assert.Equal(t, fix.ID, foundAfter.Data[0].ID)
	assert.Equal(t, 1, len(foundAfter.Data))

	foundDeleted := SomeThings{}
	assert.Nil(t, foundDeleted.FindAll(ctx, Criteria{Query: &ByOrg{ID: org.ID}, Filters: Filters{foundDeleted.AvailableFilters().ByID()["is-deleted"]}}))
	assert.Equal(t, 1, len(foundDeleted.Data))
	assert.Equal(t, fix2.ID, foundDeleted.Data[0].ID)
	assert.Equal(t, 1, len(foundDeleted.Data))

	closeTx(t, ctx)
}

func TestSomeThingHardDelete(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	org := organisationFixture()
	org.Save(ctx)

	fix := someThingFixture(org.ID)
	assert.Nil(t, fix.Save(ctx))

	found := SomeThing{}
	assert.Nil(t, found.FindByID(ctx, fix.ID))

	assert.Nil(t, fix.HardDelete(ctx))

	notFound := SomeThing{}
	assert.Equal(t, sql.ErrNoRows, notFound.FindByID(ctx, fix.ID))

	closeTx(t, ctx)
}
