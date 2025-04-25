package kv

import (
	"context"

	"github.com/micromdm/nanolib/storage/kv"
)

const (
	keyPfxEnrSet = "es"
	keyPfxSetEnr = "se"
	keyPfxEnr    = "en"
)

// b should nominally be s.enrollments, but may be a txn of such
func getSetEnrollments(ctx context.Context, b kv.KeysPrefixTraverser, setName string) (enrollmentIDs []string, err error) {
	pfx := keyPfxSetEnr + keySep + setName + keySep
	for key := range b.KeysPrefix(ctx, pfx, nil) {
		enrollmentIDs = append(enrollmentIDs, key[len(pfx):])
	}
	return
}

// b should nominally be s.enrollments, but may be a txn of such
func getEnrollmentSets(ctx context.Context, b kv.KeysPrefixTraverser, enrollmentID string) (enrollmentIDs []string, err error) {
	pfx := keyPfxEnrSet + keySep + enrollmentID + keySep
	for key := range b.KeysPrefix(ctx, pfx, nil) {
		enrollmentIDs = append(enrollmentIDs, key[len(pfx):])
	}
	return
}

// RetrieveEnrollmentSets retrieves the sets that are associated with enrollmentID.
func (s *KV) RetrieveEnrollmentSets(ctx context.Context, enrollmentID string) (setNames []string, err error) {
	return getEnrollmentSets(ctx, s.enrollments, enrollmentID)
}

// StoreEnrollmentSet associates enrollmentID and setName.
// If the association is created true is returned.
// It should not be an error if the association does not exist.
func (s *KV) StoreEnrollmentSet(ctx context.Context, enrollmentID, setName string) (changed bool, err error) {
	err = kv.PerformCRUDBucketTxn(ctx, s.enrollments, func(ctx context.Context, b kv.CRUDBucket) error {
		if found, err := b.Has(ctx, join(keyPfxEnrSet, enrollmentID, setName)); err != nil {
			return err
		} else if !found {
			changed = true
		}

		return kv.SetMap(ctx, b, map[string][]byte{
			join(keyPfxEnrSet, enrollmentID, setName): []byte(valueSet),
			join(keyPfxSetEnr, setName, enrollmentID): []byte(valueSet),
			join(keyPfxEnr, enrollmentID):             []byte(valueSet),
		})
	})
	return
}

// RemoveEnrollmentSet dissociates enrollmentID and setName.
// If the association is removed true is returned.
// It should not be an error if the association does not exist.
func (s *KV) RemoveEnrollmentSet(ctx context.Context, enrollmentID, setName string) (changed bool, err error) {
	err = kv.PerformBucketTxn(ctx, s.enrollments, func(ctx context.Context, b kv.Bucket) error {
		var found bool
		// see if we have an association first to tell if we'll be changing it
		if found, err := b.Has(ctx, join(keyPfxEnrSet, enrollmentID, setName)); err != nil {
			return err
		} else if found {
			changed = true
		}

		// remove the association
		if err = kv.DeleteSlice(ctx, b, []string{
			join(keyPfxEnrSet, enrollmentID, setName),
			join(keyPfxSetEnr, setName, enrollmentID),
		}); err != nil {
			return err
		}

		kClose := make(chan struct{})
		found = false
		// cycle through to see if there's any other associations for this enrollment ID
		for key := range b.KeysPrefix(ctx, join(keyPfxEnrSet, enrollmentID)+keySep, kClose) {
			if key != "" {
				found = true
				close(kClose)
				break
			}
		}

		// if no others were found, then delete this enrollment from
		// the list of enrollments that have sets.
		if !found {
			if err = b.Delete(ctx, join(keyPfxEnr, enrollmentID)); err != nil {
				return err
			}
		}

		return nil
	})
	return
}

// RemoveAllEnrollmentSets dissociates enrollment ID from any sets.
// If any associations are removed true is returned.
// It should not be an error if no associations exist.
func (s *KV) RemoveAllEnrollmentSets(ctx context.Context, enrollmentID string) (changed bool, err error) {
	err = kv.PerformBucketTxn(ctx, s.enrollments, func(ctx context.Context, b kv.Bucket) error {
		if found, err := b.Has(ctx, join(keyPfxEnr, enrollmentID)); err != nil {
			return err
		} else if found {
			changed = true
		}

		enrSets, err := getEnrollmentSets(ctx, b, enrollmentID)
		if err != nil {
			return err
		}

		for _, setName := range enrSets {
			// remove the association
			if err = kv.DeleteSlice(ctx, b, []string{
				join(keyPfxEnrSet, enrollmentID, setName),
				join(keyPfxSetEnr, setName, enrollmentID),
			}); err != nil {
				return err
			}
		}

		kClose := make(chan struct{})
		found := false
		// cycle through to see if there's any other associations for this enrollment ID
		for key := range b.KeysPrefix(ctx, join(keyPfxEnrSet, enrollmentID)+keySep, kClose) {
			if key != "" {
				found = true
				close(kClose)
				break
			}
		}

		// if no others were found, then delete this enrollment from
		// the list of enrollments that have sets.
		if !found {
			if err = b.Delete(ctx, join(keyPfxEnr, enrollmentID)); err != nil {
				return err
			}
		}

		return nil
	})
	return
}

// RetrieveEnrollmentIDs retrieves MDM enrollment IDs from storage.
// In the case of sets and declarations the transitive associations
// are traversed to try and collect the IDs. When multiple slices
// are given they should be treated like a logical or (i.e. finding
// all enrollment IDs for any of the given slices).
// Warning: the results may be very large for e.g. sets (or, transitively,
// declarations) that are assigned to many enrollment IDs.
func (s *KV) RetrieveEnrollmentIDs(ctx context.Context, declarations []string, sets []string, ids []string) ([]string, error) {
	lookupSets := sets
	for _, declarationID := range declarations {
		declarationSets, err := getDeclarationSets(ctx, s.sets, declarationID)
		if err != nil {
			return nil, err
		}
		lookupSets = append(lookupSets, declarationSets...)
	}
	for _, setName := range lookupSets {
		declarationIDs, err := getSetEnrollments(ctx, s.enrollments, setName)
		if err != nil {
			return nil, err
		}
		ids = append(ids, declarationIDs...)
	}
	return ids, nil
}
