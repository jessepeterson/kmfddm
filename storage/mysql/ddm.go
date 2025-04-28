package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jessepeterson/kmfddm/ddm"
	"github.com/jessepeterson/kmfddm/ddm/build"
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

func (s *MySQLStorage) build(ctx context.Context, b build.Builder, enrollmentID string) error {
	items, err := s.q.GetManifestItems(ctx, enrollmentID)
	if err != nil {
		return err
	}
	for _, i := range items {
		// note that we're selecting and assembling a very minimal Declaration
		// here. just enough to work with the builder interface. check the
		// builder implementation to make sure it doesn't need anything more
		// than what we're giving it.
		b.Add(&ddm.Declaration{
			Identifier:  i.Identifier,
			Type:        i.Type,
			ServerToken: i.ServerToken,
		})
	}
	b.Finalize()
	return nil
}

// RetrieveDeclarationItemsJSON generates Declaration Items for enrollmentID.
// See also the storage package for documentation on the storage interfaces.
func (s *MySQLStorage) RetrieveDeclarationItemsJSON(ctx context.Context, enrollmentID string) ([]byte, error) {
	di := build.NewDIBuilder(s.newHash)
	err := s.build(ctx, di, enrollmentID)
	if err != nil {
		return nil, err
	}
	return json.Marshal(&di.DeclarationItems)
}

// RetrieveTokensJSON generates Sync Tokens for enrollmentID.
// See also the storage package for documentation on the storage interfaces.
func (s *MySQLStorage) RetrieveTokensJSON(ctx context.Context, enrollmentID string) ([]byte, error) {
	b := build.NewTokensBuilder(s.newHash)
	err := s.build(ctx, b, enrollmentID)
	if err != nil {
		return nil, err
	}
	return json.Marshal(&b.TokensResponse)
}
