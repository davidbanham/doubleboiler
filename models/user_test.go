package models

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func init() {
	modelsUnderTest = append(modelsUnderTest, userFix())
	modelCollectionsUnderTest = append(modelCollectionsUnderTest, usersFix())
}

func userFixture() (user User) {
	email := fmt.Sprintf("%s@example.com", randString())
	user.New(email, randString())
	return
}

func (User) blank() model {
	return &User{}
}

func (i User) id() string {
	return i.ID
}

func (i *User) nullDynamicValues() {
}

func (Users) tablename() string {
	return "users"
}

func (Users) blank() models {
	return &Users{}
}

func usersFix() modelCollectionFixture {
	return modelCollectionFixture{
		deps: []model{},
		collection: &Users{
			Data: []User{
				userFixture(),
				userFixture(),
			},
		},
	}
}

func userFix() []model {
	fix := userFixture()
	return []model{
		&fix,
	}
}

func (c *Users) Iter() <-chan model {
	ch := make(chan model)
	go func() {
		for i := 0; i < len((*c).Data); i++ {
			ch <- &(*c).Data[i]
		}
		close(ch)
	}()
	return ch
}

func TestUserRevisionCollision(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	fix := userFixture()
	assert.Nil(t, fix.Save(ctx))
	fix.Revision = "yeahnah"
	assert.Error(t, fix.Save(ctx))

	closeTx(t, ctx)
}

func TestUserRevisionChange(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	fix := userFixture()
	defaultRev := fix.Revision
	assert.Nil(t, fix.Save(ctx))
	assert.Equal(t, defaultRev, fix.Revision)
	firstRev := fix.Revision
	assert.Nil(t, fix.Save(ctx))
	assert.NotEqual(t, firstRev, fix.Revision)

	closeTx(t, ctx)
}
