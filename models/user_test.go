package models

import (
	"fmt"
)

func init() {
	modelsUnderTest = append(modelsUnderTest, userFix())
	modelCollectionsUnderTest = append(modelCollectionsUnderTest, usersFix())
}

func userFixture() (u User) {
	email := fmt.Sprintf("%s@example.com", randString())
	u.New(email, randString())
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
			userFixture(),
			userFixture(),
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
		for i := 0; i < len((*c)); i++ {
			ch <- &(*c)[i]
		}
		close(ch)
	}()
	return ch
}
