package kv

import (
	"context"
	"errors"
	"fmt"

	"github.com/jessepeterson/kmfddm/ddm"
	"github.com/jessepeterson/kmfddm/storage"

	"github.com/micromdm/nanolib/storage/kv"
)

// RetrieveTokensJSON returns the declaration synchronization token JSON for enrollmentID.
// It builds the sync tokens from calling [RetrieveDeclarationItems].
func (s *KV) RetrieveTokensJSON(ctx context.Context, enrollmentID string) ([]byte, error) {
	return storage.TokensJSON(ctx, s, enrollmentID, s.newHash)
}

// RetrieveDeclarationItemsJSON returns the declaration items JSON for enrollmentID.
// It builds the sync tokens from calling [RetrieveDeclarationItems].
func (s *KV) RetrieveDeclarationItemsJSON(ctx context.Context, enrollmentID string) ([]byte, error) {
	return storage.DeclarationItemsJSON(ctx, s, enrollmentID, s.newHash)
}

func (s *KV) enrollmentCanAccessDeclaration(ctx context.Context, declarationID, enrollmentID string) (bool, error) {
	// lookup enrollment sets
	enrSets, err := getEnrollmentSets(ctx, s.enrollments, enrollmentID)
	if err != nil {
		return false, err
	}

	found := false
	for _, setName := range enrSets {
		cancel := make(chan struct{})
		for searchID := range s.sets.KeysPrefix(ctx, join(keyPfxSetDcl, setName)+keySep, cancel) {
			if declarationID == searchID[len(join(keyPfxSetDcl, setName)+keySep):] {
				found = true
				close(cancel)
				goto found
			}
		}
	}
found:

	return found, nil
}

// RetrieveEnrollmentDeclarationJSON returns a JSON declaration for
// enrollmentID identified by declarationID and declarationType.
// If the declaration is not found under these constraints then
// ErrDeclarationNotFound should be returned.
// This JSON is returned for a "declaration/.../..." Endpoint DDM MDM check-in request.
func (s *KV) RetrieveEnrollmentDeclarationJSON(ctx context.Context, declarationID, declarationType, enrollmentID string) (declationJSON []byte, err error) {
	if dType, err := s.declarations.Get(ctx, join(keyPfxDcl, declarationID, keyDeclarationType)); errors.Is(err, kv.ErrKeyNotFound) {
		// wrap kv error in the proper storage error
		return nil, fmt.Errorf("%w: %v", storage.ErrDeclarationNotFound, err)
	} else if err != nil {
		return nil, err
	} else {
		if declarationType != ddm.ManifestType(string(dType)) {
			return nil, fmt.Errorf("%w: incorrect type: %s", storage.ErrDeclarationNotFound, declarationType)
		}
	}

	if ok, err := s.enrollmentCanAccessDeclaration(ctx, declarationID, enrollmentID); err != nil {
		return nil, fmt.Errorf("checking transitive access: %w", err)
	} else if !ok {
		return nil, fmt.Errorf("%w: no transitive access", storage.ErrDeclarationNotFound)
	}

	d, err := s.RetrieveDeclaration(ctx, declarationID)
	if err != nil {
		return nil, err
	}
	if d != nil && len(d.Raw) > 0 {
		return d.Raw, err
	}
	return nil, fmt.Errorf("%w: empty declaration", storage.ErrDeclarationNotFound)
}

// RetrieveDeclarationItems retrieves the declarations for enrollmentID.
// The returned declarations should only contain the identifier,
// type, and server token.
func (s *KV) RetrieveDeclarationItems(ctx context.Context, enrollmentID string) ([]*ddm.Declaration, error) {
	dMap := make(map[string]struct{})

	// get all the sets for this enrollment ID
	setNames, err := getEnrollmentSets(ctx, s.enrollments, enrollmentID)
	if err != nil {
		return nil, err
	}

	for _, setName := range setNames {
		// get all the declarations for this set
		declarationIDs, err := getSetDeclarations(ctx, s.sets, setName)
		if err != nil {
			return nil, err
		}

		for _, declarationID := range declarationIDs {
			dMap[declarationID] = struct{}{}
		}
	}

	var declarations []*ddm.Declaration

	for declarationID := range dMap {
		dValues, err := kv.GetMap(ctx, s.declarations, []string{
			join(keyPfxDcl, declarationID, keyDeclarationServerToken),
			join(keyPfxDcl, declarationID, keyDeclarationType),
		})
		if err != nil {
			return nil, err
		}
		declarations = append(declarations, &ddm.Declaration{
			Identifier:  declarationID,
			Type:        string(dValues[join(keyPfxDcl, declarationID, keyDeclarationType)]),
			ServerToken: string(dValues[join(keyPfxDcl, declarationID, keyDeclarationServerToken)]),
		})
	}

	return declarations, nil
}
