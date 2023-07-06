package test

import (
	"context"
	"testing"

	"github.com/jessepeterson/kmfddm/ddm"
	"github.com/jessepeterson/kmfddm/http/api"
	"github.com/jessepeterson/kmfddm/storage"
)

const testDecl = `{
    "Type": "com.apple.configuration.management.test",
    "Payload": {
        "Echo": "Foo"
    },
    "Identifier": "test_golang_9e6a3aa7-5e4b-4d38-aacf-0f8058b2a899"
}`

type allTestStorage interface {
	setAndDeclStorage
	api.EnrollmentAPIStorage
	storage.TokensDeclarationItemsRetriever
	storage.Toucher
	storage.EnrollmentIDRetriever
}

func TestBasic(t *testing.T, storage allTestStorage, ctx context.Context) {
	decl, err := ddm.ParseDeclaration([]byte(testDecl))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("StoreDeclaration", func(t *testing.T) {
		testStoreDeclaration(t, storage, ctx, decl)
	})

	t.Run("TestSet", func(t *testing.T) {
		testSet(t, storage, ctx, decl, "test_golang_set1")
	})

	t.Run("TestEnrollmentSets", func(t *testing.T) {
		testEnrollments(t, storage, ctx, decl, "455399EA-4C94-4FA1-A87A-85A6CFEC4932", "test_golang_set1")
	})

	t.Run("TestSetRemoval", func(t *testing.T) {
		testSetRemoval(t, storage, ctx, decl, "test_golang_set1")
	})

	t.Run("DeleteDeclaration", func(t *testing.T) {
		testDeleteDeclaration(t, storage, ctx, decl.Identifier)
	})
}
