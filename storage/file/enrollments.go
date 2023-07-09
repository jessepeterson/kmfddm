package file

import (
	"context"
	"errors"
	"fmt"
	"os"
)

// RetrieveEnrollmentSets returns the slice of sets associated with an enrollment ID.
func (s *File) RetrieveEnrollmentSets(_ context.Context, enrollmentID string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return getSlice(s.enrollmentSetsFilename(enrollmentID))
}

// StoreEnrollmentSet creates the association between an enrollment and a set.
func (s *File) StoreEnrollmentSet(_ context.Context, enrollmentID, setName string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := s.assureEnrollmentDirExists(enrollmentID)
	if err != nil {
		return false, fmt.Errorf("assuring enrollment directory exists: %w", err)
	}
	// set the forward reference
	changed, err := setOrRemoveIn(s.enrollmentSetsFilename(enrollmentID), setName, true)
	if err != nil {
		return false, fmt.Errorf("setting set in enrollment file: %w", err)
	}
	if changed {
		// update the back-reference
		_, err = setOrRemoveIn(s.setEnrollmentsFilename(setName), enrollmentID, true)
		if err != nil {
			return false, fmt.Errorf("setting enrollment in set file: %w", err)
		}

		// update (all of) the enrollment ID DDM files
		err = s.writeEnrollmentDDM(enrollmentID)
		if err != nil {
			return false, fmt.Errorf("writing enrollment DDM: %w", err)
		}
	}
	return changed, nil
}

// RemoveEnrollmentSet removes the association between an enrollment and a set.
func (s *File) RemoveEnrollmentSet(_ context.Context, enrollmentID, setName string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := s.assureEnrollmentDirExists(enrollmentID)
	if err != nil {
		return false, fmt.Errorf("assuring enrollment directory exists: %w", err)
	}
	// set the forward reference
	changed, err := setOrRemoveIn(s.enrollmentSetsFilename(enrollmentID), setName, false)
	if err != nil {
		return false, fmt.Errorf("removing set in enrollment file: %w", err)
	}
	if changed {
		// update the back-reference
		_, err = setOrRemoveIn(s.setEnrollmentsFilename(setName), enrollmentID, false)
		if err != nil {
			return false, fmt.Errorf("removing enrollment in set file: %w", err)
		}

		// update (all of) the enrollment ID DDM files
		err = s.writeEnrollmentDDM(enrollmentID)
		if err != nil {
			return false, fmt.Errorf("writing enrollment DDM: %w", err)
		}
	}
	return changed, nil
}

// RetrieveEnrollmentIDs retrieves MDM enrollment IDs from storage.
// If a set, declaration, or enrollment ID doesn't exist it is ignored.
func (s *File) RetrieveEnrollmentIDs(_ context.Context, declarations []string, sets []string, ids []string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.retrieveEnrollmentIDs(declarations, sets, ids)
}

func (s *File) retrieveEnrollmentIDs(declarations []string, sets []string, ids []string) ([]string, error) {
	retIDs := make(map[string]struct{})
	setMemoize := make(map[string]struct{})

	for _, declarationID := range declarations {
		setNames, err := getSlice(s.declarationSetsFilename(declarationID))
		if err != nil {
			return nil, fmt.Errorf("getting sets for declaration %s: %w", declarationID, err)
		}
		for _, setName := range setNames {
			if _, ok := setMemoize[setName]; ok {
				continue
			}
			// find all ids associated with these sets
			setIDs, err := getSlice(s.setEnrollmentsFilename(setName))
			if err != nil {
				return nil, fmt.Errorf("getting enrollments for set %s: %w", setName, err)
			}
			for _, id := range setIDs {
				retIDs[id] = struct{}{}
			}
			setMemoize[setName] = struct{}{}
		}
	}

	for _, setName := range sets {
		if _, ok := setMemoize[setName]; ok {
			continue
		}
		// find all ids associated with these sets
		setIDs, err := getSlice(s.setEnrollmentsFilename(setName))
		if err != nil {
			return nil, fmt.Errorf("getting enrollments for set %s: %w", setName, err)
		}
		for _, id := range setIDs {
			retIDs[id] = struct{}{}
		}
		setMemoize[setName] = struct{}{}
	}

	// retrieve any enrollment IDs (if they're valid)
	for _, id := range ids {
		if _, ok := retIDs[id]; ok {
			continue
		}
		// check if we've previously seen this id before
		// (by checking if we've written a tokens file for it before)
		_, err := os.Stat(s.tokensFilename(id))
		if err == nil {
			retIDs[id] = struct{}{}
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("getting enrollments sets for id %s: %w", id, err)
		}
	}

	// s.enrollmentDeclarationFilename()

	retIDSlice := make([]string, 0, len(retIDs))
	for k := range retIDs {
		retIDSlice = append(retIDSlice, k)
	}
	return retIDSlice, nil
}
