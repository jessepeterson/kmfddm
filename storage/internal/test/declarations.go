package test

import (
	"context"
	"testing"

	"github.com/jessepeterson/kmfddm/ddm"
	"github.com/jessepeterson/kmfddm/http/api"
)

const testDecl = `{
    "Type": "com.apple.configuration.management.test",
    "Payload": {
        "Echo": "Foo"
    },
    "Identifier": "test_mysql_9e6a3aa7-5e4b-4d38-aacf-0f8058b2a899"
}`

// TestDeclarations performs a simple store, retrieve and delete test for declarations
func TestDeclarations(t *testing.T, storage api.DeclarationAPIStorage, ctx context.Context) {
	decl, err := ddm.ParseDeclaration([]byte(testDecl))
	if err != nil {
		t.Fatal(err)
	}
	_, err = storage.StoreDeclaration(ctx, decl)
	if err != nil {
		t.Fatal(err)
	}
	decls, err := storage.RetrieveDeclarations(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, v := range decls {
		if v == "test_mysql_9e6a3aa7-5e4b-4d38-aacf-0f8058b2a899" {
			found = true
			break
		}
	}
	if found != true {
		t.Error("could not find declaration id in list")
	}
	decl2, err := storage.RetrieveDeclaration(ctx, "test_mysql_9e6a3aa7-5e4b-4d38-aacf-0f8058b2a899")
	if err != nil {
		t.Fatal(err)
	}
	if have, want := decl2.Identifier, "test_mysql_9e6a3aa7-5e4b-4d38-aacf-0f8058b2a899"; have != want {
		t.Errorf("have %q; want %q", have, want)
	}
	if have, want := decl2.Type, "com.apple.configuration.management.test"; have != want {
		t.Errorf("have %q; want %q", have, want)
	}
	_, err = storage.DeleteDeclaration(ctx, "test_mysql_9e6a3aa7-5e4b-4d38-aacf-0f8058b2a899")
	if err != nil {
		t.Fatal(err)
	}
	decls, err = storage.RetrieveDeclarations(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found = false
	for _, v := range decls {
		if v == "test_mysql_9e6a3aa7-5e4b-4d38-aacf-0f8058b2a899" {
			found = true
			break
		}
	}
	if found == true {
		t.Error("found declaration id in list (should have been deleted)")
	}
}
