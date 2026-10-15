package pgsql

import (
	"context"
	"errors"
	"time"

	"github.com/jessepeterson/kmfddm/storage/pgsql/sqlc"
)

// RetrieveEnrollmentSets retrieves the list of sets an enrollment is assigned to.
// See also the storage package for documentation on the storage interfaces.
func (s *PgSQLStorage) RetrieveEnrollmentSets(ctx context.Context, enrollmentID string) ([]string, error) {
	return s.q.GetEnrollmentSets(ctx, enrollmentID)
}

// StoreEnrollmentSet creates the association between an enrollment and a set.
// See also the storage package for documentation on the storage interfaces.
func (s *PgSQLStorage) StoreEnrollmentSet(ctx context.Context, enrollmentID, setName string) (bool, error) {
	result, err := s.q.StoreEnrollmentSet(ctx, sqlc.StoreEnrollmentSetParams{
		Now:          time.Now().UnixMilli(),
		EnrollmentID: enrollmentID,
		SetName:      setName,
	})
	if err != nil {
		return false, err
	}
	return resultChangedRows(result)
}

// RemoveEnrollmentSet removes the association between an enrollment and a set.
// See also the storage package for documentation on the storage interfaces.
func (s *PgSQLStorage) RemoveEnrollmentSet(ctx context.Context, enrollmentID, setName string) (bool, error) {
	result, err := s.q.RemoveEnrollmentSet(ctx, sqlc.RemoveEnrollmentSetParams{
		EnrollmentID: enrollmentID,
		SetName:      setName,
	})
	if err != nil {
		return false, err
	}
	return resultChangedRows(result)
}

// RemoveAllEnrollmentSets dissociates enrollment ID from any sets.
// If any associations are removed true is returned.
// It should not be an error if no associations exist.
func (s *PgSQLStorage) RemoveAllEnrollmentSets(ctx context.Context, enrollmentID string) (bool, error) {
	r, err := s.q.RemoveAllEnrollmentSets(ctx, enrollmentID)
	if err != nil {
		return false, err
	}
	return resultChangedRows(r)
}

// RetrieveEnrollmentIDs retrieves enrollment IDs.
// See also the storage package for documentation on the storage interfaces.
func (s *PgSQLStorage) RetrieveEnrollmentIDs(ctx context.Context, declarations []string, sets []string, ids []string) ([]string, error) {
	if len(declarations) < 1 && len(sets) < 1 && len(ids) < 1 {
		return nil, errors.New("no parameters provided")
	}
	found, err := s.q.GetEnrollmentIDs(ctx, sqlc.GetEnrollmentIDsParams{
		Declarations: declarations,
		Sets:         sets,
		Ids:          ids,
	})
	retIDMap := make(map[string]struct{}, len(found)+len(ids))
	for _, id := range found {
		retIDMap[id] = struct{}{}
	}
	// merge in the enrollment IDs directly supplied in params
	for _, id := range ids {
		retIDMap[id] = struct{}{}
	}
	// convert back to slice for return value
	retIDs := make([]string, 0, len(retIDMap))
	for k := range retIDMap {
		retIDs = append(retIDs, k)
	}
	return retIDs, err
}
