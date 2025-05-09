package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jessepeterson/kmfddm/ddm"
	"github.com/jessepeterson/kmfddm/storage"
	"github.com/jessepeterson/kmfddm/storage/mysql/sqlc"
)

// RetrieveEnrollmentDeclarationJSON retreives a declaration intended for a
// given enrollment. As it is intended to retrieve the declaration for
// delivery to a specific enrollment it queries to make sure that enrollment
// should have access and that it is of the correct type.
// See also the storage package for documentation on the storage interfaces.
func (s *MySQLStorage) RetrieveEnrollmentDeclarationJSON(ctx context.Context, declarationID, declarationType, enrollmentID string) (raw []byte, err error) {
	// we JOIN against the enrollments table to make sure only those
	// declarations that are transitively related are able to be
	// accessed. kinda-sorta like an ACL. almost.
	raw, err = s.q.GetDDMDeclaration(ctx, sqlc.GetDDMDeclarationParams{
		Identifier:   declarationID,
		Type:         "com.apple." + declarationType + ".%",
		EnrollmentID: enrollmentID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return raw, fmt.Errorf("%w: %v", storage.ErrDeclarationNotFound, err)
	}
	return
}

// RetrieveDeclarationItems retrieves the declarations for enrollmentID.
func (s *MySQLStorage) RetrieveDeclarationItems(ctx context.Context, enrollmentID string) ([]*ddm.Declaration, error) {
	items, err := s.q.GetManifestItems(ctx, enrollmentID)
	if err != nil {
		return nil, err
	}
	// reform the results
	var decls []*ddm.Declaration
	for _, i := range items {
		decls = append(decls, &ddm.Declaration{
			Identifier:  i.Identifier,
			Type:        i.Type,
			ServerToken: i.ServerToken,
		})
	}
	return decls, nil
}

// RetrieveDeclarationItemsJSON generates Declaration Items for enrollmentID.
// See also the storage package for documentation on the storage interfaces.
func (s *MySQLStorage) RetrieveDeclarationItemsJSON(ctx context.Context, enrollmentID string) ([]byte, error) {
	return storage.DeclarationItemsJSON(ctx, s, enrollmentID, s.newHash)
}

// RetrieveTokensJSON generates Sync Tokens for enrollmentID.
// See also the storage package for documentation on the storage interfaces.
func (s *MySQLStorage) RetrieveTokensJSON(ctx context.Context, enrollmentID string) ([]byte, error) {
	return storage.TokensJSON(ctx, s, enrollmentID, s.newHash)
}
