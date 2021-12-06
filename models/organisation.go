package models

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	uuid "github.com/satori/go.uuid"
)

func init() {
	Searchables = append(Searchables, Searchable{
		Label:      "Organisations",
		searchFunc: searchOrganisations,
	})
}

type Organisation struct {
	ID        string
	Name      string
	Country   string
	Users     OrganisationUsers
	Revision  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (org *Organisation) New(name, country string, users OrganisationUsers, currency string) {
	org.ID = uuid.NewV4().String()
	org.Users = users
	org.Name = name
	org.Country = country
	org.Revision = uuid.NewV4().String()
	org.CreatedAt = time.Now()
	org.UpdatedAt = time.Now()
}

func (org *Organisation) auditQuery(ctx context.Context, action string) string {
	return auditQuery(ctx, action, "organisations", org.ID, org.ID)
}

func (org *Organisation) Save(ctx context.Context) error {
	db := ctx.Value("tx").(Querier)

	row := db.QueryRowContext(ctx, org.auditQuery(ctx, "U")+`INSERT INTO organisations (
		id,
		revision,
		name,
		country
	) VALUES ($1, $2, $4, $5) ON CONFLICT (revision) DO UPDATE SET (
		revision,
		updated_at,
		name,
		country
	) = ($3, now(), $4, $5) RETURNING revision`,
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

	org.Users = OrganisationUsers{}
	if err := org.Users.FindAll(ctx, ByOrg{ID: org.ID}); err != nil {
		return err
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
		rows, err = db.QueryContext(ctx, "SELECT id, revision, created_at, updated_at, name, country FROM organisations "+v.Pagination())
		if err != nil {
			return err
		}
		defer rows.Close()
	case OrganisationsContainingUser:
		rows, err = db.QueryContext(ctx, `
		SELECT organisations.id,
			organisations.revision,
			organisations.created_at,
			organisations.updated_at,
			organisations.name,
			organisations.country
		FROM organisations
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
		if err = rows.Scan(&org.ID,
			&org.Revision,
			&org.CreatedAt,
			&org.UpdatedAt,
			&org.Name,
			&org.Country,
		); err != nil {
			return err
		}

		(*organisations).Data = append((*organisations).Data, org)
	}

	for i, org := range (*organisations).Data {
		org.Users = OrganisationUsers{}
		if err := org.Users.FindAll(ctx, ByOrg{ID: org.ID}); err != nil {
			return err
		}
		(*organisations).Data[i] = org
	}

	return err
}

func searchOrganisations(query ByPhrase) string {
	return `SELECT
		text 'Organisation' AS entity_type, text 'organisations' AS uri_path, id AS id, name AS label, ts_rank_cd(ts, query) AS rank
FROM
		organisations, plainto_tsquery('english', $2) query WHERE id = $1 AND query @@ ts`
}
