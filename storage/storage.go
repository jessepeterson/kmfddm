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

type StatusStorer interface {
	// StoreDeclarationStatus stores the status report details.
	// For later retrieval by the StatusAPIStorage interface(s).
	StoreDeclarationStatus(ctx context.Context, enrollmentID string, status *ddm.StatusReport) error
}

type DeclarationsRetriever interface {
	// RetrieveDeclarations retrieves a list of all declarations.
	RetrieveDeclarations(ctx context.Context) ([]string, error)
}

// DeclarationAPIStorage are storage interfaces relating to declarations.
type DeclarationAPIStorage interface {
	Toucher
	DeclarationStorer
	DeclarationDeleter
	DeclarationAPIRetriever
	DeclarationsRetriever
}

type DeclarationSetRetriever interface {
	// RetrieveDeclarationSets retrieves the list of set names for declarationID.
	RetrieveDeclarationSets(ctx context.Context, declarationID string) (setNames []string, err error)
}

type SetDeclarationsRetriever interface {
	// RetrieveSetDeclarations retreives the list of declarations IDs for setName.
	RetrieveSetDeclarations(ctx context.Context, setName string) (declarationIDs []string, err error)
}

type SetDeclarationStorer interface {
	// StoreSetDeclaration associates setName and declarationID.
	StoreSetDeclaration(ctx context.Context, setName, declarationID string) (bool, error)
}

type SetDeclarationRemover interface {
	// StoreSetDeclaration dissociates setName and declarationID.
	RemoveSetDeclaration(ctx context.Context, setName, declarationID string) (bool, error)
}

// SetStorage are storage interfaces related to sets.
type SetDeclarationStorage interface {
	DeclarationSetRetriever
	SetDeclarationsRetriever
	SetDeclarationStorer
	SetDeclarationRemover
}

type SetRetreiver interface {
	// RetrieveSets returns the list of all sets.
	RetrieveSets(ctx context.Context) ([]string, error)
}

type EnrollmentSetsRetriever interface {
	// RetrieveEnrollmentSets retrieves the sets that are associated with enrollmentID.
	RetrieveEnrollmentSets(ctx context.Context, enrollmentID string) (setNames []string, err error)
}

type EnrollmentSetStorer interface {
	// StoreEnrollmentSet associates enrollmentID and setName.
	StoreEnrollmentSet(ctx context.Context, enrollmentID, setName string) (bool, error)
}

type EnrollmentSetRemover interface {
	// StoreEnrollmentSet dissociates enrollmentID and setName.
	RemoveEnrollmentSet(ctx context.Context, enrollmentID, setName string) (bool, error)
}

// EnrollmentSetStorage are storage interfaces related to MDM enrollment IDs.
type EnrollmentSetStorage interface {
	EnrollmentSetsRetriever
	EnrollmentSetStorer
	EnrollmentSetRemover
}

type StatusDeclarationsRetriever interface {
	// RetrieveDeclarationStatus retrieves the status of the declarations for enrollmentIDs.
	RetrieveDeclarationStatus(ctx context.Context, enrollmentIDs []string) (map[string][]ddm.DeclarationQueryStatus, error)
}

type StatusErrorsRetriever interface {
	// RetrieveStatusErrors retrieves the collected errors for enrollmentIDs.
	RetrieveStatusErrors(ctx context.Context, enrollmentIDs []string, offset, limit int) (map[string][]StatusError, error)
}

type StatusValuesRetriever interface {
	// RetrieveStatusErrors retrieves the collected errors for enrollmentIDs.
	RetrieveStatusValues(ctx context.Context, enrollmentIDs []string, pathPrefix string) (map[string][]StatusValue, error)
}

// StatusAPIStorage are storage interfaces related to retrieving status channel data.
type StatusAPIStorage interface {
	StatusDeclarationsRetriever
	StatusErrorsRetriever
	StatusValuesRetriever
}
