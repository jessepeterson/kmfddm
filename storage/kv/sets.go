package kv

import (
	"context"

	"github.com/jessepeterson/kmfddm/storage"

	"github.com/micromdm/nanolib/storage/kv"
)

const (
	keyPfxSetDcl = "sc"
	keyPfxDclSet = "ds"
	keyPfxSet    = "st"
)

// b should nominally be s.sets, but may be a txn of such
func getDeclarationSets(ctx context.Context, b kv.KeysPrefixTraverser, declarationID string) (setNames []string, err error) {
	pfx := keyPfxDclSet + keySep + declarationID + keySep
	for key := range b.KeysPrefix(ctx, pfx, nil) {
		setNames = append(setNames, key[len(pfx):])
	}
	return
}

// b should nominally be s.sets, but may be a txn of such
func getSetDeclarations(ctx context.Context, b kv.KeysPrefixTraverser, setName string) (declarationIDs []string, err error) {
	pfx := keyPfxSetDcl + keySep + setName + keySep
	for key := range b.KeysPrefix(ctx, pfx, nil) {
		declarationIDs = append(declarationIDs, key[len(pfx):])
	}
	return
}

// RetrieveDeclarationSets retrieves the list of set names for declarationID.
func (s *KV) RetrieveDeclarationSets(ctx context.Context, declarationID string) (setNames []string, err error) {
	return getDeclarationSets(ctx, s.sets, declarationID)
}

// RetrieveSetDeclarations retreives the list of declarations IDs for setName.
func (s *KV) RetrieveSetDeclarations(ctx context.Context, setName string) (declarationIDs []string, err error) {
	return getSetDeclarations(ctx, s.sets, setName)
}

// StoreSetDeclaration associates setName and declarationID.
// If the association is created true should be returned.
// It should not be an error if the association does not exist.
func (s *KV) StoreSetDeclaration(ctx context.Context, setName, declarationID string) (changed bool, err error) {
	// check that the declaration exists first
	if found, err := s.declarations.Has(ctx, join(keyPfxDcl, declarationID, keyDeclarationType)); err != nil {
		return false, err
	} else if !found {
		return false, storage.ErrDeclarationNotFound
	}

	err = kv.PerformCRUDBucketTxn(ctx, s.sets, func(ctx context.Context, b kv.CRUDBucket) error {
		// check that it exists first to tell whether this was a change
		if found, err := b.Has(ctx, join(keyPfxSetDcl, setName, declarationID)); err != nil {
			return err
		} else if !found {
			changed = true
		}

		return kv.SetMap(ctx, b, map[string][]byte{
			join(keyPfxSetDcl, setName, declarationID): []byte(valueSet),
			join(keyPfxDclSet, declarationID, setName): []byte(valueSet),
			join(keyPfxSet, setName):                   []byte(valueSet),
		})
	})
	return
}

// RemoveSetDeclaration dissociates setName and declarationID.
// If the association is removed true should be returned.
// It should not be an error if the association does not exist.
func (s *KV) RemoveSetDeclaration(ctx context.Context, setName, declarationID string) (changed bool, err error) {
	err = kv.PerformBucketTxn(ctx, s.sets, func(ctx context.Context, b kv.Bucket) error {
		var found bool
		// check for an association first
		if found, err := b.Has(ctx, join(keyPfxSetDcl, setName, declarationID)); err != nil {
			return err
		} else if found {
			changed = true
		}

		// remove the association
		if err = kv.DeleteSlice(ctx, b, []string{
			join(keyPfxSetDcl, setName, declarationID),
			join(keyPfxDclSet, declarationID, setName),
		}); err != nil {
			return err
		}

		kClose := make(chan struct{})
		found = false
		// cycle through to see if there's any other associations for this set
		for key := range b.KeysPrefix(ctx, join(keyPfxSetDcl, setName)+keySep, kClose) {
			if key != "" {
				found = true
				close(kClose)
				break
			}
		}

		// if no others were found, then delete this set from the list of sets
		if !found {
			if err = b.Delete(ctx, join(keyPfxSet, setName)); err != nil {
				return err
			}
		}

		return nil
	})
	return
}

// RetrieveSets returns the list of all sets.
func (s *KV) RetrieveSets(ctx context.Context) (setNames []string, err error) {
	pfx := keyPfxSet + keySep
	for key := range s.sets.KeysPrefix(ctx, pfx, nil) {
		setNames = append(setNames, key[len(pfx):])
	}
	return
}
