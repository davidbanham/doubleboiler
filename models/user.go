package models

import (
	"context"
	"doubleboiler/config"
	"doubleboiler/copy"
	"doubleboiler/util"
	"fmt"
	"log"
	"net/url"
	"strings"

	"cloud.google.com/go/firestore"
	kewpie "github.com/davidbanham/kewpie_go/v3"
	"github.com/davidbanham/notifications"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/iterator"
)

var usersTable *firestore.CollectionRef

func init() {
	usersTable = getTable("users")
}

type User struct {
	ID                    string
	Email                 string
	Password              string
	Admin                 bool
	Verified              bool
	VerificationEmailSent bool
}

func (u *User) New(email, rawpassword string) {
	hash, _ := HashPassword(rawpassword)
	u.ID = uuid.NewV4().String()
	u.Email = strings.ToLower(email)
	u.Password = string(hash)
	u.Verified = false
}

func (user *User) Save(ctx context.Context) error {
	doc := usersTable.Doc(user.ID)
	_, err := doc.Set(ctx, user)
	return err
}

func (user *User) FindByID(ctx context.Context, id string) error {
	docsnap, err := usersTable.Doc(id).Get(ctx)
	if err != nil {
		return err
	}
	return docsnap.DataTo(&user)
}

func (user *User) FindByColumn(ctx context.Context, col, val string) error {
	q := usersTable.Where(col, "==", val)

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
		if err := doc.DataTo(&user); err != nil {
			return err
		}
		return nil
	}
	return nil
}

func (u *User) SendVerificationEmail(ctx context.Context, org Organisation) error {
	if !u.HasEmail() {
		return nil
	}

	expiry := util.CalcExpiry(365)
	token := util.CalcToken(u.Email, expiry)
	escaped := url.QueryEscape(token)
	verificationUrl := fmt.Sprintf("%s/verify?expiry=%s&uid=%s&token=%s", config.URI, expiry, u.ID, escaped)
	emailHTML, emailText := copy.VerificationEmail(verificationUrl, org.Name)

	subject := []string{}

	if org.Name != "" {
		subject = append(subject, org.Name)
	}

	subject = append(subject, "Confirm your %s account")

	fromAddress := fmt.Sprintf("%s <%s>", org.Name, config.SYSTEM_EMAIL_ONLY)

	mail := notifications.Email{
		To:      u.Email,
		From:    fromAddress,
		ReplyTo: config.SYSTEM_EMAIL,
		Text:    emailText,
		HTML:    emailHTML,
		Subject: fmt.Sprintf(strings.Join(subject, " - "), config.NAME),
	}

	task := kewpie.Task{}
	if err := task.Marshal(mail); err != nil {
		return err
	}

	if err := queue.Publish(ctx, config.SEND_EMAIL_QUEUE_NAME, &task); err != nil {
		return err
	}

	u.VerificationEmailSent = true
	return u.Save(ctx)
}

func (u User) HasEmail() bool {
	if u.Email == "" {
		return false
	}

	return true
}

type Users []User

func (users *Users) FindAll(ctx context.Context, q Query, qa ...string) error {
	var iter *firestore.DocumentIterator

	switch q {
	default:
		return fmt.Errorf("Unknown query")
	case All:
		iter = usersTable.Documents(ctx)
	}

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Failed to iterate: %v", err)
		}
		user := User{}
		if err := doc.DataTo(&user); err != nil {
			return err
		}
		(*users) = append((*users), user)
	}
	return nil
}

func HashPassword(rawpassword string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(rawpassword), 10)
	return string(hash), err
}
