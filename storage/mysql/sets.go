package mysql

import (
	"context"
)

func (s *MySQLStorage) RetrieveSetDeclarations(ctx context.Context, setName string) ([]string, error) {
	return s.singleStringColumn(
		ctx,
		`SELECT declaration_identifier FROM set_declarations WHERE set_name = ?;`,
		setName,
	)
}

// StoreSetDeclaration creates the association between a declaration and a set.
func (s *MySQLStorage) StoreSetDeclaration(ctx context.Context, setName, declarationID string) (bool, error) {
	result, err := s.db.ExecContext(
		ctx, `
INSERT INTO set_declarations
    (declaration_identifier, set_name)
VALUES
    (?, ?)
ON DUPLICATE KEY
UPDATE
    set_name = set_name;`,
		declarationID,
		setName,
	)
	if err != nil {
		return false, err
	}
	return resultChangedRows(result)
}

// RemoveSetDeclaration removes the association between a declaration and a set.
func (s *MySQLStorage) RemoveSetDeclaration(ctx context.Context, setName, declarationID string) (bool, error) {
	result, err := s.db.ExecContext(
		ctx, `
DELETE FROM set_declarations
WHERE
    set_name = ? AND
    declaration_identifier = ?;`,
		setName,
		declarationID,
	)
	if err != nil {
		return false, err
	}
	return resultChangedRows(result)
}

func (s *MySQLStorage) RetrieveSets(ctx context.Context) ([]string, error) {
	return s.singleStringColumn(
		ctx,
		`SELECT DISTINCT set_name FROM set_declarations;`,
	)
}
