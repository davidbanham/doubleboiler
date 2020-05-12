package models

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
			organisationUserFixture(innerUser.ID, innerOrg.ID),
			organisationUserFixture(innerUser2.ID, innerOrg2.ID),
		},
	}
}

func (c *OrganisationUsers) Iter() <-chan model {
	ch := make(chan model)
	go func() {
		for i := 0; i < len((*c)); i++ {
			ch <- &(*c)[i]
		}
		close(ch)
	}()
	return ch
}
