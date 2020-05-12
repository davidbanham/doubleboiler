package models

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/firestore"
	uuid "github.com/satori/go.uuid"
	"google.golang.org/api/iterator"
)

var organisationsTable *firestore.CollectionRef

func init() {
	organisationsTable = getTable("organisations")
}

type Organisation struct {
	ID      string
	Name    string
	Country string
	Users   []string
}

func (o *Organisation) New(name, country string, users []string, currency string) {
	o.ID = uuid.NewV4().String()
	o.Users = users
	o.Name = name
	o.Country = country
}

func (organisation *Organisation) Save(ctx context.Context) error {
	doc := organisationsTable.Doc(organisation.ID)
	_, err := doc.Set(ctx, organisation)
	return err
}

func (organisation *Organisation) FindByColumn(ctx context.Context, col, val string) error {
	q := organisationsTable.Where(col, "==", val)

	iter := q.Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		if err := doc.DataTo(&organisation); err != nil {
			return err
		}
		if organisation.Users == nil {
			organisation.Users = []string{}
		}
		return nil
	}
	return nil
}

func (organisation *Organisation) FindByID(ctx context.Context, id string) error {
	docsnap, err := organisationsTable.Doc(id).Get(ctx)
	if err != nil {
		return err
	}
	if err := docsnap.DataTo(&organisation); err != nil {
		return err
	}
	if organisation.Users == nil {
		organisation.Users = []string{}
	}
	return nil
}

type Organisations []Organisation

func (organisations *Organisations) FindAll(ctx context.Context, q Query, qa ...string) error {
	var iter *firestore.DocumentIterator

	switch q {
	default:
		return fmt.Errorf("Unknown query")
	case OrganisationsContainingUser:
		iter = organisationsTable.Where("users", "array-contains", qa[0]).Documents(ctx)
	case All:
		iter = organisationsTable.Documents(ctx)
	}

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Failed to iterate: %v", err)
		}
		organisation := Organisation{}
		if err := doc.DataTo(&organisation); err != nil {
			return err
		}
		if organisation.Users == nil {
			organisation.Users = []string{}
		}
		(*organisations) = append((*organisations), organisation)
	}
	return nil
}
