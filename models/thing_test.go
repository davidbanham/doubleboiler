package models

func init() {
	modelsUnderTest = append(modelsUnderTest, thingFix())
	modelCollectionsUnderTest = append(modelCollectionsUnderTest, thingsFix())
}

func thingFixture(organisationID string) (u Thing) {
	u.New(randString(), organisationID)
	return
}

func (Thing) blank() model {
	return &Thing{}
}

func (i Thing) id() string {
	return i.ID
}

func (i *Thing) nullDynamicValues() {
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
			thingFixture(org.ID),
			thingFixture(org.ID),
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
		for i := 0; i < len((*c)); i++ {
			ch <- &(*c)[i]
		}
		close(ch)
	}()
	return ch
}
