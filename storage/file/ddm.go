package file

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/jessepeterson/kmfddm/ddm"
	"github.com/jessepeterson/kmfddm/ddm/build"
	"github.com/jessepeterson/kmfddm/storage"
)

// RetrieveDeclarationItems reads the declarations from disk for enrollmentID.
// First the already-cached declaration items is read from disk then
// each individual declaration is from disk.
func (s *File) RetrieveDeclarationItems(ctx context.Context, enrollmentID string) ([]*ddm.Declaration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dif, err := os.Open(s.declarationItemsFilename(enrollmentID))
	if err != nil {
		return nil, fmt.Errorf("opening declaration items file: %w", err)
	}
	defer dif.Close()

	di := new(ddm.DeclarationItems)
	err = json.NewDecoder(dif).Decode(di)
	if err != nil {
		return nil, fmt.Errorf("decoding declaration items json: %w", err)
	}

	var decls []*ddm.Declaration

	for _, md := range [][]ddm.ManifestDeclaration{
		di.Declarations.Activations,
		di.Declarations.Configurations,
		di.Declarations.Assets,
		di.Declarations.Management,
	} {
		for _, di := range md {
			d, err := s.readDeclarationFile(di.Identifier)
			if err != nil {
				return decls, fmt.Errorf("reading declaration: %s: %w", di.Identifier, err)
			}

			// clear out unused fields. decls will (likely) only be used
			// for generating DI/tokens so actual data is unecessary.
			d.Payload = nil
			d.Raw = nil

			decls = append(decls, d)
		}
	}

	return decls, nil
}

// RetrieveEnrollmentDeclarationJSON retrieves the DDM declaration JSON for an enrollment ID.
func (s *File) RetrieveEnrollmentDeclarationJSON(_ context.Context, declarationID, declarationType, enrollmentID string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, err := os.ReadFile(s.enrollmentDeclarationFilename(declarationID, declarationType, enrollmentID))
	if errors.Is(err, os.ErrNotExist) {
		err = fmt.Errorf("%w: %v", storage.ErrDeclarationNotFound, err)
	}
	return b, err
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
	declarationIDs, err := s.retrieveEnrollmentIDs([]string{declarationID}, nil, nil)
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
	di := build.NewDIBuilder(s.newHash)
	ti := build.NewTokensBuilder(s.newHash)

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
		di.Add(d)
		ti.Add(d)

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
