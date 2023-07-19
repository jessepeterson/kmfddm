package test

import (
	"context"
	"testing"

	"github.com/jessepeterson/kmfddm/ddm"
	"github.com/jessepeterson/kmfddm/storage"
)

type setAndDeclStorage interface {
	storage.SetDeclarationStorage
	storage.SetRetreiver
}

func testSet(t *testing.T, storage setAndDeclStorage, ctx context.Context, decl *ddm.Declaration, setName string) {
	// associate (wrong)
	_, err := storage.StoreSetDeclaration(ctx, setName, decl.Identifier+"_invalid")
	if err == nil {
		t.Fatal("should be an error")
	}

	// associate
	_, err = storage.StoreSetDeclaration(ctx, setName, decl.Identifier)
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

}

func testSetRemoval(t *testing.T, storage setAndDeclStorage, ctx context.Context, decl *ddm.Declaration, setName string) {
	// dissociate
	_, err := storage.RemoveSetDeclaration(ctx, setName, decl.Identifier)
	if err != nil {
		t.Fatal(err)
	}
}
