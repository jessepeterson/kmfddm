package test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jessepeterson/kmfddm/ddm"
	"github.com/jessepeterson/kmfddm/http/api"
	httpddm "github.com/jessepeterson/kmfddm/http/ddm"
	"github.com/jessepeterson/kmfddm/notifier"
)

type myStorage interface {
	api.EnrollmentAPIStorage
	api.DeclarationAPIStorage
	notifier.EnrollmentIDFinder
	httpddm.TokensDeclarationItemsRetriever
}

func testEnrollments(t *testing.T, storage myStorage, ctx context.Context, d *ddm.Declaration, enrollmentID, setName string) {
	// associate
	_, err := storage.StoreEnrollmentSet(ctx, enrollmentID, setName)
	if err != nil {
		t.Fatal(err)
	}

	// find ref
	setNames, err := storage.RetrieveEnrollmentSets(ctx, enrollmentID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, v := range setNames {
		if v == setName {
			found = true
			break
		}
	}
	if !found {
		t.Error("set name not found in enrollment sets (should exist)")
	}

	// find ref
	ids, err := storage.RetrieveSetEnrollmentIDs(ctx, setName)
	if err != nil {
		t.Fatal(err)
	}
	found = false
	for _, id := range ids {
		if id == enrollmentID {
			found = true
			break
		}
	}
	if !found {
		t.Error("did not find ID in set list of IDs")
	}

	ids, err = storage.RetrieveDeclarationEnrollmentIDs(ctx, d.Identifier)
	if err != nil {
		t.Fatal(err)
	}
	found = false
	for _, id := range ids {
		if id == enrollmentID {
			found = true
			break
		}
	}
	if !found {
		t.Error("did not find ID in transitive list")
	}

	b, err := storage.RetrieveDeclarationItemsJSON(ctx, enrollmentID)
	if err != nil {
		t.Fatal(err)
	}

	i := new(ddm.DeclarationItems)
	err = json.Unmarshal(b, i)
	if err != nil {
		t.Fatal(err)
	}

	if len(i.Declarations.Configurations) < 1 {
		t.Error("invalid number of configurations")
	} else {
		if have, want := i.Declarations.Configurations[0].Identifier, d.Identifier; have != want {
			t.Errorf("identifier: have: %v, want: %v", have, want)
		}

		// re-retrieve declaration to make sure we have the storage-written token
		d2, err := storage.RetrieveDeclaration(ctx, d.Identifier)
		if err != nil {
			t.Fatal(err)
		}

		if have, want := i.Declarations.Configurations[0].ServerToken, d2.ServerToken; have != want {
			t.Errorf("token: have: %v, want: %v", have, want)
		}
	}

	// dissociate
	_, err = storage.RemoveEnrollmentSet(ctx, enrollmentID, setName)
	if err != nil {
		t.Fatal(err)
	}

	// verify no ref
	setNames, err = storage.RetrieveEnrollmentSets(ctx, enrollmentID)
	if err != nil {
		t.Fatal(err)
	}
	found = false
	for _, v := range setNames {
		if v == setName {
			found = true
			break
		}
	}
	if found {
		t.Error("set name found in enrollment sets (should not exist)")
	}
}
