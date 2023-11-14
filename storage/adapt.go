package storage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jessepeterson/kmfddm/ddm/build"
)

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

// build retrieves the declaration items for enrollmentID and builds using b.
func (a *JSONAdapt) build(ctx context.Context, b build.Builder, enrollmentID string) error {
	declarations, err := a.store.RetrieveDeclarationItems(ctx, enrollmentID)
	if err != nil {
		return fmt.Errorf("retrieving items: %w", err)
	}
	for _, d := range declarations {
		b.Add(d)
	}
	b.Finalize()
	return nil
}

// RetrieveTokensJSON returns the declaration synchronization token JSON for enrollmentID.
// The JSON is dynamically built for enrollmentID.
func (a *JSONAdapt) RetrieveTokensJSON(ctx context.Context, enrollmentID string) ([]byte, error) {
	b := build.NewTokensBuilder(a.newHash)
	err := a.build(ctx, b, enrollmentID)
	if err != nil {
		return nil, fmt.Errorf("building items: %w", err)
	}
	return json.Marshal(&b.TokensResponse)
}

// RetrieveDeclarationItemsJSON returns the declaration items JSON for enrollmentID.
// The JSON is dynamically built for enrollmentID.
func (a *JSONAdapt) RetrieveDeclarationItemsJSON(ctx context.Context, enrollmentID string) ([]byte, error) {
	b := build.NewDIBuilder(a.newHash)
	err := a.build(ctx, b, enrollmentID)
	if err != nil {
		return nil, fmt.Errorf("building items: %w", err)
	}
	return json.Marshal(&b.DeclarationItems)
}

// RetrieveEnrollmentDeclarationJSON returns a JSON declaration for
// enrollmentID identified by declarationID and declarationType.
// The JSON is relayed from the underlying storage as-is.
func (k *JSONAdapt) RetrieveEnrollmentDeclarationJSON(ctx context.Context, declarationID, declarationType, enrollmentID string) ([]byte, error) {
	return k.store.RetrieveEnrollmentDeclarationJSON(ctx, declarationID, declarationType, enrollmentID)
}
