package models

import (
	"context"
	"database/sql"
	"fmt"

	uuid "github.com/satori/go.uuid"
)

type Thing struct {
	ID             string
	Name           string
	Description    string
	OrganisationID string
	Revision       string
}

func (thing *Thing) New(name, description, organisationID string) {
	thing.ID = uuid.NewV4().String()
	thing.Name = name
	thing.Description = description
	thing.OrganisationID = organisationID
	thing.Revision = uuid.NewV4().String()
}

func (thing *Thing) Save(ctx context.Context) error {
	db := ctx.Value("tx").(Querier)

	row := db.QueryRowContext(ctx, `INSERT INTO things (
		id, revision, name, description, organisation_id
	) VALUES (
		$1, $2, $4, $5, $6
	) ON CONFLICT (revision) DO UPDATE SET (
		revision, name, description, organisation_id
	) = (
		$3, $4, $5, $6
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
	id, revision, name, description, organisation_id
	FROM things WHERE `+col+` = $1`, val).Scan(
		&thing.ID, &thing.Revision, &thing.Name, &thing.Description, &thing.OrganisationID,
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
		id, revision, name, description, organisation_id
		FROM things WHERE organisation_id = $1`, v.ID)
	case All:
		rows, err = db.QueryContext(ctx, `SELECT
		id, revision, name, description, organisation_id
		FROM things`)
	}
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		thing := Thing{}
		err = rows.Scan(
			&thing.ID, &thing.Revision, &thing.Name, &thing.Description, &thing.OrganisationID,
		)
		if err != nil {
			return err
		}
		(*things).Data = append((*things).Data, thing)
	}
	return err
}
