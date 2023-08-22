package test

import (
	"context"
	"errors"
	"testing"

	"github.com/jessepeterson/kmfddm/ddm"
	"github.com/jessepeterson/kmfddm/storage"
)

func testStoreDeclaration(t *testing.T, storage storage.DeclarationAPIStorage, ctx context.Context, decl *ddm.Declaration) {
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
	modTime, err := storage.RetrieveDeclarationModTime(ctx, decl.Identifier)
	if err != nil {
		t.Fatal(err)
	}
	if modTime.IsZero() {
		t.Error("declaration mod time is zero")
	}
	changed, err := storage.StoreDeclaration(ctx, decl2)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Error("should have changed")
	}
	decl4, err := storage.RetrieveDeclaration(ctx, decl2.Identifier)
	if err != nil {
		t.Fatal(err)
	}
	if have, want := decl2.ServerToken, decl4.ServerToken; have != want {
		t.Errorf("have %q; want %q", have, want)
	}
	err = storage.TouchDeclaration(ctx, decl2.Identifier)
	if err != nil {
		t.Fatal(err)
	}
	decl3, err := storage.RetrieveDeclaration(ctx, decl.Identifier)
	if err != nil {
		t.Fatal(err)
	}
	if have, want := decl2.ServerToken, decl3.ServerToken; have == want {
		t.Errorf("server token should not be equal: %v", have)
	}
	// TODO: compare PayloadJSON
}

func testNotFoundDeclaration(t *testing.T, store storage.DeclarationAPIStorage, ctx context.Context) {
	_, err := store.RetrieveDeclaration(ctx, "229ea7487aae57d9c26dafe54ae96e61")
	if !errors.Is(err, storage.ErrDeclarationNotFound) {
		t.Error("retrieve declaration should error not found")
	}
	err = store.TouchDeclaration(ctx, "229ea7487aae57d9c26dafe54ae96e61")
	if !errors.Is(err, storage.ErrDeclarationNotFound) {
		t.Error("touch declaration should error not found")
	}
	_, err = store.RetrieveDeclarationModTime(ctx, "229ea7487aae57d9c26dafe54ae96e61")
	if !errors.Is(err, storage.ErrDeclarationNotFound) {
		t.Error("retrieve declaration mod time should error not found")
	}
}

func testDeleteDeclaration(t *testing.T, storage storage.DeclarationAPIStorage, ctx context.Context, id string) {
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
