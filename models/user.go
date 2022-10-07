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
	"github.com/lib/pq"
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

func (users Users) AvailableFilters() Filters {
	return userFilters()
}

func userFilters() Filters {
	return append(standardFilters(),
		HasProp{
			key:   "verification_email_sent",
			value: "true",
			label: "Has Been Invited",
			id:    "user-has-been-invited",
		},
		HasProp{
			key:   "verified",
			value: "true",
			label: "Has Accepted Invite",
			id:    "user-is-verified",
		},
	)
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

	task.Tags.Set("user_id", user.ID)
	task.Tags.Set("organisation_id", org.ID)
	task.Tags.Set("communication_subject", fmt.Sprintf("Account confirmation request"))

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
	Data []User
	baseModel
}

func (this Users) ByID() map[string]User {
	ret := map[string]User{}
	for _, t := range this.Data {
		ret[t.ID] = t
	}
	return ret
}

func (users *Users) FindAll(ctx context.Context, criteria Criteria) error {
	users.Criteria = criteria

	db := ctx.Value("tx").(Querier)

	var rows *sql.Rows
	var err error

	switch v := criteria.Query.(type) {
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
		FROM users
		`+criteria.Filters.Query()+`
		ORDER BY email`+criteria.Pagination.PaginationQuery())
	case ByIDs:
		rows, err = db.QueryContext(ctx, `SELECT
		id,
		revision,
		email,
		password,
		admin,
		verified,
		verification_email_sent,
		(jsonb_array_length(COALESCE(flashes, '[]'::jsonb)) > 0) AS has_flashes
		FROM users
		`+criteria.Filters.Query()+`
		AND id = ANY ($1)
		ORDER BY email`+criteria.Pagination.PaginationQuery(), pq.Array(v.IDs))
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
		FROM users
		`+criteria.Filters.Query()+`
		AND id IN (SELECT user_id FROM organisations_users WHERE organisation_id = $1)
		ORDER BY email`+criteria.Pagination.PaginationQuery(), v.ID)
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

func searchUsers(criteria Criteria) string {
	switch v := criteria.Query.(type) {
	default:
		return ""
	case ByPhrase:
		if v.User.Admin {
			return `SELECT
			text 'User' AS entity_type, text 'users' AS uri_path, id AS id, email AS label, 1 AS rank FROM
			users WHERE email = $2`
		} else {
			return ""
		}
	}
}
