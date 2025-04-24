package storage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jessepeterson/kmfddm/ddm/build"
)

// buildItems retrieves the declaration items for enrollmentID using s and builds using b.
func buildItems(ctx context.Context, s EnrollmentDeclarationDataStorage, b build.Builder, enrollmentID string) error {
	decs, err := s.RetrieveDeclarationItems(ctx, enrollmentID)
	if err != nil {
		return fmt.Errorf("retrieving declaration items: %w", err)
	}
	for _, d := range decs {
		b.Add(d)
	}
	b.Finalize()
	return nil
}

// TokensJSON returns the declaration synchronization token JSON for enrollmentID.
// The token JSON is dynamically build using s and newHash.
func TokensJSON(ctx context.Context, s EnrollmentDeclarationDataStorage, enrollmentID string, newHash build.NewHash) ([]byte, error) {
	b := build.NewTokensBuilder(newHash)
	if err := buildItems(ctx, s, b, enrollmentID); err != nil {
		return nil, fmt.Errorf("building items: %w", err)
	}
	return json.Marshal(&b.TokensResponse)
}

// DeclarationsItemsJSON returns the declaration items JSON for enrollmentID.
// The JSON is dynamically built for enrollmentID using s and newHash.
func DeclarationItemsJSON(ctx context.Context, s EnrollmentDeclarationDataStorage, enrollmentID string, newHash build.NewHash) ([]byte, error) {
	b := build.NewDIBuilder(newHash)
	if err := buildItems(ctx, s, b, enrollmentID); err != nil {
		return nil, fmt.Errorf("building items: %w", err)
	}
	return json.Marshal(&b.DeclarationItems)
}

// JSONAdapt generates sync token and declaration items JSON from declaration data.
type JSONAdapt struct {
	store   EnrollmentDeclarationDataStorage
	newHash build.NewHash
}

// NewJSONAdapt creates a new declaration storage adapter.
func NewJSONAdapt(store EnrollmentDeclarationDataStorage, newHash build.NewHash) *JSONAdapt {
	if store == nil {
		panic("nil store")
	}
	if newHash == nil {
		panic("nil hasher")
	}
	if newHash() == nil {
		panic("nil hash")
	}
	return &JSONAdapt{store: store, newHash: newHash}
}

// RetrieveTokensJSON returns the declaration synchronization token JSON for enrollmentID.
// The JSON is dynamically built for enrollmentID.
func (a *JSONAdapt) RetrieveTokensJSON(ctx context.Context, enrollmentID string) ([]byte, error) {
	return TokensJSON(ctx, a.store, enrollmentID, a.newHash)
}

// RetrieveDeclarationItemsJSON returns the declaration items JSON for enrollmentID.
// The JSON is dynamically built for enrollmentID.
func (a *JSONAdapt) RetrieveDeclarationItemsJSON(ctx context.Context, enrollmentID string) ([]byte, error) {
	return DeclarationItemsJSON(ctx, a.store, enrollmentID, a.newHash)
}

// RetrieveEnrollmentDeclarationJSON returns a JSON declaration for
// enrollmentID identified by declarationID and declarationType.
// The JSON is relayed from the underlying storage as-is.
func (k *JSONAdapt) RetrieveEnrollmentDeclarationJSON(ctx context.Context, declarationID, declarationType, enrollmentID string) ([]byte, error) {
	return k.store.RetrieveEnrollmentDeclarationJSON(ctx, declarationID, declarationType, enrollmentID)
}
