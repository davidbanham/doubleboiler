package models

import (
	"context"
	"fmt"

	scummodel "github.com/davidbanham/scum/model"
	scumquery "github.com/davidbanham/scum/query"
	scumsearch "github.com/davidbanham/scum/search"
	scumutil "github.com/davidbanham/scum/util"
)

type Criteria struct {
	Query      interface{}
	Filters    Filters
	Pagination Pagination
}

type Searchables = scumsearch.Searchables
type Searchable = scumsearch.Searchable
type SearchQuery = scumsearch.SearchQuery

var SearchTargets = Searchables{}

type Querier = scummodel.Querier

type Query = scumquery.Query

var StandardSave = scummodel.StandardSave
var StandardFindByColumn = scummodel.FindByColumn

type ClientSafeError = scumutil.ClientSafeError

var ErrWrongRev = scummodel.ErrWrongRev

func currentUser(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	unconv := ctx.Value("user")

	if unconv != nil {
		return unconv.(User).ID
	}
	return ""
}

func auditQuery(ctx context.Context, action, tableName, entityID, organisationID string) string {
	return fmt.Sprintf("WITH audit_entry AS (INSERT INTO audit_log (entity_id, organisation_id, table_name, action, user_id, old_row_data) VALUES ('%s', '%s', '%s', '%s', '%s', (SELECT to_jsonb(%s) - 'ts' FROM %s WHERE id = '%s')))", entityID, organisationID, tableName, action, currentUser(ctx), tableName, tableName, entityID)
}
