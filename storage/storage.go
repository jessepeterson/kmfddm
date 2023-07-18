// Package storage defines shared types and interfaces for storage.
package storage

import (
	"context"

	"github.com/jessepeterson/kmfddm/ddm"
)

type Toucher interface {
	// TouchDeclaration forces a change to a declaration's ServerToken only.
	TouchDeclaration(ctx context.Context, declarationID string) error
}

type DeclarationStorer interface {
	// StoreDeclaration stores a declaration.
	//
	// Note that a storage backend may tried to creation relations
	// based on the the ddm.IdentifierRefs field.
	StoreDeclaration(ctx context.Context, d *ddm.Declaration) (bool, error)
}

type DeclarationDeleter interface {
	// DeleteDeclaration deletes a declaration.
	//
	// Implementations should return an error if there are declarations
	// that depend on it or if it is in a set.
	DeleteDeclaration(ctx context.Context, declarationID string) (bool, error)
}

type DeclarationAPIRetriever interface {
	RetrieveDeclaration(ctx context.Context, declarationID string) (*ddm.Declaration, error)
}

// DeclarationAPIStorage are storage interfaces relating to declaration APIs.
type DeclarationAPIStorage interface {
	Toucher
	DeclarationStorer
	DeclarationDeleter
	DeclarationAPIRetriever
}

type EnrollmentIDRetriever interface {
	// RetrieveEnrollmentIDs retrieves MDM enrollment IDs from storage.
	// In the case of sets and declarations the transitive associations
	// are traversed to try and collect the IDs. When multiple slices
	// are given they should be treated like a logical or (i.e. finding
	// all enrollment IDs for any of the given slices).
	// Warning: the results may be very large for e.g. sets (or, transitively,
	// declarations) that are assigned to many enrollment IDs.
	RetrieveEnrollmentIDs(ctx context.Context, declarations []string, sets []string, ids []string) ([]string, error)
}

type TokensJSONRetriever interface {
	// RetrieveTokensJSON returns the token JSON for an enrollment ID.
	RetrieveTokensJSON(ctx context.Context, enrollmentID string) ([]byte, error)
}

type TokensDeclarationItemsRetriever interface {
	// RetrieveDeclarationItemsJSON returns the declaration items JSON for an enrollment ID.
	RetrieveDeclarationItemsJSON(ctx context.Context, enrollmentID string) ([]byte, error)
	TokensJSONRetriever
}

type DeclarationRetriever interface {
	// RetrieveEnrollmentDeclarationJSON fetches the JSON for a declaration for an enrollment ID.
	RetrieveEnrollmentDeclarationJSON(ctx context.Context, declarationID, declarationType, enrollmentID string) ([]byte, error)
}

// EnrollmentDeclarationStorage is the storage required to support declarations in the DDM protocol.
type EnrollmentDeclarationStorage interface {
	TokensDeclarationItemsRetriever
	DeclarationRetriever
}

type StatusStorage interface {
	StoreDeclarationStatus(ctx context.Context, enrollmentID string, status *ddm.StatusReport) error
}
