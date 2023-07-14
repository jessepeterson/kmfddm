// Package storage defines shared types and interfaces for storage.
package storage

import (
	"context"
)

type Toucher interface {
	// TouchDeclaration forces a change only to a declaration's ServerToken.
	TouchDeclaration(ctx context.Context, declarationID string) error
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
	RetrieveTokensJSON(ctx context.Context, enrollmentID string) ([]byte, error)
}

type TokensDeclarationItemsRetriever interface {
	RetrieveDeclarationItemsJSON(ctx context.Context, enrollmentID string) ([]byte, error)
	TokensJSONRetriever
}

type DeclarationRetriever interface {
	// RetrieveEnrollmentDeclarationJSON fetches the JSON for a declaration for an enrollment ID.
	RetrieveEnrollmentDeclarationJSON(ctx context.Context, declarationID, declarationType, enrollmentID string) ([]byte, error)
}
