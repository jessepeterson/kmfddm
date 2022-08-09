package mysql

import (
	"context"
	"encoding/json"

	"github.com/jessepeterson/kmfddm/ddm"
)

// RetrieveEnrollmentDeclarationJSON retreives a declaration intended for a
// given enrollment. As it is intended to retrieve the declaration for
// delivery to the enrollment it queries to make sure that enrollment should
// have access and that it is of the correct type.
func (s *MySQLStorage) RetrieveEnrollmentDeclarationJSON(ctx context.Context, declarationID, declarationType, enrollmentID string) (raw []byte, err error) {
	// we JOIN against the enrollments table to make sure only those
	// declarations that are transitively related are able to be
	// accessed. kinda-sorta like an ACL. almost.
	err = s.db.QueryRowContext(
		ctx, `
SELECT
    JSON_OBJECT(
        "Identifier",  d.identifier,
        "Type",        d.type,
        "Payload",     d.payload,
        "ServerToken", d.server_token
    ) AS declaration
FROM
    declarations d
    INNER JOIN set_declarations sd
        ON d.identifier = sd.declaration_identifier
    INNER JOIN enrollment_sets es
        ON sd.set_name = es.set_name
WHERE
    d.identifier = ? AND
    es.enrollment_id = ? AND
    d.type LIKE ?;`,
		declarationID,
		enrollmentID,
		"com.apple."+declarationType+".%",
	).Scan(&raw)
	return
}

type Builder interface {
	AddDeclarationData(*ddm.Declaration)
	Finalize()
}

func (s *MySQLStorage) build(ctx context.Context, b Builder, enrollmentID string) error {
	rows, err := s.db.QueryContext(
		ctx, `
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
    es.enrollment_id = ?;`,
		enrollmentID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		// note that we're selecting and assembling a very minimal Declaration
		// here. just enough to work with the Builder interface. check the
		// Builder implementation to make sure it doesn't need anything more
		// than what we're giving it.
		d := new(ddm.Declaration)
		err = rows.Scan(
			&d.Identifier,
			&d.Type,
			&d.ServerToken,
		)
		if err != nil {
			break
		}
		b.AddDeclarationData(d)
	}
	if err != nil {
		return err
	}
	if err = rows.Err(); err != nil {
		return err
	}
	b.Finalize()
	return nil
}

func (s *MySQLStorage) RetrieveDeclarationItemsJSON(ctx context.Context, enrollmentID string) ([]byte, error) {
	di := ddm.NewDIBuilder(s.newHash)
	err := s.build(ctx, di, enrollmentID)
	if err != nil {
		return nil, err
	}
	return json.Marshal(&di.DeclarationItems)
}

func (s *MySQLStorage) RetrieveTokensJSON(ctx context.Context, enrollmentID string) ([]byte, error) {
	b := ddm.NewTokensBuilder(s.newHash)
	err := s.build(ctx, b, enrollmentID)
	if err != nil {
		return nil, err
	}
	return json.Marshal(&b.TokensResponse)
}
