package storage

import (
	"context"
	"errors"

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
func (s *Multi) RetrieveDeclarationItems(ctx context.Context, enrollmentID string) ([]*ddm.Declaration, error) {
	var allDecls []*ddm.Declaration
	var err error
	for _, s := range s.storage {
		decls, err := s.RetrieveDeclarationItems(ctx, enrollmentID)
		if err != nil {
			break
		}
		allDecls = append(allDecls, decls...)
	}
	return allDecls, err
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
