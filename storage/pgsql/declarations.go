package pgsql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jessepeterson/kmfddm/ddm"
	"github.com/jessepeterson/kmfddm/storage"
	"github.com/jessepeterson/kmfddm/storage/pgsql/sqlc"
)

// StoreDeclaration stores a declaration and returns whether it changed or not.
// See also the storage package for documentation on the storage interfaces.
func (s *PgSQLStorage) StoreDeclaration(ctx context.Context, d *ddm.Declaration) (bool, error) {
	result, err := s.q.StoreDeclaration(ctx, sqlc.StoreDeclarationParams{
		Now:        time.Now().UnixMilli(),
		Identifier: d.Identifier,
		Type:       d.Type,
		Payload:    string(d.Payload),
	})
	if err != nil {
		return false, err
	}
	return resultChangedRows(result)
}

// RetrieveDeclaration retrieves a declaration.
// See also the storage package for documentation on the storage interfaces.
func (s *PgSQLStorage) RetrieveDeclaration(ctx context.Context, declarationID string) (*ddm.Declaration, error) {
	qd, err := s.q.GetDeclaration(ctx, declarationID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: %v", storage.ErrDeclarationNotFound, err)
	} else if err != nil {
		return nil, err
	}
	return &ddm.Declaration{
		Identifier:  qd.Identifier,
		Type:        qd.Type,
		ServerToken: qd.ServerToken,
		Payload:     []byte(qd.Payload),
		Raw:         []byte(qd.Declaration),
	}, nil
}

// RetrieveDeclarationModTime retrieves the last modification time of the declaration.
// See also the storage package for documentation on the storage interfaces.
func (s *PgSQLStorage) RetrieveDeclarationModTime(ctx context.Context, declarationID string) (time.Time, error) {
	updatedAt, err := s.q.GetDeclarationModTime(ctx, declarationID)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, fmt.Errorf("%w: %v", storage.ErrDeclarationNotFound, err)
	} else if err != nil {
		return time.Time{}, err
	}
	return time.UnixMilli(updatedAt), nil
}

// DeleteDeclaration deletes a declaration and returns whether it was deleted or already existed.
// See also the storage package for documentation on the storage interfaces.
func (s *PgSQLStorage) DeleteDeclaration(ctx context.Context, declarationID string) (bool, error) {
	result, err := s.q.DeleteDeclaration(ctx, declarationID)
	if err != nil {
		return false, err
	}
	return resultChangedRows(result)
}

// RetrieveDeclarationSets returns the list of sets a declaration is a part of.
// See also the storage package for documentation on the storage interfaces.
func (s *PgSQLStorage) RetrieveDeclarationSets(ctx context.Context, declarationID string) ([]string, error) {
	return s.q.GetDeclarationSets(ctx, declarationID)
}

// RetrieveDeclarations returns the list of declaration IDs.
// See also the storage package for documentation on the storage interfaces.
func (s *PgSQLStorage) RetrieveDeclarations(ctx context.Context) ([]string, error) {
	return s.q.GetDeclarationIdentifiers(ctx)
}

// TouchDeclaration updates a declaration's "touch count" which makes a new server token.
// See also the storage package for documentation on the storage interfaces.
func (s *PgSQLStorage) TouchDeclaration(ctx context.Context, declarationID string) error {
	result, err := s.q.TouchDeclaration(ctx, sqlc.TouchDeclarationParams{
		Identifier: declarationID,
		Now:        time.Now().UnixMilli(),
	})
	if err != nil {
		return err
	}
	changed, err := resultChangedRows(result)
	if err != nil {
		return err
	}
	if !changed {
		return fmt.Errorf("%w: declaration not touched (may not exist)", storage.ErrDeclarationNotFound)
	}
	return nil
}
