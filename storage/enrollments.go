package storage

import "context"

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

type EnrollmentIDRetriever interface {
	// RetrieveEnrollmentIDs retrieves MDM enrollment IDs from storage.
	//
	// In the case of sets and declarations the transitive associations
	// are traversed to try and collect the IDs. When multiple slices
	// are given they should be treated like a logical or (i.e. finding
	// all enrollment IDs for any of the given slices).
	//
	// Warning: the results may be very large for e.g. sets (or, transitively,
	// declarations) that are assigned to many enrollment IDs.
	RetrieveEnrollmentIDs(ctx context.Context, declarations []string, sets []string, ids []string) ([]string, error)
}
