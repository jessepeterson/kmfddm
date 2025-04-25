package storage

import (
	"context"
	"errors"

	"github.com/jessepeterson/kmfddm/ddm"
)

var ErrDeclarationNotFound = errors.New("declaration not found")

// TokensJSONRetriever retrieves sync token JSON for an enrollment.
type TokensJSONRetriever interface {
	// RetrieveTokensJSON returns the declaration synchronization token JSON for enrollmentID.
	// This JSON can be returned for a "tokens" Endpoint DDM MDM check-in request.
	// Or it can be placed within a DeclarativeManagement MDM command.
	// See https://developer.apple.com/documentation/devicemanagement/tokensresponse
	// for the structure of the returned JSON document.
	RetrieveTokensJSON(ctx context.Context, enrollmentID string) ([]byte, error)
}

// DeclarationItemsRetriever retrieves declarations items JSON for an enrollment.
type DeclarationItemsJSONRetriever interface {
	// RetrieveDeclarationItemsJSON returns the declaration items JSON for enrollmentID.
	// This JSON is returned for a "declaration-items" Endpoint DDM MDM check-in request.
	// See https://developer.apple.com/documentation/devicemanagement/declarationitemsresponse
	// for the structure of the returned JSON document.
	RetrieveDeclarationItemsJSON(ctx context.Context, enrollmentID string) ([]byte, error)
}

// TokensDeclarationItemsRetriever retrieves declaration items or sync token JSON for an enrollment.
type TokensDeclarationItemsStorage interface {
	DeclarationItemsJSONRetriever
	TokensJSONRetriever
}

// DeclarationRetriever retrieves a JSON declaration for an enrollment.
type DeclarationJSONRetriever interface {
	// RetrieveEnrollmentDeclarationJSON returns a JSON declaration for
	// enrollmentID identified by declarationID and declarationType.
	// The storage backend should make sure that the declaration
	// identified by declaration ID is of the declarationType and is
	// transitively associated to the given enrollmentID.
	// If the declaration is not found under these constraints then
	// [ErrDeclarationNotFound] should be returned.
	// This JSON is returned for a "declaration/.../..." Endpoint DDM MDM check-in request.
	RetrieveEnrollmentDeclarationJSON(ctx context.Context, declarationID, declarationType, enrollmentID string) ([]byte, error)
}

// EnrollmentDeclarationStorage is the storage required to support declarations in the DDM protocol.
// This is part of the core DDM protocol for handling declarations for enrollments.
type EnrollmentDeclarationStorage interface {
	TokensDeclarationItemsStorage
	DeclarationJSONRetriever
}

// EnrollmentDeclarationDataStorage is the storage required to support dynamic declartaions in the DDM protocol.
type EnrollmentDeclarationDataStorage interface {
	// RetrieveDeclarationItems retrieves the declarations for enrollmentID.
	// The returned declarations need only include the identifier, type, and server token.
	// Sync token or declaration items JSON will likely be constructed using the returned declarations.
	RetrieveDeclarationItems(ctx context.Context, enrollmentID string) ([]*ddm.Declaration, error)

	DeclarationJSONRetriever
}
