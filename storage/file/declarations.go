package file

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/jessepeterson/kmfddm/ddm"
	"github.com/jessepeterson/kmfddm/storage"
)

func newSalt() (salt []byte, err error) {
	salt = make([]byte, 32)
	_, err = rand.Read(salt)
	return
}

// StoreDeclaration stores a declaration on disk.
// See also the storage package for documentation on the storage interfaces.
func (s *File) StoreDeclaration(_ context.Context, d *ddm.Declaration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.writeDeclarationFiles(d, false)
}

func (s *File) writeDeclarationFiles(d *ddm.Declaration, forceNewSalt bool) (bool, error) {
	var err error
	var token string
	var creationSalt []byte
	var tokenMissing bool

	tokenFilename := s.declarationTokenFilename(d.Identifier)
	saltFilename := s.declarationSaltFilename(d.Identifier)

	// try to read token and cache our existing token if it exists
	if tokenBytes, err := os.ReadFile(tokenFilename); errors.Is(err, os.ErrNotExist) {
		tokenMissing = true
	} else if err != nil {
		return false, fmt.Errorf("reading server token: %w", err)
	} else {
		// found the token, lets convert it (and read our salt)
		token = string(tokenBytes)
		if !forceNewSalt {
			if creationSalt, err = os.ReadFile(saltFilename); err != nil {
				return false, fmt.Errorf("reading creation salt: %w", err)
			}
		}
	}

	if tokenMissing || forceNewSalt {
		if creationSalt, err = newSalt(); err != nil {
			return false, fmt.Errorf("creating new salt: %w", err)
		}
	}

	// unmarshal the raw declaration
	var declaration ddm.Declaration
	if err = json.Unmarshal(d.Raw, &declaration); err != nil {
		return false, err
	}

	// remove the servertoken to make the marshaling idempotent
	declaration.ServerToken = ""

	// re-marshal (without a servertoken)
	dBytes, err := json.Marshal(&declaration)
	if err != nil {
		return false, fmt.Errorf("marshaling no-token declaration: %w", err)
	}

	// hash the marshaled declaration (again without token but with creation salt)
	hasher := s.newHash()
	_, err = hasher.Write(append(dBytes, creationSalt...))
	if err != nil {
		return false, fmt.Errorf("hashing marshaled no-token declaration: %w", err)
	}
	dHash := fmt.Sprintf("%x", hasher.Sum(nil))

	if !tokenMissing && dHash == token {
		// the hashed version of our profile is the same
		// as the token we already have. bail telling the caller there
		// were no changes.
		return false, nil
	}
	token = dHash

	declaration.ServerToken = token

	// marshal the declaration (with the new token)
	dBytes, err = json.Marshal(&declaration)
	if err != nil {
		return false, fmt.Errorf("marshaling declaration: %w", err)
	}

	if err = os.WriteFile(s.declarationFilename(d.Identifier), dBytes, 0644); err != nil {
		return false, fmt.Errorf("writing declaration: %w", err)
	}

	if err = os.WriteFile(tokenFilename, []byte(token), 0644); err != nil {
		return false, fmt.Errorf("writing declaration token: %w", err)
	}

	if tokenMissing || forceNewSalt {
		// we only want to change the salt if we're either touching or
		// making a "new" declaration.
		if err = os.WriteFile(saltFilename, creationSalt, 0644); err != nil {
			return false, fmt.Errorf("writing creation salt: %w", err)
		}
	}

	// finally, write all the DDM files for this declaration
	if err = s.writeDeclarationDDM(d.Identifier); err != nil {
		return false, err
	}

	return true, nil
}

// RetrieveDeclaration retrieves a declaration by its ID.
// See also the storage package for documentation on the storage interfaces.
func (s *File) RetrieveDeclaration(_ context.Context, declarationID string) (*ddm.Declaration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.readDeclarationFile(declarationID)
}

func (s *File) readDeclarationFile(declarationID string) (*ddm.Declaration, error) {
	dBytes, err := os.ReadFile(s.declarationFilename(declarationID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err = fmt.Errorf("%w: %v", storage.ErrDeclarationNotFound, err)
		}
		return nil, fmt.Errorf("reading declaration: %w", err)
	}
	d, err := ddm.ParseDeclaration(dBytes)
	if err != nil {
		return nil, fmt.Errorf("parsing declaration: %w", err)
	}
	return d, nil
}

// RetrieveDeclarationModTime retrieves the last modification time of the declaration.
// See also the storage package for documentation on the storage interfaces.
func (s *File) RetrieveDeclarationModTime(ctx context.Context, declarationID string) (time.Time, error) {
	fi, err := os.Stat(s.declarationFilename(declarationID))
	if errors.Is(err, os.ErrNotExist) {
		err = fmt.Errorf("%w: %v", storage.ErrDeclarationNotFound, err)
	}
	return fi.ModTime(), err
}

// DeleteDeclaration deletes a declaration by its ID.
// See also the storage package for documentation on the storage interfaces.
func (s *File) DeleteDeclaration(_ context.Context, identifier string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// fetch all sets this declaration belongs to
	sets, err := getSlice(s.declarationSetsFilename(identifier))
	if err != nil {
		return false, fmt.Errorf("getting sets from declaration: %w", err)
	}
	if len(sets) > 0 {
		// try to maintain some semblance of referential integrity by
		// not preventing deletion if we're with sets.
		return false, fmt.Errorf("declaration %s contained in %d set(s)", identifier, len(sets))
	}
	rmFiles := []string{
		s.declarationFilename(identifier),
		s.declarationTokenFilename(identifier),
		s.declarationSaltFilename(identifier),
		s.declarationSetsFilename(identifier),
	}
	changed := false
	for _, rm := range rmFiles {
		if err := os.Remove(rm); err == nil {
			changed = true
		} else if !errors.Is(err, os.ErrNotExist) {
			return false, fmt.Errorf("deleting declaration file: %w", err)
		}
	}
	return changed, nil
}

// RetrieveDeclarations retrieves a slice of all declaration IDs.
// See also the storage package for documentation on the storage interfaces.
func (s *File) RetrieveDeclarations(_ context.Context) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pathPrefix := path.Join(s.path, prefixDeclararion)
	matches, err := filepath.Glob(pathPrefix + "*" + suffixJSON)
	if err != nil {
		return nil, fmt.Errorf("getting declaration file list: %w", err)
	}
	truncated := make([]string, len(matches))
	for i, match := range matches {
		truncated[i] = match[len(pathPrefix) : len(match)-len(suffixJSON)]
	}
	return truncated, nil
}

// TouchDeclaration rewrites a declaration with a new ServerToken.
// See also the storage package for documentation on the storage interfaces.
func (s *File) TouchDeclaration(ctx context.Context, declarationID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	d, err := s.readDeclarationFile(declarationID)
	if err != nil {
		return err
	}
	_, err = s.writeDeclarationFiles(d, true)
	return err
}
