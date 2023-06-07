package file

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/jessepeterson/kmfddm/ddm"
)

// RetrieveEnrollmentDeclarationJSON retrieves the DDM declaration JSON for an enrollment ID.
func (s *File) RetrieveEnrollmentDeclarationJSON(_ context.Context, declarationID, declarationType, enrollmentID string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return os.ReadFile(s.enrollmentDeclarationFilename(declarationID, declarationType, enrollmentID))
}

// RetrieveDeclarationItemsJSON retrieves the DDM declaration-items JSON for an enrollment ID.
func (s *File) RetrieveDeclarationItemsJSON(_ context.Context, enrollmentID string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return os.ReadFile(s.declarationItemsFilename(enrollmentID))
}

// RetrieveDeclarationItemsJSON retrieves the DDM token JSON for an enrollment ID.
func (s *File) RetrieveTokensJSON(_ context.Context, enrollmentID string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return os.ReadFile(path.Join(s.path, enrollmentID, tokensFilename))
}

// writeDeclarationDDM looks up the enrollments associated with a declaration and writes the DDM files for each.
func (s *File) writeDeclarationDDM(declarationID string) error {
	// first find all enrollment IDs mapped to this declaration.
	declarationIDs, err := s.declarationEnrollmentIDs(declarationID)
	if err != nil {
		return err
	}
	for _, id := range declarationIDs {
		// write the enrollment DDM files
		if err = s.writeEnrollmentDDM(id); err != nil {
			return err
		}
	}
	return nil
}

// writeSetDDM writes the DDM files for all enrollments belonging to a set.
func (s *File) writeSetDDM(setName string) error {
	// get all the enrollment ids for a this set
	setEnrIDs, err := getSlice(s.setEnrollmentsFilename(setName))
	if err != nil {
		return err
	}
	for _, setEnrID := range setEnrIDs {
		// write the enrollment DDM files
		if err = s.writeEnrollmentDDM(setEnrID); err != nil {
			return err
		}
	}
	return nil
}

// writeEnrollmentDDM generates all enrollment ID-specific DDM declarations.
func (s *File) writeEnrollmentDDM(enrollmentID string) error {
	// get all the sets this id is enrolled in
	enrollmentSets, err := getSlice(s.enrollmentSetsFilename(enrollmentID))
	if err != nil {
		return fmt.Errorf("getting sets for enrollment: %w", err)
	}

	enrollmentDeclarations := make(map[string]struct{})
	for _, setName := range enrollmentSets {
		// get all the declarations for this set
		setDeclarations, err := getSlice(s.setFilename(setName))
		if err != nil {
			return fmt.Errorf("getting declarations from set for %s: %w", setName, err)
		}
		for _, declarationID := range setDeclarations {
			// collect declaration IDs in our map
			enrollmentDeclarations[declarationID] = struct{}{}
		}
	}

	if err = s.assureEnrollmentDirExists(enrollmentID); err != nil {
		return fmt.Errorf("assuring enrollment directory exists: %w", err)
	}

	// create our token and declaration-items builders
	di := ddm.NewDIBuilder(s.newHash)
	ti := ddm.NewTokensBuilder(s.newHash)

	// find any existing declaration symlinks
	matches, err := filepath.Glob(path.Join(s.path, enrollmentID, "declaration.*.json"))
	if err != nil {
		return fmt.Errorf("finding declaration symlinks: %w", err)
	}

	for declarationID := range enrollmentDeclarations {
		// read and parse declaration
		dBytes, err := os.ReadFile(s.declarationFilename(declarationID))
		if err != nil {
			return fmt.Errorf("reading declaration: %w", err)
		}
		d, err := ddm.ParseDeclaration(dBytes)
		if err != nil {
			return fmt.Errorf("parsing declaration: %w", err)
		}

		// add to our DI and tokens builders
		di.AddDeclarationData(d)
		ti.AddDeclarationData(d)

		// create declaration symlink if not exists
		symlinkName := s.enrollmentDeclarationFilename(d.Identifier, ddm.ManifestType(d.Type), enrollmentID)
		if pos := contains(matches, symlinkName); pos >= 0 {
			matches = append(matches[:pos], matches[pos+1:]...)
		} else {
			err = os.Symlink(
				path.Join("..", relativeDeclarationFilename(d.Identifier)),
				symlinkName,
			)
			if err != nil {
				return fmt.Errorf("creating declaration symlink: %w", err)
			}
		}
	}

	// remove any symlinks for previous declarations
	for _, oldSymlink := range matches {
		if err = os.Remove(oldSymlink); err != nil {
			return fmt.Errorf("removing declaration symlink: %w", err)
		}
	}

	// finalize the builders
	di.Finalize()
	ti.Finalize()

	// marshal and write the declarations-items JSON
	diJSON, err := json.Marshal(&di.DeclarationItems)
	if err != nil {
		return err
	}
	if err = os.WriteFile(s.declarationItemsFilename(enrollmentID), diJSON, 0644); err != nil {
		return err
	}

	// marshal and write the tokens JSON
	tiJSON, err := json.Marshal(&ti.TokensResponse)
	if err != nil {
		return err
	}
	if err = os.WriteFile(s.tokensFilename(enrollmentID), tiJSON, 0644); err != nil {
		return err
	}

	return nil
}
