// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0
// source: query.sql

package sqlc

import (
	"context"
	"database/sql"
)

const getManifestItems = `-- name: GetManifestItems :many
SELECT DISTINCT
    d.identifier,
    d.type,
    d.server_token
FROM
    declarations d
    INNER JOIN set_declarations sd
        ON d.identifier = sd.declaration_identifier
    INNER JOIN enrollment_sets es
        ON sd.set_name = es.set_name
WHERE
    es.enrollment_id = ?
`

type GetManifestItemsRow struct {
	Identifier  string
	Type        string
	ServerToken string
}

func (q *Queries) GetManifestItems(ctx context.Context, enrollmentID string) ([]GetManifestItemsRow, error) {
	rows, err := q.db.QueryContext(ctx, getManifestItems, enrollmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetManifestItemsRow
	for rows.Next() {
		var i GetManifestItemsRow
		if err := rows.Scan(&i.Identifier, &i.Type, &i.ServerToken); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const removeAllEnrollmentSets = `-- name: RemoveAllEnrollmentSets :execresult
DELETE FROM
    enrollment_sets
WHERE
    enrollment_id = ?
`

func (q *Queries) RemoveAllEnrollmentSets(ctx context.Context, enrollmentID string) (sql.Result, error) {
	return q.db.ExecContext(ctx, removeAllEnrollmentSets, enrollmentID)
}
