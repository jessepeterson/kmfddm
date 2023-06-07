package file

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
)

// RetrieveSetDeclarations returns a slice of declaration IDs that are associated with setName.
func (s *File) RetrieveSetDeclarations(_ context.Context, setName string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return getSlice(s.setFilename(setName))
}

// StoreSetDeclaration creates the association between a declaration and a set.
func (s *File) StoreSetDeclaration(_ context.Context, setName, declarationID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// set the forward reference
	changed, err := setOrRemoveIn(s.setFilename(setName), declarationID, true)
	if err != nil {
		return false, fmt.Errorf("setting declaration in set file: %w", err)
	}
	if changed {
		// update the back-reference
		_, err = setOrRemoveIn(s.declarationSetsFilename(declarationID), setName, true)
		if err != nil {
			return false, fmt.Errorf("setting set in declaration file: %w", err)
		}

		// update (all of) the enrollment ID DDM files
		if err = s.writeSetDDM(setName); err != nil {
			return false, fmt.Errorf("writing set DDM: %w", err)
		}
	}
	return changed, nil
}

// RemoveSetDeclaration removes the association between a declaration and a set.
func (s *File) RemoveSetDeclaration(_ context.Context, setName, declarationID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// set the forward reference
	changed, err := setOrRemoveIn(s.setFilename(setName), declarationID, false)
	if err != nil {
		return false, fmt.Errorf("removing declaration in set file: %w", err)
	}
	if changed {
		// update the back-reference
		_, err = setOrRemoveIn(s.declarationSetsFilename(declarationID), setName, false)
		if err != nil {
			return false, fmt.Errorf("removing set in declaration file: %w", err)
		}

		// update (all of) the enrollment ID DDM files
		if err = s.writeSetDDM(setName); err != nil {
			return false, fmt.Errorf("writing set DDM: %w", err)
		}
	}
	return changed, nil
}

// RetrieveSets retrieves the list of all sets.
func (s *File) RetrieveSets(_ context.Context) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pathPrefix := path.Join(s.path, prefixSet)
	matches, err := filepath.Glob(pathPrefix + "*" + suffixTXT)
	if err != nil {
		return nil, fmt.Errorf("getting set file list: %w", err)
	}
	truncated := make([]string, len(matches))
	for i, match := range matches {
		truncated[i] = match[len(pathPrefix) : len(match)-len(suffixTXT)]
	}
	return truncated, nil
}

// RetrieveDeclarationSets returns the list of sets associated with a declaration.
func (s *File) RetrieveDeclarationSets(_ context.Context, declarationID string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return getSlice(s.declarationSetsFilename(declarationID))
}
