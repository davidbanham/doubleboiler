package models

import (
	"context"
	"database/sql"
	"fmt"

	uuid "github.com/satori/go.uuid"
)

func init() {
	searchFuncs = append(searchFuncs, searchOrganisations)
}

type Organisation struct {
	ID       string
	Name     string
	Country  string
	Users    []OrganisationUser
	Revision string
}

func (org *Organisation) New(name, country string, users []OrganisationUser, currency string) {
	org.ID = uuid.NewV4().String()
	org.Users = users
	org.Name = name
	org.Country = country
	org.Revision = uuid.NewV4().String()
}

func (org *Organisation) Save(ctx context.Context) error {
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
		org.ID,
		org.Revision,
		uuid.NewV4().String(),
		org.Name,
		org.Country,
	)

	err := row.Scan(&org.Revision)
	return err
}

func (org *Organisation) FindByColumn(ctx context.Context, col, val string) error {
	db := ctx.Value("tx").(Querier)

	err := db.QueryRowContext(ctx, `SELECT
	id,
	revision,
	name,
	country
	FROM organisations WHERE `+col+` = $1`, val).Scan(
		&org.ID,
		&org.Revision,
		&org.Name,
		&org.Country,
	)

	if err != nil {
		return err
	}

	org.Users = []OrganisationUser{}

	rows, err := db.QueryContext(ctx, `SELECT
	organisations_users.id,
	organisations_users.revision,
	organisations_users.user_id,
	organisations_users.organisation_id,
	users.email
	FROM organisations_users
	INNER JOIN users
	ON organisations_users.user_id = users.id
	WHERE organisations_users.organisation_id = $1`, org.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		orguser := OrganisationUser{}
		err = rows.Scan(&orguser.ID, &orguser.Revision, &orguser.UserID, &orguser.OrganisationID, &orguser.Email)
		if err != nil {
			return err
		}
		org.Users = append(org.Users, orguser)
	}

	return err
}

func (org *Organisation) FindByID(ctx context.Context, id string) error {
	return org.FindByColumn(ctx, "id", id)
}

type Organisations struct {
	Data  []Organisation
	Query Query
}

func (organisations *Organisations) FindAll(ctx context.Context, q Query) error {
	organisations.Query = q

	db := ctx.Value("tx").(Querier)

	var rows *sql.Rows
	var err error

	switch v := q.(type) {
	default:
		return fmt.Errorf("Unknown query")
	case All:
		rows, err = db.QueryContext(ctx, "SELECT id, revision, name, country FROM organisations "+v.Pagination())
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
		`+v.Pagination(), v.ID)
		if err != nil {
			return err
		}
		defer rows.Close()
	}

	for rows.Next() {
		org := Organisation{}
		err = rows.Scan(&org.ID, &org.Revision, &org.Name, &org.Country)
		if err != nil {
			return err
		}

		(*organisations).Data = append((*organisations).Data, org)
	}

	return err
}

func searchOrganisations(user User) string {
	return `SELECT
		text 'Organisation' AS entity_type, text 'organisations' AS uri_path, id AS id, name AS label, ts_rank_cd(ts, query) AS rank
FROM
		organisations, plainto_tsquery('english', $2) query WHERE id = $1 AND query @@ ts`
}
