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
	OrganisationID string
	Revision       string
}

func (thing *Thing) New(name, organisationID string) {
	thing.ID = uuid.NewV4().String()
	thing.Name = name
	thing.OrganisationID = organisationID
	thing.Revision = uuid.NewV4().String()
}

func (thing *Thing) Save(ctx context.Context) error {
	db := ctx.Value("tx").(Querier)

	row := db.QueryRowContext(ctx, `INSERT INTO things (
		id, revision, name, organisation_id
	) VALUES (
		$1, $2, $4, $5
	) ON CONFLICT (revision) DO UPDATE SET (
		revision, name, organisation_id
	) = (
		$3, $4, $5
	) RETURNING revision`,
		thing.ID, thing.Revision, uuid.NewV4().String(), thing.Name, thing.OrganisationID,
	)
	return row.Scan(&thing.Revision)
}

func (thing *Thing) FindByID(ctx context.Context, id string) error {
	return thing.FindByColumn(ctx, "id", id)
}

func (thing *Thing) FindByColumn(ctx context.Context, col, val string) error {
	db := ctx.Value("tx").(Querier)

	return db.QueryRowContext(ctx, `SELECT
	id, revision, name, organisation_id
	FROM things WHERE `+col+` = $1`, val).Scan(
		&thing.ID, &thing.Revision, &thing.Name, &thing.OrganisationID,
	)
}

type Things []Thing

func (things *Things) FindAll(ctx context.Context, q Query, qa ...string) error {
	db := ctx.Value("tx").(Querier)

	var rows *sql.Rows
	var err error

	switch q {
	default:
		return fmt.Errorf("Unknown query")
	case ByOrg:
		rows, err = db.QueryContext(ctx, `SELECT
		id, revision, name, organisation_id
		FROM things WHERE organisation_id = $1`, qa[0])
	case All:
		rows, err = db.QueryContext(ctx, `SELECT
		id, revision, name, organisation_id
		FROM things`)
	}
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		thing := Thing{}
		err = rows.Scan(
			&thing.ID, &thing.Revision, &thing.Name, &thing.OrganisationID,
		)
		if err != nil {
			return err
		}
		(*things) = append((*things), thing)
	}
	return err
}
