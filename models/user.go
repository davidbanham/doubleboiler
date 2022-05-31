package models

import (
	"context"
	"database/sql"
	"doubleboiler/config"
	"doubleboiler/copy"
	"doubleboiler/flashes"
	"doubleboiler/util"
	"fmt"
	"net/url"
	"strings"
	"time"

	kewpie "github.com/davidbanham/kewpie_go/v3"
	"github.com/davidbanham/notifications"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/crypto/bcrypt"
)

func init() {
	Searchables = append(Searchables, Searchable{
		Label:      "Users",
		searchFunc: searchUsers,
	})
}

type User struct {
	ID                    string
	Email                 string
	Password              string
	Admin                 bool
	Verified              bool
	VerificationEmailSent bool
	Revision              string
	CreatedAt             time.Time
	UpdatedAt             time.Time
	HasFlashes            bool
	Flashes               flashes.Flashes
}

func (user *User) New(email, rawpassword string) {
	hash, _ := HashPassword(rawpassword)
	user.ID = uuid.NewV4().String()
	user.Email = strings.ToLower(email)
	user.Password = string(hash)
	user.Verified = false
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
}

func (user *User) auditQuery(ctx context.Context, action string) string {
	return auditQuery(ctx, action, "users", user.ID, user.ID)
}

func (user User) PersistFlash(ctx context.Context, flash flashes.Flash) error {
	user.Flashes = append(user.Flashes, flash)
	db := ctx.Value("tx").(Querier)
	_, err := db.ExecContext(ctx, "UPDATE users SET flashes = $2 WHERE id = $1", user.ID, user.Flashes)
	return err
}

func (user User) DeleteFlash(ctx context.Context, flash flashes.Flash) error {
	db := ctx.Value("tx").(Querier)
	_, err := db.ExecContext(ctx, `
UPDATE users
SET flashes = flashes #- coalesce(('{' || (
	SELECT i
		FROM generate_series(0, jsonb_array_length(flashes) - 1) AS i
	 WHERE (flashes->i->'id' = '"`+flash.ID+`"')
) || '}')::text[], '{}')
WHERE id = $1`, user.ID)
	return err
}

func (user *User) FetchFlashes(ctx context.Context) error {
	db := ctx.Value("tx").(Querier)
	return db.QueryRowContext(ctx, `SELECT flashes FROM users WHERE id = $1`, user.ID).Scan(&user.Flashes)
}

func (user *User) Save(ctx context.Context) error {
	db := ctx.Value("tx").(Querier)

	newRev := uuid.NewV4().String()

	result, err := db.ExecContext(ctx, user.auditQuery(ctx, "U")+`INSERT INTO users (
		updated_at,
		id,
		revision,
		email,
		password,
		verified,
		verification_email_sent
	) VALUES (
		now(), $1, $3, $4, $5, $6, $7
	) ON CONFLICT (id) DO UPDATE SET (
		updated_at,
		revision,
		email,
		password,
		verified,
		verification_email_sent
	) = (
		now(), $3, $4, $5, $6, $7
	) WHERE users.revision = $2`,
		user.ID,
		user.Revision,
		newRev,
		strings.ToLower(user.Email),
		user.Password,
		user.Verified,
		user.VerificationEmailSent,
	)
	if err != nil {
		return err
	}
	num, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if num == 0 {
		return ErrWrongRev
	}

	user.Revision = newRev

	return nil
}

func (user *User) FindByID(ctx context.Context, id string) error {
	return user.FindByColumn(ctx, "id", id)
}

func (user *User) FindByColumn(ctx context.Context, col, val string) error {
	db := ctx.Value("tx").(Querier)

	err := db.QueryRowContext(ctx, `SELECT
		id,
		revision,
		created_at,
		updated_at,
		email,
		password,
		admin,
		verified,
		verification_email_sent,
		(jsonb_array_length(COALESCE(flashes, '[]'::jsonb)) > 0) AS has_flashes
	FROM users WHERE `+col+" = $1", val).Scan(&user.ID,
		&user.Revision,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.Email,
		&user.Password,
		&user.Admin,
		&user.Verified,
		&user.VerificationEmailSent,
		&user.HasFlashes,
	)
	if err != nil {
		return err
	}

	return nil
}

func (user *User) SendVerificationEmail(ctx context.Context, org Organisation) error {
	if !user.HasEmail() {
		return nil
	}

	expiry := util.CalcExpiry(365)
	token := util.CalcToken(user.Email, expiry)
	escaped := url.QueryEscape(token)
	verificationUrl := fmt.Sprintf("%s/verify?expiry=%s&uid=%s&token=%s", config.URI, expiry, user.ID, escaped)
	emailHTML, emailText := copy.VerificationEmail(verificationUrl, org.Name)

	subject := []string{}

	if org.Name != "" {
		subject = append(subject, org.Name)
	}

	subject = append(subject, "Confirm your %s account")

	fromAddress := fmt.Sprintf("%s <%s>", org.Name, config.SYSTEM_EMAIL_ONLY)

	mail := notifications.Email{
		To:      user.Email,
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

	if err := config.QUEUE.Publish(ctx, config.SEND_EMAIL_QUEUE_NAME, &task); err != nil {
		return err
	}

	db := ctx.Value("tx").(Querier)
	_, err := db.ExecContext(ctx, "UPDATE users SET verification_email_sent = true WHERE id = $1", user.ID)
	if err == nil {
		user.VerificationEmailSent = true
	}
	return err
}

func (user User) HasEmail() bool {
	if user.Email == "" {
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
		rows, err = db.QueryContext(ctx, `SELECT
		id,
		revision,
		email,
		password,
		admin,
		verified,
		verification_email_sent,
		(jsonb_array_length(COALESCE(flashes, '[]'::jsonb)) > 0) AS has_flashes
		FROM users `+v.Pagination())
	case ByOrg:
		rows, err = db.QueryContext(ctx, `SELECT
		id,
		revision,
		email,
		password,
		admin,
		verified,
		verification_email_sent,
		(jsonb_array_length(COALESCE(flashes, '[]'::jsonb)) > 0) AS has_flashes
		FROM users WHERE id
		IN (SELECT user_id FROM members WHERE organisation_id = $1)
		`+v.Pagination(), v.ID)
	}

	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		user := User{}

		err = rows.Scan(
			&user.ID,
			&user.Revision,
			&user.Email,
			&user.Password,
			&user.Admin,
			&user.Verified,
			&user.VerificationEmailSent,
			&user.HasFlashes,
		)
		if err != nil {
			return err
		}
		(*users).Data = append((*users).Data, user)
	}
	return err
}

func HashPassword(rawpassword string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(rawpassword), 10)
	return string(hash), err
}

func searchUsers(query ByPhrase) string {
	if query.User.Admin {
		return `SELECT
			text 'User' AS entity_type, text 'users' AS uri_path, id AS id, email AS label, 1 AS rank FROM
			users WHERE email = $2`
	} else {
		return ""
	}
}
