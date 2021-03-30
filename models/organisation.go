package models

import (
	"context"
	"database/sql"
	"fmt"

	uuid "github.com/satori/go.uuid"
)

type Organisation struct {
	ID       string
	Name     string
	Country  string
	Users    []OrganisationUser
	Revision string
}

func (o *Organisation) New(name, country string, users []OrganisationUser, currency string) {
	o.ID = uuid.NewV4().String()
	o.Users = users
	o.Name = name
	o.Country = country
	o.Revision = uuid.NewV4().String()
}

func (o *Organisation) Save(ctx context.Context) error {
	db := ctx.Value("tx").(Querier)

	row := db.QueryRowContext(ctx, `INSERT INTO organisations (
		id,
		revision,
		name,
		country
	) VALUES ($1, $2, $4, $5) ON CONFLICT (revision) DO UPDATE SET (
		revision,
		name,
		country
	) = ($3, $4, $5) RETURNING revision`,
		o.ID,
		o.Revision,
		uuid.NewV4().String(),
		o.Name,
		o.Country,
	)

	err := row.Scan(&o.Revision)
	return err
}

func (o *Organisation) FindByColumn(ctx context.Context, col, val string) error {
	db := ctx.Value("tx").(Querier)

	err := db.QueryRowContext(ctx, `SELECT
	id,
	revision,
	name,
	country
	FROM organisations WHERE `+col+` = $1`, val).Scan(
		&o.ID,
		&o.Revision,
		&o.Name,
		&o.Country,
	)

	if err != nil {
		return err
	}

	o.Users = []OrganisationUser{}

	rows, err := db.QueryContext(ctx, `SELECT
	organisations_users.id,
	organisations_users.revision,
	organisations_users.user_id,
	organisations_users.organisation_id,
	users.email
	FROM organisations_users
	INNER JOIN users
	ON organisations_users.user_id = users.id
	WHERE organisations_users.organisation_id = $1`, o.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		u := OrganisationUser{}
		err = rows.Scan(&u.ID, &u.Revision, &u.UserID, &u.OrganisationID, &u.Email)
		if err != nil {
			return err
		}
		o.Users = append(o.Users, u)
	}

	return err
}

func (o *Organisation) FindByID(ctx context.Context, id string) error {
	return o.FindByColumn(ctx, "id", id)
}

type Organisations struct {
	Data  []Organisation
	Query Query
}

func (organisations *Organisations) FindAll(ctx context.Context, q Query, qa ...string) error {
	organisations.Query = q

	db := ctx.Value("tx").(Querier)

	var rows *sql.Rows
	var err error

	switch q.(type) {
	default:
		return fmt.Errorf("Unknown query")
	case All:
		rows, err = db.QueryContext(ctx, "SELECT id, revision, name, country FROM organisations")
		if err != nil {
			return err
		}
		defer rows.Close()
	case OrganisationsContainingUser:
		rows, err = db.QueryContext(ctx, `
		SELECT organisations.id, organisations.revision, organisations.name, organisations.country FROM organisations
		JOIN organisations_users
		ON organisations_users.organisation_id = organisations.id
		WHERE organisations_users.user_id = $1
		`, qa[0])
		if err != nil {
			return err
		}
		defer rows.Close()
	}

	for rows.Next() {
		o := Organisation{}
		err = rows.Scan(&o.ID, &o.Revision, &o.Name, &o.Country)
		if err != nil {
			return err
		}

		(*organisations).Data = append((*organisations).Data, o)
	}

	return err
}
