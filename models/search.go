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
		query := `SELECT
					text 'Thing' AS entity_type, text 'things' AS uri_path, id AS id, name || ' - ' || description AS label, ts_rank_cd(ts, query) AS rank
			FROM
					things, plainto_tsquery('english', $2) query WHERE organisation_id = $1 AND query @@ ts
			UNION ALL
			SELECT
					text 'Organisation' AS entity_type, text 'organisations' AS uri_path, id AS id, name AS label, ts_rank_cd(ts, query) AS rank
			FROM
					organisations, plainto_tsquery('english', $2) query WHERE id = $1 AND query @@ ts`

		if v.IncludeUsers {
			query += `
			UNION ALL
			SELECT
					text 'User' AS entity_type, text 'users' AS uri_path, id AS id, email AS label, 1 AS rank
			FROM
					users WHERE email = $2`
		}

		query += `
			ORDER BY rank DESC`

		rows, err = db.QueryContext(ctx, query, v.OrgID, v.Phrase)
	}
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		result := SearchResult{}
		err = rows.Scan(
			&result.EntityType, &result.Path, &result.ID, &result.Label, &result.Rank,
		)
		if err != nil {
			return err
		}
		(*results).Data = append((*results).Data, result)
	}

	return err
}
