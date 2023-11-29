package models

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
)

func communicationFixture(user User, org Organisation) (comm Communication) {
	comm.New(org.ID, "email", randString())
	comm.UserID = sql.NullString{Valid: true, String: user.ID}
	return
}

func (Communication) blank() model {
	return &Communication{}
}

func (comm Communication) id() string {
	return comm.ID
}

func (Communication) searchCol() string {
	return "id"
}

func (Communication) tablename() string {
	return "communications"
}

func (comm *Communication) nullDynamicValues() {
}

func (Communications) tablename() string {
	return "communications"
}

func (Communications) blank() models {
	return &Communications{}
}

func communicationsFix() modelCollectionFixture {
	org := organisationFixture()
	user := userFixture()

	return modelCollectionFixture{
		deps: []model{&org, &user},
		collection: &Communications{
			Data: []Communication{
				communicationFixture(user, org),
				communicationFixture(user, org),
			},
		},
	}
}

func communicationFix() []model {
	org := organisationFixture()
	user := userFixture()

	fix := communicationFixture(user, org)
	return []model{
		&org,
		&user,
		&fix,
	}
}

func (this Communications) data() []model {
	ret := []model{}
	for _, m := range this.Data {
		ret = append(ret, &m)
	}
	return ret
}

func TestCommunicationRevisionCollision(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	org := organisationFixture()
	org.Save(ctx)
	user := userFixture()
	user.Save(ctx)

	fix := communicationFixture(user, org)
	assert.Nil(t, fix.Save(ctx))
	fix.Revision = "yeahnah"
	assert.Error(t, fix.Save(ctx))

	closeTx(t, ctx)
}

func TestCommunicationRevisionChange(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	org := organisationFixture()
	org.Save(ctx)
	user := userFixture()
	user.Save(ctx)

	fix := communicationFixture(user, org)
	assert.Nil(t, fix.Save(ctx))
	firstRev := fix.Revision
	assert.Nil(t, fix.Save(ctx))
	assert.NotEqual(t, firstRev, fix.Revision)

	closeTx(t, ctx)
}
