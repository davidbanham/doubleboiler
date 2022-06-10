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
		Label:            "SomeThings",
		RequiredRole:     requiredRole,
		searchFunc:       searchSomeThings(requiredRole),
		availableFilters: someThingFilters,
	})
}

type SomeThing struct {
	ID             string
	Name           string
	Description    string
	OrganisationID string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Revision       string
}

func (someThing *SomeThing) New(name, description, organisationID string) {
	someThing.ID = uuid.NewV4().String()
	someThing.Name = name
	someThing.Description = description
	someThing.OrganisationID = organisationID
	someThing.CreatedAt = time.Now()
	someThing.UpdatedAt = time.Now()
}

func (someThing *SomeThing) auditQuery(ctx context.Context, action string) string {
	return auditQuery(ctx, action, "some_things", someThing.ID, someThing.OrganisationID)
}

func (someThing *SomeThing) Save(ctx context.Context) error {
	db := ctx.Value("tx").(Querier)

	newRev := uuid.NewV4().String()

	result, err := db.ExecContext(ctx, someThing.auditQuery(ctx, "U")+`INSERT INTO some_things (
		updated_at,
		id,
		revision,
		name,
		description,
		organisation_id
	) VALUES (
		now(), $1, $3, $4, $5, $6
	) ON CONFLICT (id) DO UPDATE SET (
		updated_at,
		revision,
		name,
		description,
		organisation_id
	) = (
		now(), $3, $4, $5, $6
	) WHERE some_things.revision = $2`,
		someThing.ID,
		someThing.Revision,
		newRev,
		someThing.Name,
		someThing.Description,
		someThing.OrganisationID,
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

	someThing.Revision = newRev

	return nil
}

func (someThing *SomeThing) FindByID(ctx context.Context, id string) error {
	return someThing.FindByColumn(ctx, "id", id)
}

func (someThing *SomeThing) FindByColumn(ctx context.Context, col, val string) error {
	db := ctx.Value("tx").(Querier)

	return db.QueryRowContext(ctx, `SELECT
	id,
	revision,
	created_at,
	updated_at,
	name,
	description,
	organisation_id
	FROM some_things
	WHERE `+col+` = $1
	`, val).Scan(
		&someThing.ID,
		&someThing.Revision,
		&someThing.CreatedAt,
		&someThing.UpdatedAt,
		&someThing.Name,
		&someThing.Description,
		&someThing.OrganisationID,
	)
}

type SomeThings struct {
	Data  []SomeThing
	Query Query
}

func (this SomeThings) ByID() map[string]SomeThing {
	ret := map[string]SomeThing{}
	for _, t := range this.Data {
		ret[t.ID] = t
	}
	return ret
}

func (someThings *SomeThings) FindAll(ctx context.Context, q Query) error {
	someThings.Query = q

	db := ctx.Value("tx").(Querier)

	var rows *sql.Rows
	var err error

	switch v := q.(type) {
	default:
		return fmt.Errorf("Unknown query")
	case ByOrg:
		rows, err = db.QueryContext(ctx, `SELECT
			id,
			revision,
			created_at,
			updated_at,
			name,
			description,
			organisation_id
		FROM some_things
		`+filterQuery(v)+`
		AND organisation_id = $1
		ORDER BY name`+v.Pagination(), v.ID)
	case All:
		rows, err = db.QueryContext(ctx, `SELECT
			id,
			revision,
			created_at,
			updated_at,
			name,
			description,
			organisation_id
		FROM some_things
		`+filterQuery(v)+`
		ORDER BY name`+v.Pagination())
	}
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		someThing := SomeThing{}
		err = rows.Scan(
			&someThing.ID,
			&someThing.Revision,
			&someThing.CreatedAt,
			&someThing.UpdatedAt,
			&someThing.Name,
			&someThing.Description,
			&someThing.OrganisationID,
		)
		if err != nil {
			return err
		}
		(*someThings).Data = append((*someThings).Data, someThing)
	}
	return err
}

func (someThings SomeThings) AvailableFilters() Filters {
	return standardFilters()
}

func someThingFilters() Filters {
	return Filters{}
}

func searchSomeThings(requiredRole Role) func(ByPhrase) string {
	return func(query ByPhrase) string {
		if query.User.Admin || query.Roles.Can(requiredRole.Name) {
			return `SELECT
		text 'SomeThing' AS entity_type, text 'some_things' AS uri_path, id AS id, name || ' - ' || description AS label, ts_rank_cd(ts, query) AS rank
FROM
		some_things, plainto_tsquery('english', $2) query ` + filterQuery(query) + ` AND organisation_id = $1 AND query @@ ts`
		}
		return ""
	}
}
