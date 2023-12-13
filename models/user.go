package models

import (
	"context"
	"database/sql"
	"doubleboiler/config"
	"doubleboiler/copy"
	"doubleboiler/flashes"
	"doubleboiler/logger"
	"doubleboiler/util"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	bandname "github.com/davidbanham/bandname_go"
	kewpie "github.com/davidbanham/kewpie_go/v3"
	"github.com/davidbanham/notifications"
	"github.com/davidbanham/scum/search"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
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
	totpSecret            sql.NullString
	TOTPActive            bool
	recoveryCodes         NullStringList
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
		"totp_active":             &this.TOTPActive,
		"totp_secret":             &this.totpSecret,
		"recovery_codes":          &this.recoveryCodes,
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

func (user User) PersistFlashes(ctx context.Context) (context.Context, error) {
	persisted := flashes.Flashes{}
	for _, flash := range user.Flashes {
		if flash.Persistent {
			persisted = append(persisted, flash)
		}
	}

	if len(persisted) > 0 {
		db := ctx.Value("tx").(Querier)
		if _, err := db.ExecContext(ctx, "UPDATE users SET flashes = $2 WHERE id = $1", user.ID, persisted); err != nil {
			return ctx, err
		}
	}

	return context.WithValue(ctx, "user", user), nil
}

func (user *User) PersistFlash(ctx context.Context, flash flashes.Flash) (context.Context, error) {
	if err := user.FetchFlashes(ctx); err != nil {
		return ctx, err
	}

	(*user).Flashes.Add(flash)
	(*user).HasFlashes = len(user.Flashes) > 0

	return user.PersistFlashes(ctx)
}

func (user User) DeleteFlash(ctx context.Context, id string) error {
	db := ctx.Value("tx").(Querier)
	_, err := db.ExecContext(ctx, `
UPDATE users
SET flashes = flashes #- coalesce(('{' || (
	SELECT i
		FROM generate_series(0, jsonb_array_length(flashes) - 1) AS i
	 WHERE (flashes->i->'id' = '"`+id+`"')
) || '}')::text[], '{}')
WHERE id = $1`, user.ID)
	return err
}

func (user *User) FetchFlashes(ctx context.Context) error {
	db := ctx.Value("tx").(Querier)
	persisted := flashes.Flashes{}
	if err := db.QueryRowContext(ctx, `SELECT flashes FROM users WHERE id = $1`, user.ID).Scan(&persisted); err != nil {
		return err
	}
	for _, existing := range user.Flashes {
		if !existing.Persistent {
			persisted = append(persisted, existing)
		}
	}
	(*user).Flashes = persisted
	return nil
}

func (user *User) Validate2FA(ctx context.Context, code, recoveryCode string) (bool, error) {
	db := ctx.Value("tx").(Querier)

	var failureCount int
	var lastFailure time.Time
	if err := db.QueryRowContext(ctx, "SELECT totp_failure_count, totp_last_failure FROM users WHERE id = $1", user.ID).Scan(&failureCount, &lastFailure); err != nil {
		return false, err
	} else {
		if failureCount > 3 && lastFailure.After(time.Now().Add(-60*time.Second)) {
			return false, ClientSafeError{"Too many attempts. Please wait 60 seconds before trying again."}
		}
	}

	if recoveryCode != "" && code == "" {
		for _, target := range user.recoveryCodes.Strings {
			matched := false
			if target == recoveryCode {
				snipped := NullStringList{Valid: true}
				for _, inner := range user.recoveryCodes.Strings {
					if inner != target {
						snipped.Strings = append(snipped.Strings, inner)
					}
				}
				user.recoveryCodes = snipped
				if err := user.saveRecoveryCodes(ctx); err != nil {
					return false, err
				}
				matched = true

				payload := notifications.Email{
					To:      user.Email,
					From:    fmt.Sprintf("%s <%s>", config.NAME, config.SYSTEM_EMAIL_ONLY),
					ReplyTo: config.SYSTEM_EMAIL_ONLY,
					Text:    fmt.Sprintf("The account under this email address: %s has been accessed using a one-time recovery code. If this action was not initiated by the rightful account owner, please contact us immediately.", user.Email),
					Subject: fmt.Sprintf("%s - Recovery Code Used", config.NAME),
				}

				task := kewpie.Task{}
				if err := task.Marshal(payload); err != nil {
					logger.Log(ctx, logger.Error, "marshaling recovery code used email", err)
					return false, err
				}

				if err := config.QUEUE.Buffer(ctx, config.SEND_EMAIL_QUEUE_NAME, &task); err != nil {
					logger.Log(ctx, logger.Error, "buffering recovery code used email", err)
					return false, err
				}
			}

			return matched, nil
		}
	}

	valid := totp.Validate(code, user.totpSecret.String)
	if !valid {
		if !user.TOTPActive {
			return valid, nil
		} else {
			if _, err := db.ExecContext(ctx, "UPDATE users SET totp_failure_count = $2, totp_last_failure = NOW() WHERE id = $1", user.ID, failureCount+1); err != nil {
				return false, err
			}

			return valid, nil
		}
	} else {
		if !user.TOTPActive {
			if _, err := db.ExecContext(ctx, "UPDATE users SET totp_active = true WHERE id = $1", user.ID); err != nil {
				return false, err
			} else {
				user.TOTPActive = true
			}
		}

		if _, err := db.ExecContext(ctx, "UPDATE users SET totp_failure_count = 0 WHERE id = $1", user.ID); err != nil {
			return false, err
		}

		return valid, nil
	}
}

func (user *User) Generate2FA(ctx context.Context, code, recoveryCode string) (*otp.Key, error) {
	if user.TOTPActive {
		valid, err := user.Validate2FA(ctx, code, recoveryCode)
		if err != nil {
			return nil, err
		}
		if !valid {
			if code == "" && recoveryCode != "" {
				return nil, ClientSafeError{"Provided 2FA recovery phrase did not match. Each phrase may only be used once before it is invalidated."}
			} else {
				return nil, ClientSafeError{"Provided 2FA code did not match expected value"}
			}
		}
	}

	db := ctx.Value("tx").(Querier)

	if key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      config.DOMAIN,
		AccountName: user.Email,
	}); err != nil {
		return nil, err
	} else {
		user.totpSecret.Valid = true
		user.totpSecret.String = key.Secret()
		if _, err := db.ExecContext(ctx, "UPDATE users SET totp_secret = $2 WHERE id = $1", user.ID, key.Secret()); err != nil {
			return nil, err
		}

		return key, nil
	}
}

func (user User) saveRecoveryCodes(ctx context.Context) error {
	db := ctx.Value("tx").(Querier)

	if _, err := db.ExecContext(ctx, "UPDATE users SET recovery_codes = $2 WHERE id = $1", user.ID, user.recoveryCodes); err != nil {
		return err
	}
	return nil
}

func (user *User) generateRecoveryCodes(ctx context.Context) ([]string, error) {
	fresh := NullStringList{Valid: true}

	for i := 0; i < 10; i++ {
		code := bandname.Bandname() + " " + uuid.NewV4().String()
		fresh.Strings = append(fresh.Strings, code)
	}
	user.recoveryCodes = fresh

	if err := user.saveRecoveryCodes(ctx); err != nil {
		return []string{}, err
	}

	return user.recoveryCodes.Strings, nil
}

func (user *User) GenerateRecoveryCodes(ctx context.Context, code string) ([]string, error) {
	ok, err := user.Validate2FA(ctx, code, "")
	if err != nil {
		return []string{}, err
	}
	if !ok {
		return []string{}, ClientSafeError{"Provided 2FA code did not match expected value"}
	}

	return user.generateRecoveryCodes(ctx)
}

func (user *User) GenerateRecoveryCodesBypassCheck(ctx context.Context) ([]string, error) {
	return user.generateRecoveryCodes(ctx)
}

func (user *User) Disable2FA(ctx context.Context) error {
	db := ctx.Value("tx").(Querier)
	if _, err := db.ExecContext(ctx, "UPDATE users SET totp_active = false WHERE id = $1", user.ID); err != nil {
		return err
	}

	payload := notifications.Email{
		To:      user.Email,
		From:    fmt.Sprintf("%s <%s>", config.NAME, config.SYSTEM_EMAIL_ONLY),
		ReplyTo: config.SYSTEM_EMAIL_ONLY,
		Text:    fmt.Sprintf("2 Factor authentication has been removed from the user account with this email: %s. If this action was not initiated by the rightful account owner, please contact us immediately.", user.Email),
		Subject: fmt.Sprintf("%s - 2 Factor Authentication Disabled", config.NAME),
	}

	task := kewpie.Task{}
	if err := task.Marshal(payload); err != nil {
		logger.Log(ctx, logger.Error, "marshaling 2fa disabled email", err)
		return err
	}

	if err := config.QUEUE.Buffer(ctx, config.SEND_EMAIL_QUEUE_NAME, &task); err != nil {
		logger.Log(ctx, logger.Error, "buffering 2fa disabled email", err)
		return err
	}

	return nil
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

func (user User) SendEmailChangedNotification(ctx context.Context, newEmail string) error {
	emailHTML, emailText := copy.EmailChangedEmail(newEmail, user.Email)

	subject := fmt.Sprintf("%s email changed", config.NAME)

	recipients := []string{newEmail, user.Email}

	for _, recipient := range recipients {
		mail := notifications.Email{
			To:      recipient,
			From:    config.SYSTEM_EMAIL,
			ReplyTo: config.SUPPORT_EMAIL,
			Text:    emailText,
			HTML:    emailHTML,
			Subject: subject,
		}

		task := kewpie.Task{}
		if err := task.Marshal(mail); err != nil {
			return err
		}

		if err := config.QUEUE.Publish(ctx, config.SEND_EMAIL_QUEUE_NAME, &task); err != nil {
			return err
		}
	}
	return nil
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
