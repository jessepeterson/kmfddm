package storage

import (
	"context"

	"github.com/jessepeterson/kmfddm/ddm"
)

// TokensJSONRetriever retrieves sync token JSON for an enrollment.
type TokensJSONRetriever interface {
	// RetrieveTokensJSON returns the declaration synchronization token JSON for enrollmentID.
	// This JSON can be placed in a DeclarativeManagement MDM command
	// or returned for a "tokens" Endpoint DDM MDM check-in request.
	// See https://developer.apple.com/documentation/devicemanagement/tokensresponse
	// for the structure of the returned JSON document.
	RetrieveTokensJSON(ctx context.Context, enrollmentID string) ([]byte, error)
}

// TokensDeclarationItemsRetriever retrieves declaration items or sync token JSON for an enrollment.
type TokensDeclarationItemsRetriever interface {
	// RetrieveDeclarationItemsJSON returns the declaration items JSON for enrollmentID.
	// This JSON is returned for a "declaration-items" Endpoint DDM MDM check-in request.
	// See https://developer.apple.com/documentation/devicemanagement/declarationitemsresponse
	// for the structure of the returned JSON document.
	RetrieveDeclarationItemsJSON(ctx context.Context, enrollmentID string) ([]byte, error)

	TokensJSONRetriever
}

// DeclarationRetriever retrieves a JSON declaration for an enrollment.
type DeclarationRetriever interface {
	// RetrieveEnrollmentDeclarationJSON returns a JSON declaration for
	// enrollmentID identified by declarationID and declarationType.
	// If the declaration is not found under these constraints then
	// ErrDeclarationNotFound should be returned.
	// This JSON is returned for a "declaration/.../..." Endpoint DDM MDM check-in request.
	RetrieveEnrollmentDeclarationJSON(ctx context.Context, declarationID, declarationType, enrollmentID string) ([]byte, error)
}

// EnrollmentDeclarationStorage is the storage required to support declarations in the DDM protocol.
// This is part of the core DDM protocol for handling declarations for enrollments.
type EnrollmentDeclarationStorage interface {
	TokensDeclarationItemsRetriever
	DeclarationRetriever
}

// EnrollmentDeclarationDataStorage
type EnrollmentDeclarationDataStorage interface {
	// RetrieveDeclarationItems retrieves the declarations for enrollmentID.
	// The returned declarations may omit the payload JSON, identifier
	// references, and raw data.
	// Sync token or declaration items JSON may be constructed using the returned declarations.
	RetrieveDeclarationItems(ctx context.Context, enrollmentID string) ([]*ddm.Declaration, error)

	DeclarationRetriever
}
