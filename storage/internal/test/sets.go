package test

import (
	"context"
	"testing"

	"github.com/jessepeterson/kmfddm/ddm"
	"github.com/jessepeterson/kmfddm/http/api"
)

type setAndDeclStorage interface {
	api.SetAPIStorage
	api.DeclarationAPIStorage
}

func testSet(t *testing.T, storage setAndDeclStorage, ctx context.Context, decl *ddm.Declaration, setName string) {
	// associate
	_, err := storage.StoreSetDeclaration(ctx, setName, decl.Identifier)
	if err != nil {
		t.Fatal(err)
	}

	// find in list
	sets, err := storage.RetrieveSets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range sets {
		if setName == s {
			found = true
			break
		}
	}
	if !found {
		t.Error("could not find set in list")
	}

	// find the ref
	setNames, err := storage.RetrieveDeclarationSets(ctx, decl.Identifier)
	if err != nil {
		t.Fatal(err)
	}
	found = false
	for _, v := range setNames {
		if setName == v {
			found = true
			break
		}
	}
	if !found {
		t.Error("could not find set in declaration sets list")
	}

	// find the backref
	decls, err := storage.RetrieveSetDeclarations(ctx, setName)
	if err != nil {
		t.Fatal(err)
	}
	found = false
	for _, v := range decls {
		if decl.Identifier == v {
			found = true
			break
		}
	}
	if !found {
		t.Error("could not find declaration in declaration sets list")
	}

	// dissociate
	_, err = storage.RemoveSetDeclaration(ctx, setName, decl.Identifier)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSets(t *testing.T, storage setAndDeclStorage, ctx context.Context) {
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

	t.Run("DeleteDeclaration", func(t *testing.T) {
		testDeleteDeclaration(t, storage, ctx, decl.Identifier)
	})
}
