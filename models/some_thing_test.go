package models

import (
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

func (c *SomeThings) Iter() <-chan model {
	ch := make(chan model)
	go func() {
		for i := 0; i < len((*c).Data); i++ {
			ch <- &(*c).Data[i]
		}
		close(ch)
	}()
	return ch
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
	assert.Nil(t, fix.Save(ctx))
	assert.NotEqual(t, firstRev, fix.Revision)

	closeTx(t, ctx)
}
