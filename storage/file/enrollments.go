package file

import (
	"context"
	"fmt"
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

// declarationEnrollmentIDs finds all the enrollment IDs that are associated with a declaration.
func (s *File) declarationEnrollmentIDs(declarationID string) ([]string, error) {
	// find all sets for a declaration
	declSets, err := getSlice(s.declarationSetsFilename(declarationID))
	if err != nil {
		return nil, fmt.Errorf("getting sets for declaration %s: %w", declarationID, err)
	}
	ids := make(map[string]struct{})
	for _, declSet := range declSets {
		// find all ids associated with these sets
		setIDs, err := getSlice(s.setEnrollmentsFilename(declSet))
		if err != nil {
			return nil, fmt.Errorf("getting enrollments for set %s: %w", declSet, err)
		}
		for _, id := range setIDs {
			ids[id] = struct{}{}
		}
	}
	idSlice := make([]string, 0, len(ids))
	for id := range ids {
		idSlice = append(idSlice, id)
	}
	return idSlice, nil
}

// RetrieveDeclarationEnrollmentIDs retrieves a list of enrollment IDs that are transitively associated with a declaration.
func (s *File) RetrieveDeclarationEnrollmentIDs(_ context.Context, declarationID string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.declarationEnrollmentIDs(declarationID)
}

// RetrieveSetEnrollmentIDs retrieves a list of enrollment IDs that are associated with a set.
func (s *File) RetrieveSetEnrollmentIDs(_ context.Context, setName string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return getSlice(s.setEnrollmentsFilename(setName))
}
