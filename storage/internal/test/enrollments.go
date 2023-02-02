package test

import (
	"context"
	"testing"

	"github.com/jessepeterson/kmfddm/http/api"
)

func testEnrollments(t *testing.T, storage api.EnrollmentAPIStorage, ctx context.Context, enrollmentID, setName string) {
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
