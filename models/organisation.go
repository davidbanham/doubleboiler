package models

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	uuid "github.com/satori/go.uuid"
)

func init() {
	requiredRole := ValidRoles["admin"]
	Searchables = append(Searchables, Searchable{
		Label:        "Organisations",
		RequiredRole: requiredRole,
		searchFunc:   searchOrganisations(requiredRole),
	})
}

type Organisation struct {
	ID        string
	Name      string
	Country   string
	Revision  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (org *Organisation) New(name, country string) {
	org.ID = uuid.NewV4().String()
	org.Name = name
	org.Country = country
	org.CreatedAt = time.Now()
	org.UpdatedAt = time.Now()
}

func (org *Organisation) auditQuery(ctx context.Context, action string) string {
	return auditQuery(ctx, action, "organisations", org.ID, org.ID)
}

func (organisations Organisations) AvailableFilters() Filters {
	return organisationFilters()
}

func organisationFilters() Filters {
	return standardFilters()
}

func (org *Organisation) Save(ctx context.Context) error {
	db := ctx.Value("tx").(Querier)

	newRev := uuid.NewV4().String()

	result, err := db.ExecContext(ctx, org.auditQuery(ctx, "U")+`INSERT INTO organisations (
		updated_at,
		id,
		revision,
		name,
		country
	) VALUES (
		now(), $1, $3, $4, $5
	) ON CONFLICT (id) DO UPDATE SET (
		updated_at,
		revision,
		name,
		country
	) = (
		now(), $3, $4, $5
	) WHERE organisations.revision = $2`,
		org.ID,
		org.Revision,
		newRev,
		org.Name,
		org.Country,
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

	org.Revision = newRev

	return nil
}

func (org *Organisation) FindByColumn(ctx context.Context, col, val string) error {
	db := ctx.Value("tx").(Querier)

	return db.QueryRowContext(ctx, `SELECT
	id,
	revision,
	created_at,
	updated_at,
	name,
	country
	FROM organisations WHERE `+col+` = $1`, val).Scan(
		&org.ID,
		&org.Revision,
		&org.CreatedAt,
		&org.UpdatedAt,
		&org.Name,
		&org.Country,
	)
}

func (org *Organisation) FindByID(ctx context.Context, id string) error {
	return org.FindByColumn(ctx, "id", id)
}

type Organisations struct {
	Data  []Organisation
	Query Query
}

func (this Organisations) ByID() map[string]Organisation {
	ret := map[string]Organisation{}
	for _, t := range this.Data {
		ret[t.ID] = t
	}
	return ret
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
		rows, err = db.QueryContext(ctx, `SELECT
		id,
		revision,
		created_at,
		updated_at,
		name,
		country
		FROM organisations
		`+filterQuery(v)+`
		ORDER BY name`+v.Pagination())
		if err != nil {
			return err
		}
		defer rows.Close()
	case OrganisationsContainingUser:
		rows, err = db.QueryContext(ctx, `
		SELECT
		organisations.id,
			organisations.revision,
			organisations.created_at,
			organisations.updated_at,
			organisations.name,
			organisations.country
		FROM organisations
		JOIN organisations_users
		ON organisations_users.organisation_id = organisations.id
		`+filterQuery(v)+`
		AND organisations_users.user_id = $1
		ORDER BY name`+v.Pagination(), v.ID)
		if err != nil {
			return err
		}
		defer rows.Close()
	}

	for rows.Next() {
		org := Organisation{}
		if err = rows.Scan(
			&org.ID,
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

	return err
}
func searchOrganisations(requiredRole Role) func(ByPhrase) string {
	return func(query ByPhrase) string {
		if query.User.Admin || query.Roles.Can(requiredRole.Name) {
			return `SELECT
		text 'Organisation' AS entity_type, text 'organisations' AS uri_path, id AS id, name AS label, ts_rank_cd(ts, query) AS rank
FROM
		organisations, plainto_tsquery('english', $2) query WHERE id = $1 AND query @@ ts`
		}
		return ""
	}
}
