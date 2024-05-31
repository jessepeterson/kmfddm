package mysql

import (
	"context"
)

// RetrieveSetDeclarations retrieves the list of declarations a set is associated with.
// See also the storage package for documentation on the storage interfaces.
func (s *MySQLStorage) RetrieveSetDeclarations(ctx context.Context, setName string) ([]string, error) {
	return s.singleStringColumn(
		ctx,
		`SELECT declaration_identifier FROM set_declarations WHERE set_name = ?;`,
		setName,
	)
}

// StoreSetDeclaration creates the association between a declaration and a set.
// See also the storage package for documentation on the storage interfaces.
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
// See also the storage package for documentation on the storage interfaces.
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

// RetrieveSets retrieves the list of sets.
// See also the storage package for documentation on the storage interfaces.
func (s *MySQLStorage) RetrieveSets(ctx context.Context) ([]string, error) {
	return s.singleStringColumn(
		ctx,
		`SELECT DISTINCT set_name FROM set_declarations;`,
	)
}

// RemoveDeclarationsFromSet removes all declarations associated with a set.
// See also the storage package for documentation on the storage interfaces.
func (s *MySQLStorage) RemoveDeclarationsFromSet(ctx context.Context, setName string) (bool, error) {
	result, err := s.db.ExecContext(
		ctx, `
DELETE FROM set_declarations
WHERE
    set_name = ?;`,
		setName,
	)
	if err != nil {
		return false, err
	}
	return resultChangedRows(result)
}
