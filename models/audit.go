package models

import (
	"context"
	"database/sql"
	"doubleboiler/util"
	"time"

	"github.com/davidbanham/scum/query"
)

type Audit struct {
	ID              string
	EntityID        string
	OrganisationID  string
	TableName       string
	Stamp           time.Time
	UserID          string
	UserName        string
	Action          string
	OldRowData      string
	maybeOldRowData sql.NullString
	NewRowData      string
	maybeNewRowData sql.NullString
	Diff            string
}

type Audits struct {
	Data     []Audit
	Criteria Criteria
}

func (this *Audits) FindAll(ctx context.Context, criteria Criteria) error {
	this.Criteria = criteria

	db := ctx.Value("tx").(Querier)

	var rows *sql.Rows
	var err error

	cols := append([]string{
		"audit_log.id",
		"entity_id",
		"organisation_id",
		"table_name",
		"stamp",
		"user_id",
		"action",
		"old_row_data - 'revision' - 'updated_at'",
		"users.email",
		"lead(old_row_data - 'revision' - 'updated_at', 1) OVER (PARTITION BY entity_id ORDER BY stamp) new_row_data",
	})

	switch v := criteria.Query.(type) {
	default:
		return ErrInvalidQuery{Query: v, Model: "audit_log"}
	case custom:
		switch v := criteria.customQuery.(type) {
		default:
			return ErrInvalidQuery{Query: v, Model: "audit_log"}
		case ByEntityID:
			rows, err = db.QueryContext(ctx, `SELECT
		audit_log.id, entity_id, organisation_id, table_name, stamp, user_id, action, old_row_data - 'revision' - 'updated_at' - 'password' - 'totp_secret' - 'recovery_codes', users.email,
		lead(old_row_data - 'revision' - 'updated_at' - 'password' - 'totp_secret' - 'recovery_codes', 1) OVER (PARTITION BY entity_id ORDER BY stamp) new_row_data
		FROM audit_log LEFT JOIN users ON audit_log.user_id = users.id::text WHERE entity_id = $1 ORDER BY stamp DESC`+criteria.Pagination.PaginationQuery(), v.EntityID)
		}
	case query.Query:
		rows, err = db.QueryContext(ctx, v.Construct(cols, "audit_log LEFT JOIN users ON audit_log.user_id = users.id::text", criteria.Filters, criteria.Pagination, "stamp"), v.Args()...)
	}
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		audit := Audit{}
		maybeUserName := sql.NullString{}
		if err := rows.Scan(
			&audit.ID,
			&audit.EntityID,
			&audit.OrganisationID,
			&audit.TableName,
			&audit.Stamp,
			&audit.UserID,
			&audit.Action,
			&audit.maybeOldRowData,
			&maybeUserName,
			&audit.maybeNewRowData,
		); err != nil {
			return err
		}
		audit.OldRowData = "{}"
		if audit.maybeOldRowData.Valid {
			audit.OldRowData = audit.maybeOldRowData.String
		}
		audit.NewRowData = "{}"
		if audit.maybeNewRowData.Valid {
			audit.NewRowData = audit.maybeNewRowData.String
		}
		audit.UserName = audit.UserID
		if maybeUserName.Valid {
			audit.UserName = maybeUserName.String
		}

		(*this).Data = append((*this).Data, audit)
	}

	for i, audit := range (*this).Data {
		if !audit.maybeNewRowData.Valid && audit.Action != "D" {
			if err := db.QueryRowContext(ctx, `SELECT to_jsonb(`+audit.TableName+`) - 'ts' - 'revision' - 'updated_at' FROM `+audit.TableName+` WHERE id = $1`, audit.EntityID).Scan(&audit.NewRowData); err != nil && err != sql.ErrNoRows {
				return err
			}
		}

		if audit.Action == "D" {
			audit.Diff = "Deleted"
		} else if audit.maybeOldRowData.Valid {
			audit.OldRowData = util.PrettyJsonString(audit.OldRowData)
			audit.NewRowData = util.PrettyJsonString(audit.NewRowData)

			audit.Diff = util.DiffOnly(audit.OldRowData, audit.NewRowData)
		} else {
			audit.Diff = "Created"
		}

		(*this).Data[i] = audit
	}
	return err
}
