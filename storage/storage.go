// Package storage defines shared types and interfaces for storage.
package storage

import (
	"context"
	"time"

	"github.com/jessepeterson/kmfddm/ddm"
)

type Toucher interface {
	// TouchDeclaration forces a change to a declaration's ServerToken only.
	TouchDeclaration(ctx context.Context, declarationID string) error
}

type DeclarationStorer interface {
	// StoreDeclaration stores a declaration.
	// If the declaration is new or has changed true should be returned.
	//
	// Note that a storage backend may try to create relations
	// based on the the ddm.IdentifierRefs field.
	StoreDeclaration(ctx context.Context, d *ddm.Declaration) (bool, error)
}

type DeclarationDeleter interface {
	// DeleteDeclaration deletes a declaration.
	// If the declaration was deleted true should be returned.
	//
	// Implementations should return an error if there are declarations
	// that depend on it or if the declaration is associated with a set.
	DeleteDeclaration(ctx context.Context, declarationID string) (bool, error)
}

type DeclarationAPIRetriever interface {
	// RetrieveDeclaration retrieves a declaration from storage.
	RetrieveDeclaration(ctx context.Context, declarationID string) (*ddm.Declaration, error)
	// RetrieveDeclarationModTime retrieves the last modification time of the declaration.
	RetrieveDeclarationModTime(ctx context.Context, declarationID string) (time.Time, error)
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
	// If the association is created true should be returned.
	// It should not be an error if the association does not exist.
	StoreSetDeclaration(ctx context.Context, setName, declarationID string) (bool, error)
}

type SetDeclarationRemover interface {
	// StoreSetDeclaration dissociates setName and declarationID.
	// If the association is removed true should be returned.
	// It should not be an error if the association does not exist.
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
	// If the association is created true is returned.
	// It should not be an error if the association does not exist.
	StoreEnrollmentSet(ctx context.Context, enrollmentID, setName string) (bool, error)
}

type EnrollmentSetRemover interface {
	// RemoveEnrollmentSet dissociates enrollmentID and setName.
	// If the association is removed true is returned.
	// It should not be an error if the association does not exist.
	RemoveEnrollmentSet(ctx context.Context, enrollmentID, setName string) (bool, error)

	// RemoveAllEnrollmentSets dissociates enrollment ID from any sets.
	// If any associations are removed true is returned.
	// It should not be an error if no associations exist.
	RemoveAllEnrollmentSets(ctx context.Context, enrollmentID string) (bool, error)
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

type StatusReportRetriever interface {
	RetrieveStatusReport(ctx context.Context, q StatusReportQuery) (*StoredStatusReport, error)
}

// StatusAPIStorage are storage interfaces related to retrieving status channel data.
type StatusAPIStorage interface {
	StatusDeclarationsRetriever
	StatusErrorsRetriever
	StatusValuesRetriever
	StatusReportRetriever
}
