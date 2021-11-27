package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func init() {
	modelsUnderTest = append(modelsUnderTest, thingFix())
	modelCollectionsUnderTest = append(modelCollectionsUnderTest, thingsFix())
}

func thingFixture(organisationID string) (u Thing) {
	u.New(randString(), randString(), organisationID)
	return
}

func (Thing) blank() model {
	return &Thing{}
}

func (thing Thing) id() string {
	return thing.ID
}

func (thing *Thing) nullDynamicValues() {
}

func (Things) tablename() string {
	return "things"
}

func (Things) blank() models {
	return &Things{}
}

func thingsFix() modelCollectionFixture {
	org := organisationFixture()
	return modelCollectionFixture{
		deps: []model{&org},
		collection: &Things{
			Data: []Thing{
				thingFixture(org.ID),
				thingFixture(org.ID),
			},
		},
	}
}

func thingFix() []model {
	org := organisationFixture()
	fix := thingFixture(org.ID)
	return []model{
		&org,
		&fix,
	}
}

func (c *Things) Iter() <-chan model {
	ch := make(chan model)
	go func() {
		for i := 0; i < len((*c).Data); i++ {
			ch <- &(*c).Data[i]
		}
		close(ch)
	}()
	return ch
}

func TestThingRevisionCollision(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	org := organisationFixture()
	org.Save(ctx)

	fix := thingFixture(org.ID)
	assert.Nil(t, fix.Save(ctx))
	fix.Revision = "yeahnah"
	assert.Error(t, fix.Save(ctx))

	closeTx(t, ctx)
}

func TestThingRevisionChange(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	org := organisationFixture()
	org.Save(ctx)

	fix := thingFixture(org.ID)
	defaultRev := fix.Revision
	assert.Nil(t, fix.Save(ctx))
	assert.Equal(t, defaultRev, fix.Revision)
	firstRev := fix.Revision
	assert.Nil(t, fix.Save(ctx))
	assert.NotEqual(t, firstRev, fix.Revision)

	closeTx(t, ctx)
}
