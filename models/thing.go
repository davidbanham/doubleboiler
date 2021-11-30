package models

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	uuid "github.com/satori/go.uuid"
)

func init() {
	searchFuncs = append(searchFuncs, searchThings)
}

type Thing struct {
	ID             string
	Name           string
	Description    string
	OrganisationID string
	Revision       string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (thing *Thing) New(name, description, organisationID string) {
	thing.ID = uuid.NewV4().String()
	thing.Name = name
	thing.Description = description
	thing.OrganisationID = organisationID
	thing.Revision = uuid.NewV4().String()
	thing.CreatedAt = time.Now()
	thing.UpdatedAt = time.Now()
}

func (thing *Thing) auditQuery(ctx context.Context, action string) string {
	return auditQuery(ctx, action, "things", thing.ID)
}

func (thing *Thing) Save(ctx context.Context) error {
	db := ctx.Value("tx").(Querier)

	row := db.QueryRowContext(ctx, thing.auditQuery(ctx, "U")+`INSERT INTO things (
		id, revision, name, description, organisation_id
	) VALUES (
		$1, $2, $4, $5, $6
	) ON CONFLICT (revision) DO UPDATE SET (
		updated_at, revision, name, description, organisation_id
	) = (
		now(), $3, $4, $5, $6
	) RETURNING revision`,
		thing.ID, thing.Revision, uuid.NewV4().String(), thing.Name, thing.Description, thing.OrganisationID,
	)
	return row.Scan(&thing.Revision)
}

func (thing *Thing) FindByID(ctx context.Context, id string) error {
	return thing.FindByColumn(ctx, "id", id)
}

func (thing *Thing) FindByColumn(ctx context.Context, col, val string) error {
	db := ctx.Value("tx").(Querier)

	return db.QueryRowContext(ctx, `SELECT
	id, revision, created_at, updated_at, name, description, organisation_id
	FROM things WHERE `+col+` = $1`, val).Scan(
		&thing.ID, &thing.Revision, &thing.CreatedAt, &thing.UpdatedAt, &thing.Name, &thing.Description, &thing.OrganisationID,
	)
}

type Things struct {
	Data  []Thing
	Query Query
}

func (things *Things) FindAll(ctx context.Context, q Query) error {
	things.Query = q

	db := ctx.Value("tx").(Querier)

	var rows *sql.Rows
	var err error

	switch v := q.(type) {
	default:
		return fmt.Errorf("Unknown query")
	case ByOrg:
		rows, err = db.QueryContext(ctx, `SELECT
		id, revision, created_at, updated_at, name, description, organisation_id
		FROM things WHERE organisation_id = $1 `+v.Pagination(), v.ID)
	case All:
		rows, err = db.QueryContext(ctx, `SELECT
		id, revision, created_at, updated_at, name, description, organisation_id
		FROM things `+v.Pagination())
	}
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		thing := Thing{}
		err = rows.Scan(
			&thing.ID, &thing.Revision, &thing.CreatedAt, &thing.UpdatedAt, &thing.Name, &thing.Description, &thing.OrganisationID,
		)
		if err != nil {
			return err
		}
		(*things).Data = append((*things).Data, thing)
	}
	return err
}

func searchThings(user User) string {
	return `SELECT
		text 'Thing' AS entity_type, text 'things' AS uri_path, id AS id, name || ' - ' || description AS label, ts_rank_cd(ts, query) AS rank
FROM
		things, plainto_tsquery('english', $2) query WHERE organisation_id = $1 AND query @@ ts`
}
