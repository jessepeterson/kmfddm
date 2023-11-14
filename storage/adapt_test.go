package storage

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"testing"

	"github.com/jessepeterson/kmfddm/ddm"
)

type MockStore struct {
	declaration      *ddm.Declaration
	declarationBytes []byte
	enrollmentID     string
}

func NewMockStore(declarationBytes []byte, declarationServerToken string, forEnrollmentID string) (*MockStore, error) {
	declaration, err := ddm.ParseDeclaration(declarationBytes)
	if err != nil {
		return nil, err
	}
	declaration.ServerToken = declarationServerToken
	return &MockStore{
		declaration:      declaration,
		declarationBytes: declarationBytes,
		enrollmentID:     forEnrollmentID,
	}, nil
}

func (s *MockStore) RetrieveDeclarationItems(ctx context.Context, enrollmentID string) ([]*ddm.Declaration, error) {
	if enrollmentID != s.enrollmentID {
		return nil, errors.New("invalid enrollment ID")
	}
	return []*ddm.Declaration{s.declaration}, nil
}

func (s *MockStore) RetrieveEnrollmentDeclarationJSON(ctx context.Context, declarationID, declarationType, enrollmentID string) ([]byte, error) {
	if enrollmentID != s.enrollmentID {
		return nil, fmt.Errorf("%w: invalid enrollment ID", ErrDeclarationNotFound)
	}
	if declarationID != s.declaration.Identifier {
		return nil, fmt.Errorf("%w: invalid declaration ID", ErrDeclarationNotFound)
	}
	if declarationType != ddm.ManifestType(s.declaration.Type) {
		return nil, fmt.Errorf("%w: invalid declaration type", ErrDeclarationNotFound)
	}
	return s.declarationBytes, nil
}

const testDecl = `{
    "Type": "com.apple.configuration.management.test",
    "Payload": {
        "Echo": "Foo"
    },
    "Identifier": "test_golang_bd23307c-4704-4a31-9b8f-a5e8406d8633"
}`

func TestJSONAdapt(t *testing.T) {
	enrollmentID := "ABC-123"
	declSrvTok := "DEF-456"

	// create new mock storage
	s, err := NewMockStore([]byte(testDecl), declSrvTok, enrollmentID)
	if err != nil {
		t.Fatal(err)
	}

	// wrap it in the adapter
	a := NewJSONAdapt(s, func() hash.Hash { return sha256.New() })

	ctx := context.Background()

	// get the single declaration from the adapter
	declBytes, err := a.RetrieveEnrollmentDeclarationJSON(ctx, "test_golang_bd23307c-4704-4a31-9b8f-a5e8406d8633", "configuration", enrollmentID)
	if err != nil {
		t.Fatal(err)
	}
	if have, want := string(declBytes), testDecl; have != want {
		t.Errorf("declarations not equal: have: %v, want: %v", have, want)
	}

	// get the declaration items from the adapter
	diBytes, err := a.RetrieveDeclarationItemsJSON(ctx, enrollmentID)
	if err != nil {
		t.Fatal(err)
	}

	di := new(ddm.DeclarationItems)

	// unmarshal the JSON
	err = json.Unmarshal(diBytes, di)
	if err != nil {
		t.Fatal(err)
	}

	// check that we have our config declaration in the declaration items
	if have, want := len(di.Declarations.Configurations), 1; have != want {
		t.Errorf("wrong number of declarations (configurations): have: %v, want: %v", have, want)
	} else {
		if have, want := s.declaration.Identifier, di.Declarations.Configurations[0].Identifier; have != want {
			t.Errorf("wrong declaration: have: %v, want: %v", have, want)
		}
		if have, want := s.declaration.ServerToken, di.Declarations.Configurations[0].ServerToken; have != want {
			t.Errorf("wrong server token: have: %v, want: %v", have, want)
		}
	}
}
