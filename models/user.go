package models

import (
	"context"
	"database/sql"
	"doubleboiler/config"
	"doubleboiler/copy"
	"doubleboiler/util"
	"fmt"
	"net/url"
	"strings"

	kewpie "github.com/davidbanham/kewpie_go/v3"
	"github.com/davidbanham/notifications"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID                    string
	Email                 string
	Password              string
	Admin                 bool
	Verified              bool
	VerificationEmailSent bool
	Revision              string
}

func (u *User) New(email, rawpassword string) {
	hash, _ := HashPassword(rawpassword)
	u.ID = uuid.NewV4().String()
	u.Email = strings.ToLower(email)
	u.Password = string(hash)
	u.Verified = false
	u.Revision = uuid.NewV4().String()
}

func (u *User) Save(ctx context.Context) error {
	db := ctx.Value("tx").(Querier)

	row := db.QueryRowContext(ctx, "INSERT INTO users (id, revision, email, password, verified, verification_email_sent) VALUES ($1, $2, $4, $5, $6, $7) ON CONFLICT (revision) DO UPDATE SET (revision, email, password, verified, verification_email_sent) = ($3, $4, $5, $6, $7) RETURNING revision", u.ID, u.Revision, uuid.NewV4().String(), strings.ToLower(u.Email), u.Password, u.Verified, u.VerificationEmailSent)
	err := row.Scan(&u.Revision)
	if err != nil {
		return err
	}

	task := kewpie.Task{}
	err = task.Marshal(u)
	if err != nil {
		return err
	}

	return nil
}

func (u *User) FindByID(ctx context.Context, id string) error {
	return u.FindByColumn(ctx, "id", id)
}

func (u *User) FindByColumn(ctx context.Context, col, val string) error {
	db := ctx.Value("tx").(Querier)

	err := db.QueryRowContext(ctx, "SELECT id, revision, email, password, admin, verified, verification_email_sent FROM users WHERE "+col+" = $1", val).Scan(&u.ID, &u.Revision, &u.Email, &u.Password, &u.Admin, &u.Verified, &u.VerificationEmailSent)
	if err != nil {
		return err
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

	db := ctx.Value("tx").(Querier)
	_, err := db.ExecContext(ctx, "UPDATE users SET verification_email_sent = true WHERE id = $1", u.ID)
	if err == nil {
		u.VerificationEmailSent = true
	}
	return err
}

func (u User) HasEmail() bool {
	if u.Email == "" {
		return false
	}

	return true
}

type Users struct {
	Data  []User
	Query Query
}

func (users *Users) FindAll(ctx context.Context, q Query) error {
	users.Query = q

	db := ctx.Value("tx").(Querier)

	var rows *sql.Rows
	var err error

	switch v := q.(type) {
	default:
		return fmt.Errorf("Unknown query")
	case All:
		rows, err = db.QueryContext(ctx, "SELECT id, revision, email, password, admin, verified, verification_email_sent FROM users")
	case ByOrg:
		rows, err = db.QueryContext(ctx, "SELECT id, revision, email, password, admin, verified, verification_email_sent FROM users WHERE id IN (SELECT user_id FROM members WHERE organisation_id = $1)", v.ID)
	}

	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		u := User{}

		err = rows.Scan(&u.ID, &u.Revision, &u.Email, &u.Password, &u.Admin, &u.Verified, &u.VerificationEmailSent)
		if err != nil {
			return err
		}
		(*users).Data = append((*users).Data, u)
	}
	return err
}

func HashPassword(rawpassword string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(rawpassword), 10)
	return string(hash), err
}
