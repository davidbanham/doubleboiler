package models

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/firestore"
	uuid "github.com/satori/go.uuid"
	"google.golang.org/api/iterator"
)

var organisationUsersTable *firestore.CollectionRef

func init() {
	organisationUsersTable = getTable("organisations_users")
}

type OrganisationUser struct {
	ID             string
	UserID         string
	OrganisationID string
	Revision       string
	Roles          Roles
}

type Roles map[string]bool

func (c *OrganisationUser) New(userID, organisationID string, roles Roles) {
	c.ID = uuid.NewV4().String()
	c.UserID = userID
	c.OrganisationID = organisationID
	c.Revision = uuid.NewV4().String()
	c.Roles = roles
}

func (organisationUser *OrganisationUser) Save(ctx context.Context) error {
	doc := organisationUsersTable.Doc(organisationUser.ID)
	_, err := doc.Set(ctx, organisationUser)
	return err
}

func (organisationUser *OrganisationUser) FindByID(ctx context.Context, id string) error {
	docsnap, err := organisationUsersTable.Doc(id).Get(ctx)
	if err != nil {
		return err
	}
	return docsnap.DataTo(&organisationUser)
}

func (organisationUser *OrganisationUser) FindByColumn(ctx context.Context, col, val string) error {
	q := organisationUsersTable.Where(col, "==", val)

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
		if err := doc.DataTo(&organisationUser); err != nil {
			return err
		}
		return nil
	}
	return nil
}

func (organisationUser OrganisationUser) Delete(ctx context.Context) error {
	_, err := organisationUsersTable.Doc(organisationUser.ID).Delete(ctx)
	return err
}

func (organisationUser OrganisationUser) Email(ctx context.Context) (string, error) {
	// FIXME This is hilariously unperformant with the way it's used
	user := User{}
	if err := user.FindByID(ctx, organisationUser.UserID); err != nil {
		return "", err
	}
	return user.Email, nil
}

type OrganisationUsers []OrganisationUser

func (organisationUsers *OrganisationUsers) FindAll(ctx context.Context, q Query, qa ...string) error {
	var iter *firestore.DocumentIterator

	switch q {
	default:
		return fmt.Errorf("Unknown query")
	case ByOrg:
		iter = organisationUsersTable.Where("OrganisationID", "==", qa[0]).Documents(ctx)
	case ByUser:
		iter = organisationUsersTable.Where("UserID", "==", qa[0]).Documents(ctx)
	case All:
		iter = organisationUsersTable.Documents(ctx)
	}

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Failed to iterate: %v", err)
		}
		organisationUser := OrganisationUser{}
		if err := doc.DataTo(&organisationUser); err != nil {
			return err
		}
		(*organisationUsers) = append((*organisationUsers), organisationUser)
	}
	return nil
}
