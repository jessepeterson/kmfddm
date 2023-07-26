// Package file is a filesystem-based storage backend for KMFDDM.
package file

import (
	"errors"
	"fmt"
	"hash"
	"os"
	"path"
	"strings"
	"sync"
)

// File is a filesystem-based storage backend.
type File struct {
	mu      sync.RWMutex
	path    string
	newHash func() hash.Hash
}

// New creates and initializes a new filesystem-based storage backend.
func New(path string, newHash func() hash.Hash) (*File, error) {
	if newHash == nil {
		panic("newHash must not be nil")
	}
	if err := os.Mkdir(path, 0755); err != nil && !errors.Is(err, os.ErrExist) {
		return nil, err
	}
	return &File{
		path:    path,
		newHash: newHash,
	}, nil
}

const (
	prefixDeclararion    = "declaration."
	suffixJSON           = ".json"
	suffixTXT            = ".txt"
	prefixSet            = "set.declarations."
	prefixSetEnrollments = "set.enrollments."

	declarationItemsFilename = "declaration-items.json"
	tokensFilename           = "tokens.json"
)

// setFilename returns the path to the set-to-declaration mapping text file.
func (s *File) setFilename(setName string) string {
	return path.Join(s.path, prefixSet+setName+suffixTXT)
}

// declarationSetsFilename returns the path to the declaration-to-set mapping text file.
func (s *File) declarationSetsFilename(declarationID string) string {
	return path.Join(s.path, prefixDeclararion+declarationID+".sets.txt")
}

// declarationFilename returns the path to the full declaration json.
func (s *File) declarationFilename(identifier string) string {
	return path.Join(s.path, relativeDeclarationFilename(identifier))
}

// relativeDeclarationFilename returns the in-storage directory path to the full declaration json.
func relativeDeclarationFilename(identifier string) string {
	return prefixDeclararion + identifier + ".json"
}

// declarationTokenFilename returns the path to the declaration server token.
// This is a "cache" of the computed ServerToken in the declaration.
func (s *File) declarationTokenFilename(identifier string) string {
	return path.Join(s.path, prefixDeclararion+identifier+".token.txt")
}

// declarationSaltFilename returns the path to declaration "salt."
// This "salt" is created when a declaration is created to try and
// generate a different ServerToken each time a declaration is uploaded.
func (s *File) declarationSaltFilename(identifier string) string {
	return path.Join(s.path, prefixDeclararion+identifier+".salt.dat")
}

// enrollmentSetsFilename returns the path to the enrollment ID-to-set mapping file.
// Note it is contained within the enrollment ID directory.
func (s *File) enrollmentSetsFilename(enrollmentID string) string {
	return path.Join(s.path, enrollmentID, "sets.txt")
}

// setEnrollmentsFilename returns the path to the set-to-enrollment ID mapping file.
func (s *File) setEnrollmentsFilename(setName string) string {
	return path.Join(s.path, prefixSetEnrollments+setName+suffixTXT)
}

// declarationItemsFilename returns the path to the enrollment's declaration-items JSON file.
func (s *File) declarationItemsFilename(enrollmentID string) string {
	return path.Join(s.path, enrollmentID, declarationItemsFilename)
}

// tokensFilename returns the path to the enrollment's token JSON file.
func (s *File) tokensFilename(enrollmentID string) string {
	return path.Join(s.path, enrollmentID, tokensFilename)
}

// enrollmentDeclarationFilename returns the path to the enrollment's declaration symlink file.
func (s *File) enrollmentDeclarationFilename(declarationID, declarationType, enrollmentID string) string {
	return path.Join(
		s.path,
		enrollmentID,
		fmt.Sprintf("declaration.%s.%s.json", declarationType, declarationID),
	)
}

// getSlice returns a string slice of the contains of filename split by newlines.
// An empty slice is returned if filename does not exist or contains only whitespace.
func getSlice(filename string) ([]string, error) {
	b, err := os.ReadFile(filename)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(string(b))
	if trimmed == "" {
		return nil, nil
	}
	return strings.Split(trimmed, "\n"), nil
}

// putSlice writes a string slice joined by newlines to filename.
func putSlice(filename string, slice []string) error {
	return os.WriteFile(filename, []byte(strings.Join(slice, "\n")), 0644)
}

// contains returns the position of elemn within the slice elems or -1.
func contains(elems []string, elem string) int {
	for i, v := range elems {
		if v == elem {
			return i
		}
	}
	return -1
}

// assureIn returns a new slice that makes sure elem is in elems.
// The first return value is whether we the slice was changed or not.
func assureIn(elems []string, elem string) (bool, []string) {
	if contains(elems, elem) >= 0 {
		// already in slice
		return false, elems
	}
	out := append([]string{}, elems...) // new slice
	out = append(out, elem)
	return true, out
}

// assureOut returns a new slice that makes sure elem is not in elems.
// The first return value is whether we the slice was changed or not.
func assureOut(elems []string, elem string) (bool, []string) {
	pos := contains(elems, elem)
	if pos < 0 {
		// not in slice already
		return false, elems
	}
	out := append([]string{}, elems[:pos]...) // new slice
	out = append(out, elems[pos+1:]...)
	return true, out
}

// setOrRemoveIn reads the slice from disk, modifies it, and writes it back out if it was changed.
func setOrRemoveIn(filename, elem string, set bool) (bool, error) {
	elems, err := getSlice(filename)
	if err != nil {
		return false, err
	}
	var changed bool
	if set {
		changed, elems = assureIn(elems, elem)
	} else {
		changed, elems = assureOut(elems, elem)
	}
	if !changed {
		return false, nil
	}
	if !set && len(elems) < 1 {
		return true, os.Remove(filename)
	}
	return true, putSlice(filename, elems)
}

// assureEnrollmentDirExists makes sure the enrollment ID directory exists.
func (s *File) assureEnrollmentDirExists(enrollmentID string) error {
	dirname := path.Join(s.path, enrollmentID)
	err := os.Mkdir(dirname, 0755)
	if err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	return nil
}
