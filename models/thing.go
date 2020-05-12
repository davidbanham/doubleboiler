package models

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/firestore"
	uuid "github.com/satori/go.uuid"
	"google.golang.org/api/iterator"
)

var thingsTable *firestore.CollectionRef

func init() {
	thingsTable = getTable("things")
}

type Thing struct {
	ID             string
	Name           string
	OrganisationID string
}

func (thing *Thing) New(name, organisationID string) {
	thing.ID = uuid.NewV4().String()
	thing.Name = name
	thing.OrganisationID = organisationID
}

func (thing *Thing) Save(ctx context.Context) error {
	doc := thingsTable.Doc(thing.ID)
	_, err := doc.Set(ctx, thing)
	return err
}

func (thing *Thing) FindByID(ctx context.Context, id string) error {
	docsnap, err := thingsTable.Doc(id).Get(ctx)
	if err != nil {
		return err
	}
	return docsnap.DataTo(&thing)
}

func (thing *Thing) FindByColumn(ctx context.Context, col, val string) error {
	q := thingsTable.Where(col, "==", val)

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
		if err := doc.DataTo(&thing); err != nil {
			return err
		}
		return nil
	}
	return nil
}

type Things []Thing

func (things *Things) FindAll(ctx context.Context, q Query, qa ...string) error {
	var iter *firestore.DocumentIterator

	switch q {
	default:
		return fmt.Errorf("Unknown query")
	case ByOrg:
		iter = thingsTable.Where("organisation_id", "==", qa[0]).Documents(ctx)
	case All:
		iter = thingsTable.Documents(ctx)
	}

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Failed to iterate: %v", err)
		}
		thing := Thing{}
		if err := doc.DataTo(&thing); err != nil {
			return err
		}
		(*things) = append((*things), thing)
	}
	return nil
}
