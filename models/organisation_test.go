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
	m.New(randString(), "Australia")
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
	i.Revision = ""
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

func (this Organisations) data() []model {
	ret := []model{}
	for _, m := range this.Data {
		ret = append(ret, &m)
	}
	return ret
}

func TestOrganisationRevisionCollision(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)
	fix := organisationFixture()
	assert.Nil(t, fix.Save(ctx))
	fix.Revision = "yeahnah"
	assert.Equal(t, ErrWrongRev, fix.Save(ctx))

	closeTx(t, ctx)
}

func TestOrganisationRevisionChange(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)
	fix := organisationFixture()
	assert.Nil(t, fix.Save(ctx))
	firstRev := fix.Revision
	assert.Nil(t, fix.Save(ctx))
	assert.NotEqual(t, firstRev, fix.Revision)

	closeTx(t, ctx)
}
