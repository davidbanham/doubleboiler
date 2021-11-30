package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func init() {
	modelsUnderTest = append(modelsUnderTest, organisationUserFix())
	modelCollectionsUnderTest = append(modelCollectionsUnderTest, organisationUsersFix())
}

func organisationUserFixture(userID, organisationID string) (c OrganisationUser) {
	c.New(
		userID,
		organisationID,
		Roles{"admin": true},
	)
	return
}

func (OrganisationUser) blank() model {
	return &OrganisationUser{}
}

func (c OrganisationUser) id() string {
	return c.ID
}

func (i *OrganisationUser) nullDynamicValues() {
	i.Email = ""
	i.CreatedAt = time.Time{}
	i.UpdatedAt = time.Time{}
}

func organisationUserFix() []model {
	innerUser := userFixture()
	innerOrg := organisationFixture()

	innerOrgUserFix := organisationUserFixture(innerUser.ID, innerOrg.ID)

	return []model{
		&innerUser,
		&innerOrg,
		&innerOrgUserFix,
	}
}

func (OrganisationUsers) tablename() string {
	return "organisations_users"
}

func (OrganisationUsers) blank() models {
	return &OrganisationUsers{}
}

func organisationUsersFix() modelCollectionFixture {
	innerUser := userFixture()
	innerUser2 := userFixture()
	innerOrg := organisationFixture()
	innerOrg2 := organisationFixture()

	return modelCollectionFixture{
		deps: []model{&innerUser, &innerOrg, &innerUser2, &innerOrg2},
		collection: &OrganisationUsers{
			Data: []OrganisationUser{
				organisationUserFixture(innerUser.ID, innerOrg.ID),
				organisationUserFixture(innerUser2.ID, innerOrg2.ID),
			},
		},
	}
}

func (c *OrganisationUsers) Iter() <-chan model {
	ch := make(chan model)
	go func() {
		for i := 0; i < len((*c).Data); i++ {
			ch <- &(*c).Data[i]
		}
		close(ch)
	}()
	return ch
}

func TestOrganisationUserRevisionCollision(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	user := userFixture()
	user.Save(ctx)
	org := organisationFixture()
	org.Save(ctx)

	fix := organisationUserFixture(user.ID, org.ID)
	assert.Nil(t, fix.Save(ctx))
	fix.Revision = "yeahnah"
	assert.Error(t, fix.Save(ctx))

	closeTx(t, ctx)
}

func TestOrganisationUserRevisionChange(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	user := userFixture()
	user.Save(ctx)
	org := organisationFixture()
	org.Save(ctx)

	fix := organisationUserFixture(user.ID, org.ID)
	defaultRev := fix.Revision
	assert.Nil(t, fix.Save(ctx))
	assert.Equal(t, defaultRev, fix.Revision)
	firstRev := fix.Revision
	assert.Nil(t, fix.Save(ctx))
	assert.NotEqual(t, firstRev, fix.Revision)

	closeTx(t, ctx)
}
