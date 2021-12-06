package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func init() {
	modelsUnderTest = append(modelsUnderTest, organisationFix())
	modelCollectionsUnderTest = append(modelCollectionsUnderTest, organisationsFix())
}

func organisationFixture() (m Organisation) {
	m.New(randString(), "Australia", OrganisationUsers{}, "aud")
	return
}

func (Organisation) blank() model {
	return &Organisation{}
}

func (i Organisation) id() string {
	return i.ID
}

func (i *Organisation) nullDynamicValues() {
	i.CreatedAt = time.Time{}
	i.UpdatedAt = time.Time{}
	i.Users.Query = All{}
}

func (Organisation) tablename() string {
	return "organisations"
}

func (Organisations) tablename() string {
	return "organisations"
}

func (Organisations) blank() models {
	return &Organisations{}
}

func organisationsFix() modelCollectionFixture {
	return modelCollectionFixture{
		deps: []model{},
		collection: &Organisations{
			Data: []Organisation{
				organisationFixture(),
				organisationFixture(),
			},
		},
	}
}

func organisationFix() []model {
	fix := organisationFixture()
	return []model{
		&fix,
	}
}

func (c *Organisations) Iter() <-chan model {
	ch := make(chan model)
	go func() {
		for i := 0; i < len((*c).Data); i++ {
			ch <- &(*c).Data[i]
		}
		close(ch)
	}()
	return ch
}

func TestOrganisationRevisionCollision(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)
	fix := organisationFixture()
	assert.Nil(t, fix.Save(ctx))
	fix.Revision = "yeahnah"
	assert.Error(t, fix.Save(ctx))

	closeTx(t, ctx)
}

func TestOrganisationRevisionChange(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)
	fix := organisationFixture()
	defaultRev := fix.Revision
	assert.Nil(t, fix.Save(ctx))
	assert.Equal(t, defaultRev, fix.Revision)
	firstRev := fix.Revision
	assert.Nil(t, fix.Save(ctx))
	assert.NotEqual(t, firstRev, fix.Revision)

	closeTx(t, ctx)
}
