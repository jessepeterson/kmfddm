package test

import (
	"context"
	"encoding/json"
	"errors"
	"hash"
	"testing"

	"github.com/cespare/xxhash"
	"github.com/jessepeterson/kmfddm/ddm"
	"github.com/jessepeterson/kmfddm/ddm/build"
	"github.com/jessepeterson/kmfddm/storage"
)

type myStorage interface {
	storage.TokensDeclarationItemsRetriever
	storage.EnrollmentDeclarationDataStorage
	storage.EnrollmentIDRetriever
	storage.DeclarationAPIStorage
	storage.EnrollmentSetStorage
}

func testEnrollments(t *testing.T, store myStorage, ctx context.Context, d *ddm.Declaration, enrollmentID, setName string) {
	// associate
	_, err := store.StoreEnrollmentSet(ctx, enrollmentID, setName)
	if err != nil {
		t.Fatal(err)
	}

	// find ref
	setNames, err := store.RetrieveEnrollmentSets(ctx, enrollmentID)
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
	ids, err := store.RetrieveEnrollmentIDs(ctx, nil, []string{setName}, []string{enrollmentID})
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

	ids, err = store.RetrieveEnrollmentIDs(ctx, []string{d.Identifier}, nil, []string{enrollmentID})
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

	declarations, err := store.RetrieveDeclarationItems(ctx, enrollmentID)
	if err != nil {
		t.Fatal(err)
	}

	build := build.NewDIBuilder(func() hash.Hash { return xxhash.New() })
	for _, d := range declarations {
		build.Add(d)
	}
	build.Finalize()

	i := &build.DeclarationItems

	if len(i.Declarations.Configurations) < 1 {
		t.Error("invalid number of configurations")
	} else {
		if have, want := i.Declarations.Configurations[0].Identifier, d.Identifier; have != want {
			t.Errorf("identifier: have: %v, want: %v", have, want)
		}

		// re-retrieve declaration to make sure we have the storage-written token
		d2, err := store.RetrieveDeclaration(ctx, d.Identifier)
		if err != nil {
			t.Fatal(err)
		}

		if have, want := i.Declarations.Configurations[0].ServerToken, d2.ServerToken; have != want {
			t.Errorf("token: have: %v, want: %v", have, want)
		}
	}

	// same test, but parsing the actual JSON

	b, err := store.RetrieveDeclarationItemsJSON(ctx, enrollmentID)
	if err != nil {
		t.Fatal(err)
	}

	i = new(ddm.DeclarationItems)
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
		d2, err := store.RetrieveDeclaration(ctx, d.Identifier)
		if err != nil {
			t.Fatal(err)
		}

		if have, want := i.Declarations.Configurations[0].ServerToken, d2.ServerToken; have != want {
			t.Errorf("token: have: %v, want: %v", have, want)
		}
	}

	// dissociate
	changed, err := store.RemoveEnrollmentSet(ctx, enrollmentID, setName)
	if err != nil {
		t.Fatal(err)
	}

	if have, want := changed, true; have != want {
		t.Errorf("changed: have: %v, want: %v", have, want)
	}

	// verify no ref
	setNames, err = store.RetrieveEnrollmentSets(ctx, enrollmentID)
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

	// verify error return type for missing declaration
	_, err = store.RetrieveEnrollmentDeclarationJSON(ctx, "should.be.missing.a", "asset", "should.be.missing.b")
	if !errors.Is(err, storage.ErrDeclarationNotFound) {
		t.Error("error should be ErrDeclarationNotFound")
	}

	// associate again
	changed, err = store.StoreEnrollmentSet(ctx, enrollmentID, setName)
	if err != nil {
		t.Fatal(err)
	}

	if have, want := changed, true; have != want {
		t.Errorf("changed: have: %v, want: %v", have, want)
	}

	// verify 1 ref
	setNames, err = store.RetrieveEnrollmentSets(ctx, enrollmentID)
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(setNames), 1; have != want {
		t.Errorf("len(setName) have: %v, want: %v", have, want)
	} else {
		if have, want := setNames[0], setName; have != want {
			t.Errorf("setNames[0] have: %v, want: %v", have, want)
		}
	}

	// remove all sets associated with enrollmentID
	changed, err = store.RemoveAllEnrollmentSets(ctx, enrollmentID)
	if err != nil {
		t.Fatal(err)
	}

	if have, want := changed, true; have != want {
		t.Errorf("changed: have: %v, want: %v", have, want)
	}

	// verify no ref
	setNames, err = store.RetrieveEnrollmentSets(ctx, enrollmentID)
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(setNames), 0; have != want {
		t.Errorf("len(setName) have: %v, want: %v", have, want)
	}

	// remove all sets associated with enrollmentID again (to check changed status)
	changed, err = store.RemoveAllEnrollmentSets(ctx, enrollmentID)
	if err != nil {
		t.Fatal(err)
	}

	if have, want := changed, false; have != want {
		t.Errorf("changed: have: %v, want: %v", have, want)
	}
}
