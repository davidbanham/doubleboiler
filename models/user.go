package models

import (
	"context"
	"database/sql"
	"doubleboiler/config"
	"doubleboiler/copy"
	"doubleboiler/flashes"
	"doubleboiler/util"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	kewpie "github.com/davidbanham/kewpie_go/v3"
	"github.com/davidbanham/notifications"
	"github.com/davidbanham/scum/search"
	uuid "github.com/satori/go.uuid"
)

type User struct {
	ID                    string
	Email                 string
	Password              string
	SuperAdmin            bool
	Verified              bool
	VerificationEmailSent bool
	Revision              string
	CreatedAt             time.Time
	UpdatedAt             time.Time
	HasFlashes            bool
	Flashes               flashes.Flashes
}

func (this *User) colmap() *Colmap {
	return &Colmap{
		"id":                      &this.ID,
		"email":                   &this.Email,
		"password":                &this.Password,
		"admin":                   &this.SuperAdmin,
		"verified":                &this.Verified,
		"verification_email_sent": &this.VerificationEmailSent,
		"revision":                &this.Revision,
		"created_at":              &this.CreatedAt,
		"updated_at":              &this.UpdatedAt,
		"has_flashes":             &this.HasFlashes,
		"flashes":                 &this.Flashes,
	}
}

func (user *User) New(email, rawpassword string) {
	hash, _ := util.HashPassword(rawpassword)
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

func (this *User) Save(ctx context.Context) error {
	colmap := this.colmap().Delete("has_flashes", "flashes")
	q, props, newRev := StandardSave("users", colmap, this.auditQuery(ctx, "U"))

	if err := ExecSave(ctx, q, props); err != nil {
		return err
	}

	this.Revision = newRev

	return nil
}

func (user *User) FindByID(ctx context.Context, id string) error {
	return user.FindByColumn(ctx, "id", id)
}

func (this *User) FindByColumn(ctx context.Context, col, val string) error {
	colmap := this.colmap().Delete("flashes")
	q, props := StandardFindByColumn("users", colmap, col)

	return StandardExecFindByColumn(ctx, q, val, props)
}

func (user *User) SendVerificationEmail(ctx context.Context, org Organisation) error {
	if !user.HasEmail() {
		return nil
	}

	token := util.CalcToken(user.Email, 365, config.SECRET)
	escaped := url.QueryEscape(token.String())
	verificationUrl := fmt.Sprintf("%s/verify?expiry=%s&uid=%s&token=%s", config.URI, token.ExpiryString(), user.ID, escaped)
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

func (user User) Avatar() string {
	return fmt.Sprintf("https://secure.gravatar.com/avatar/%s?s=70&d=mp", util.Hash(user.Email))
}

func (user User) Label() string {
	return user.Email
}

type Users struct {
	Data     []User
	Criteria Criteria
}

func (this Users) colmap() *Colmap {
	r := User{}
	return r.colmap()
}

func (users Users) AvailableFilters() Filters {
	viaEmail := HasProp{}
	if err := viaEmail.Hydrate(HasPropOpts{
		Label: "Has Been Invited",
		ID:    "user-has-been-invited",
		Table: "users",
		Col:   "verification_email_sent",
		Value: "true",
	}); err != nil {
		log.Fatal(err)
	}
	verified := HasProp{}
	if err := verified.Hydrate(HasPropOpts{
		Label: "Has Accepted Invite",
		ID:    "user-is-verified",
		Table: "users",
		Col:   "verified",
		Value: "true",
	}); err != nil {
		log.Fatal(err)
	}
	return append(standardFilters("users"), &viaEmail, &verified)
}

func (Users) Searchable() Searchable {
	return Searchable{
		EntityType: "User",
		Label:      "email",
		Path:       "users",
		Tablename:  "users",
		Permitted:  search.BasicRoleCheck("superadmin"),
	}
}

func (this Users) ByID() map[string]User {
	ret := map[string]User{}
	for _, t := range this.Data {
		ret[t.ID] = t
	}
	return ret
}

func (this *Users) FindAll(ctx context.Context, criteria Criteria) error {
	this.Criteria = criteria

	db := ctx.Value("tx").(Querier)

	colmap := this.colmap().Delete("flashes")
	cols, _ := colmap.Split()

	var rows *sql.Rows
	var err error

	switch v := criteria.Query.(type) {
	default:
		return ErrInvalidQuery{Query: v, Model: "users"}
	case custom:
		switch v := criteria.customQuery.(type) {
		default:
			return ErrInvalidQuery{Query: v, Model: "users"}
		}
	case Query:
		rows, err = db.QueryContext(ctx, v.Construct(cols, "users", criteria.Filters, criteria.Pagination, "email"), v.Args()...)
	}
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		user := User{}
		props := user.colmap().ByKeys(cols)
		if err := rows.Scan(props...); err != nil {
			return err
		}
		(*this).Data = append((*this).Data, user)
	}

	return err
}
