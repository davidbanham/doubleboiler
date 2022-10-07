package models

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kylelemons/godebug/diff"
)

type Audit struct {
	ID             string
	EntityID       string
	OrganisationID string
	TableName      string
	Stamp          time.Time
	UserID         string
	UserName       string
	Action         string
	OldRowData     string
	NewRowData     string
	Diff           string
}

type Audits struct {
	Data []Audit
	baseModel
}

func (audits *Audits) FindAll(ctx context.Context, criteria Criteria) error {
	audits.Criteria = criteria

	db := ctx.Value("tx").(Querier)

	var rows *sql.Rows
	var err error

	switch v := criteria.Query.(type) {
	default:
		return fmt.Errorf("Unknown query")
	case ByEntityID:
		rows, err = db.QueryContext(ctx, `SELECT
		audit_log.id, entity_id, organisation_id, table_name, stamp, user_id, action, old_row_data - 'revision' - 'updated_at', users.email,
		lead(old_row_data - 'revision' - 'updated_at', 1) OVER (PARTITION BY entity_id ORDER BY stamp) new_row_data
		FROM audit_log LEFT JOIN users ON audit_log.user_id = users.id::text WHERE entity_id = $1 ORDER BY stamp DESC`+criteria.Pagination.PaginationQuery(), v.EntityID)
	case ByOrg:
		rows, err = db.QueryContext(ctx, `SELECT
		audit_log.id, entity_id, organisation_id, table_name, stamp, user_id, action, old_row_data - 'revision' - 'updated_at', users.email,
		lead(old_row_data - 'revision' - 'updated_at', 1) OVER (PARTITION BY entity_id ORDER BY stamp) new_row_data
		FROM audit_log LEFT JOIN users ON audit_log.user_id = users.id::text WHERE organisation_id = $1 ORDER BY stamp DESC`+criteria.Pagination.PaginationQuery(), v.ID)
	case All:
		rows, err = db.QueryContext(ctx, `SELECT
		audit_log.id, entity_id, organisation_id, table_name, stamp, user_id, action, old_row_data - 'revision' - 'updated_at', users.email,
		lead(old_row_data - 'revision' - 'updated_at', 1) OVER (PARTITION BY entity_id ORDER BY stamp) new_row_data
		FROM audit_log LEFT JOIN users ON audit_log.user_id = users.id::text ORDER BY stamp DESC`+criteria.Pagination.PaginationQuery())
	}
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		audit := Audit{}
		maybeNewRowData := sql.NullString{}
		maybeOldRowData := sql.NullString{}
		maybeUserName := sql.NullString{}
		if err := rows.Scan(
			&audit.ID,
			&audit.EntityID,
			&audit.OrganisationID,
			&audit.TableName,
			&audit.Stamp,
			&audit.UserID,
			&audit.Action,
			&maybeOldRowData,
			&maybeUserName,
			&maybeNewRowData,
		); err != nil {
			return err
		}
		audit.OldRowData = "{}"
		if maybeOldRowData.Valid {
			audit.OldRowData = maybeOldRowData.String
		}
		audit.NewRowData = "{}"
		if maybeNewRowData.Valid {
			audit.NewRowData = maybeNewRowData.String
		}
		audit.UserName = audit.UserID
		if maybeUserName.Valid {
			audit.UserName = maybeUserName.String
		}

		if !maybeNewRowData.Valid && audit.Action != "D" {
			if err := db.QueryRowContext(ctx, `SELECT to_jsonb(`+audit.TableName+`) - 'ts' - 'revision' - 'updated_at' FROM `+audit.TableName+` WHERE id = $1`, audit.EntityID).Scan(&audit.NewRowData); err != nil && err != sql.ErrNoRows {
				return err
			}
		}

		if audit.Action == "D" {
			audit.Diff = "Deleted"
		} else if maybeOldRowData.Valid {
			audit.OldRowData = prettyJsonString(audit.OldRowData)
			audit.NewRowData = prettyJsonString(audit.NewRowData)

			audit.Diff = diffOnly(diff.Diff(audit.OldRowData, audit.NewRowData))

		} else {
			audit.Diff = "Created"
		}

		(*audits).Data = append((*audits).Data, audit)
	}
	return err
}

func prettyJsonString(input string) string {
	var out bytes.Buffer
	json.Indent(&out, []byte(input), "", "  ")
	return out.String()
}

func diffOnly(input string) string {
	parts := strings.Split(input, "\n")
	relevant := []string{}
	for _, part := range parts {
		if strings.Index(part, "+") == 0 || strings.Index(part, "-") == 0 {
			if string(part[len(part)-1]) == "," {
				relevant = append(relevant, part[1:len(part)-1])
			} else {
				relevant = append(relevant, part[1:])
			}
		}
	}
	pairs := []string{}
	hold := ""
	for _, part := range relevant {
		if hold == "" {
			hold = part
		} else {
			sep := strings.Index(part, ":")
			pairs = append(pairs, fmt.Sprintf("%s -> %s", hold, part[sep+1:]))
			hold = ""
		}
	}
	if hold != "" {
		pairs = append(pairs, hold)
	}
	return strings.Join(pairs, " ")
}
