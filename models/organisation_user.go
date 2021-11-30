package models

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	uuid "github.com/satori/go.uuid"
)

type OrganisationUser struct {
	ID             string
	UserID         string
	OrganisationID string
	Email          string
	Revision       string
	Roles          Roles
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Roles map[string]bool

func (roles Roles) Value() (driver.Value, error) {
	return json.Marshal(roles)
}

func (roles *Roles) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &roles)
}

func (c *OrganisationUser) New(userID, organisationID string, roles Roles) {
	c.ID = uuid.NewV4().String()
	c.UserID = userID
	c.OrganisationID = organisationID
	c.Revision = uuid.NewV4().String()
	c.Roles = roles
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()
}

func (orguser *OrganisationUser) auditQuery(ctx context.Context, action string) string {
	return auditQuery(ctx, action, "organisations_users", orguser.ID)
}

func (c *OrganisationUser) Save(ctx context.Context) error {
	db := ctx.Value("tx").(Querier)

	row := db.QueryRowContext(ctx, c.auditQuery(ctx, "U")+"INSERT INTO organisations_users (id, revision, user_id, organisation_id, roles) VALUES ($1, $2, $4, $5, $6) ON CONFLICT (revision) DO UPDATE SET (revision, updated_at, user_id, organisation_id, roles) = ($3, now(), $4, $5, $6) RETURNING revision", c.ID, c.Revision, uuid.NewV4().String(), c.UserID, c.OrganisationID, c.Roles)
	return row.Scan(&c.Revision)
}

func (c *OrganisationUser) FindByID(ctx context.Context, id string) error {
	return c.FindByColumn(ctx, "id", id)
}

func (c *OrganisationUser) FindByColumn(ctx context.Context, col, val string) error {
	db := ctx.Value("tx").(Querier)

	err := db.QueryRowContext(ctx, `SELECT
	organisations_users.id,
	organisations_users.revision,
	organisations_users.created_at,
	organisations_users.updated_at,
	organisations_users.user_id,
	organisations_users.organisation_id,
	organisations_users.roles,
	users.email
	FROM organisations_users
	INNER JOIN users
	ON organisations_users.user_id = users.id
	WHERE organisations_users.id = $1`, val).Scan(
		&c.ID,
		&c.Revision,
		&c.CreatedAt,
		&c.UpdatedAt,
		&c.UserID,
		&c.OrganisationID,
		&c.Roles,
		&c.Email,
	)
	return err
}

func (c OrganisationUser) Delete(ctx context.Context) error {
	db := ctx.Value("tx").(Querier)

	_, err := db.ExecContext(ctx, c.auditQuery(ctx, "D")+"DELETE FROM organisations_users WHERE id = $1 AND revision = $2", c.ID, c.Revision)
	return err
}

type OrganisationUsers struct {
	Data  []OrganisationUser
	Query Query
}

func (organisationusers *OrganisationUsers) FindAll(ctx context.Context, q Query) error {
	organisationusers.Query = q

	db := ctx.Value("tx").(Querier)

	var rows *sql.Rows
	var err error

	switch v := q.(type) {
	default:
		return fmt.Errorf("Unknown query")
	case All:
		rows, err = db.QueryContext(ctx, `SELECT
	organisations_users.id,
	organisations_users.revision,
	organisations_users.created_at,
	organisations_users.updated_at,
	organisations_users.user_id,
	organisations_users.organisation_id,
	organisations_users.roles,
	users.email
	FROM organisations_users
	INNER JOIN users
	ON organisations_users.user_id = users.id`)
		defer rows.Close()
		if err != nil {
			return err
		}
	case ByUser:
		rows, err = db.QueryContext(ctx, `SELECT
	organisations_users.id,
	organisations_users.revision,
	organisations_users.created_at,
	organisations_users.updated_at,
	organisations_users.user_id,
	organisations_users.organisation_id,
	organisations_users.roles,
	users.email
	FROM organisations_users
	INNER JOIN users
	ON organisations_users.user_id = users.id
	WHERE users.id = $1`, v.ID)
		defer rows.Close()
		if err != nil {
			return err
		}
	}

	for rows.Next() {
		ou := OrganisationUser{}
		if err := rows.Scan(
			&ou.ID,
			&ou.Revision,
			&ou.CreatedAt,
			&ou.UpdatedAt,
			&ou.UserID,
			&ou.OrganisationID,
			&ou.Roles,
			&ou.Email,
		); err != nil {
			return err
		}

		(*organisationusers).Data = append((*organisationusers).Data, ou)
	}

	return nil
}
