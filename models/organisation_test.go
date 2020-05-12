package models

func init() {
	modelsUnderTest = append(modelsUnderTest, organisationFix())
	modelCollectionsUnderTest = append(modelCollectionsUnderTest, organisationsFix())
}

func organisationFixture() (m Organisation) {
	m.New(randString(), "Australia", []string{}, "aud")
	return
}

func (Organisation) blank() model {
	return &Organisation{}
}

func (i Organisation) id() string {
	return i.ID
}

func (i *Organisation) nullDynamicValues() {
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
			organisationFixture(),
			organisationFixture(),
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
		for i := 0; i < len((*c)); i++ {
			ch <- &(*c)[i]
		}
		close(ch)
	}()
	return ch
}
