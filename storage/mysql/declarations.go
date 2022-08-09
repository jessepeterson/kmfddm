package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/jessepeterson/kmfddm/ddm"
)

func (s *MySQLStorage) StoreDeclaration(ctx context.Context, d *ddm.Declaration) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	result, err := tx.ExecContext(
		ctx,
		`
INSERT INTO declarations
    (identifier, type, payload)
VALUES
    (?, ?, ?) AS new
ON DUPLICATE KEY
UPDATE
    type    = new.type,
    payload = new.payload;`,
		d.Identifier,
		d.Type,
		d.PayloadJSON,
	)
	var changed bool
	if err == nil {
		changed, err = resultChangedRows(result)
	}
	if err == nil {
		// i don't like this delete+re-insert pattern
		_, err = tx.ExecContext(
			ctx,
			`DELETE FROM declaration_references WHERE declaration_identifier = ?;`,
			d.Identifier,
		)
		if err == nil && len(d.IdentifierRefs) > 0 {
			vals := make([]interface{}, len(d.IdentifierRefs)*2)
			for i, id := range d.IdentifierRefs {
				vals[i*2] = d.Identifier
				vals[i*2+1] = id
			}
			sqlVals := strings.Repeat(", (?, ?)", len(d.IdentifierRefs))[1:]
			_, err = tx.ExecContext(
				ctx,
				`INSERT INTO declaration_references (declaration_identifier, declaration_reference) VALUES`+sqlVals,
				vals...,
			)
		}
	}
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return false, fmt.Errorf("rollback error: %w; while trying to handle error: %v", rbErr, err)
		}
		return false, err
	}
	return changed, tx.Commit()
}

func (s *MySQLStorage) RetrieveDeclaration(ctx context.Context, declarationID string) (d *ddm.Declaration, err error) {
	// we could simply load the declaration JSON and do a ddm.ParseDeclaration()
	// but we'll try to take advantage of our fancy RDBMS we have here :)
	d = new(ddm.Declaration)
	err = s.db.QueryRowContext(
		ctx, `
SELECT
    d.identifier,
    d.type,
    d.payload,
    d.server_token,
    JSON_OBJECT(
        "Identifier",  d.identifier,
        "Type",        d.type,
        "Payload",     d.payload,
        "ServerToken", d.server_token
    ) AS declaration
FROM
    declarations d
WHERE
    d.identifier = ?;`,
		declarationID,
	).Scan(
		&d.Identifier,
		&d.Type,
		&d.PayloadJSON,
		&d.ServerToken,
		&d.Raw,
	)
	if err != nil {
		return
	}
	rows, err := s.db.QueryContext(
		ctx, `
SELECT
    declaration_reference
FROM
    declaration_references
WHERE
    declaration_identifier = ?;`,
		declarationID,
	)
	if err != nil {
		return
	}
	defer rows.Close()
	var declarationRef string
	for rows.Next() {
		err = rows.Scan(&declarationRef)
		if err != nil {
			break
		}
		d.IdentifierRefs = append(d.IdentifierRefs, declarationRef)
	}
	if err == nil {
		err = rows.Err()
	}
	return
}

func (s *MySQLStorage) DeleteDeclaration(ctx context.Context, identifier string) (bool, error) {
	result, err := s.db.ExecContext(
		ctx, `DELETE FROM declarations WHERE identifier = ?;`,
		identifier,
	)
	if err != nil {
		return false, err
	}
	return resultChangedRows(result)
}

func (s *MySQLStorage) RetrieveDeclarationSets(ctx context.Context, declarationID string) ([]string, error) {
	return s.singleStringColumn(
		ctx,
		`SELECT set_name FROM set_declarations WHERE declaration_identifier = ?;`,
		declarationID,
	)
}

func (s *MySQLStorage) RetrieveDeclarations(ctx context.Context) ([]string, error) {
	return s.singleStringColumn(
		ctx,
		`SELECT identifier FROM declarations;`,
	)
}
