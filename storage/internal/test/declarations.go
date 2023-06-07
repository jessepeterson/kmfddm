package test

import (
	"context"
	"testing"

	"github.com/jessepeterson/kmfddm/ddm"
	"github.com/jessepeterson/kmfddm/http/api"
)

func testStoreDeclaration(t *testing.T, storage api.DeclarationAPIStorage, ctx context.Context, decl *ddm.Declaration) {
	_, err := storage.StoreDeclaration(ctx, decl)
	if err != nil {
		t.Fatal(err)
	}
	decls, err := storage.RetrieveDeclarations(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, v := range decls {
		if v == decl.Identifier {
			found = true
			break
		}
	}
	if !found {
		t.Error("could not find declaration id in list")
	}
	decl2, err := storage.RetrieveDeclaration(ctx, decl.Identifier)
	if err != nil {
		t.Fatal(err)
	}
	if have, want := decl2.Identifier, decl.Identifier; have != want {
		t.Errorf("have %q; want %q", have, want)
	}
	if have, want := decl2.Type, decl.Type; have != want {
		t.Errorf("have %q; want %q", have, want)
	}
	// TODO: compare PayloadJSON
}

func testDeleteDeclaration(t *testing.T, storage api.DeclarationAPIStorage, ctx context.Context, id string) {
	_, err := storage.DeleteDeclaration(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	decls, err := storage.RetrieveDeclarations(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, v := range decls {
		if v == id {
			found = true
			break
		}
	}
	if found {
		t.Error("found declaration id in list (should have been deleted)")
	}
}
