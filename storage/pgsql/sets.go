package pgsql

import (
	"context"
	"time"

	"github.com/jessepeterson/kmfddm/storage/pgsql/sqlc"
)

// RetrieveSetDeclarations retrieves the list of declarations a set is associated with.
// See also the storage package for documentation on the storage interfaces.
func (s *PgSQLStorage) RetrieveSetDeclarations(ctx context.Context, setName string) ([]string, error) {
	return s.q.GetSetDeclarations(ctx, setName)
}

// StoreSetDeclaration creates the association between a declaration and a set.
// See also the storage package for documentation on the storage interfaces.
func (s *PgSQLStorage) StoreSetDeclaration(ctx context.Context, setName, declarationID string) (bool, error) {
	result, err := s.q.StoreSetDeclaration(ctx, sqlc.StoreSetDeclarationParams{
		Now:                   time.Now().UnixMilli(),
		DeclarationIdentifier: declarationID,
		SetName:               setName,
	})
	if err != nil {
		return false, err
	}
	return resultChangedRows(result)
}

// RemoveSetDeclaration removes the association between a declaration and a set.
// See also the storage package for documentation on the storage interfaces.
func (s *PgSQLStorage) RemoveSetDeclaration(ctx context.Context, setName, declarationID string) (bool, error) {
	result, err := s.q.RemoveSetDeclaration(ctx, sqlc.RemoveSetDeclarationParams{
		SetName:               setName,
		DeclarationIdentifier: declarationID,
	})
	if err != nil {
		return false, err
	}
	return resultChangedRows(result)
}

// RetrieveSets retrieves the list of sets.
// See also the storage package for documentation on the storage interfaces.
func (s *PgSQLStorage) RetrieveSets(ctx context.Context) ([]string, error) {
	return s.q.GetSets(ctx)
}
