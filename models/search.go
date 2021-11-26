package models

import (
	"context"
	"database/sql"
	"fmt"
)

type SearchResult struct {
	Path       string
	EntityType string
	ID         string
	Label      string
	Rank       float64
}

type SearchResults struct {
	Data  []SearchResult
	Query Query
}

func (results *SearchResults) FindAll(ctx context.Context, q Query) error {
	results.Query = q

	db := ctx.Value("tx").(Querier)

	var rows *sql.Rows
	var err error

	switch v := q.(type) {
	default:
		return fmt.Errorf("Unknown query")
	case ByPhrase:
		rows, err = db.QueryContext(ctx, `SELECT
		id, entity_type, label, uri_path, ts_rank_cd(ts, query) AS rank
		FROM search_items, plainto_tsquery('english', $2) query WHERE organisation_id = $1 AND query @@ ts ORDER BY rank DESC`, v.OrgID, v.Phrase)
	}
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		result := SearchResult{}
		err = rows.Scan(
			&result.ID, &result.EntityType, &result.Label, &result.Path, &result.Rank,
		)
		if err != nil {
			return err
		}
		(*results).Data = append((*results).Data, result)
	}

	return err
}
