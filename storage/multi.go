package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/jessepeterson/kmfddm/ddm"
)

// Multi adapts and combines multiple storage backends.
// The intent is to provide a "unified" view of multiple backends
// as a single set of declarations and declaration items.
type Multi struct {
	storage []EnrollmentDeclarationDataStorage
}

// NewMulti creates a new multi storage adapter using s backing stores.
// Note the stores are consulted in slice order—i.e. the first store "wins."
func NewMulti(s ...EnrollmentDeclarationDataStorage) *Multi {
	return &Multi{storage: s}
}

// RetrieveDeclarationItems combines the declarations for enrollmentID from each backing store.
// If any backing store errors no declarations are returned: a partial set of
// declaration items would instruct an enrollment to unload the missing declarations.
func (s *Multi) RetrieveDeclarationItems(ctx context.Context, enrollmentID string) ([]*ddm.Declaration, error) {
	var allDecls []*ddm.Declaration
	for i, store := range s.storage {
		decls, err := store.RetrieveDeclarationItems(ctx, enrollmentID)
		if err != nil {
			return nil, fmt.Errorf("retrieving declaration items from store %d: %w", i, err)
		}
		allDecls = append(allDecls, decls...)
	}
	return allDecls, nil
}

// RetrieveEnrollmentDeclarationJSON returns a JSON declaration for
// enrollmentID identified by declarationID and declarationType.
// The first found declaration is returned in order of backing stores.
func (s *Multi) RetrieveEnrollmentDeclarationJSON(ctx context.Context, declarationID, declarationType, enrollmentID string) ([]byte, error) {
	var declarationBytes []byte
	var err error = ErrDeclarationNotFound
	for _, s := range s.storage {
		declarationBytes, err = s.RetrieveEnrollmentDeclarationJSON(ctx, declarationID, declarationType, enrollmentID)
		if errors.Is(err, ErrDeclarationNotFound) {
			// skip to the next
			continue
		}
		// this will leave intact any err, or a nil err and the return bytes
		break
	}
	return declarationBytes, err
}
