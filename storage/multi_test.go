package storage

import (
	"context"
	"errors"
	"testing"
)

const testDecl2 = `{
    "Type": "com.apple.configuration.management.test",
    "Payload": {
        "Echo": "Foo"
    },
    "Identifier": "test_golang_9c6319a1-df3e-48db-aada-6ee944fc507b"
}`

func TestMulti(t *testing.T) {
	ctx := context.Background()

	t.Run("empty", func(t *testing.T) {
		empty := NewMulti()

		_, err := empty.RetrieveEnrollmentDeclarationJSON(ctx, "foo", "bar", "baz")
		if !errors.Is(err, ErrDeclarationNotFound) {
			t.Errorf("incorrect error type: %v", err)
		}

		decls, err := empty.RetrieveDeclarationItems(ctx, "baz")
		if err != nil {
			t.Fatal(err)
		}
		if have, want := len(decls), 0; have != want {
			t.Errorf("decl len wrong: have: %v, want: %v", have, want)
		}
	})

	t.Run("one", func(t *testing.T) {
		s, err := NewMockStore([]byte(testDecl), "abc", "baz")
		if err != nil {
			t.Fatal(err)
		}

		one := NewMulti(s)

		// check that when we request a missing decl, we get the right error
		_, err = one.RetrieveEnrollmentDeclarationJSON(ctx, "foo", "bar", "baz")
		if !errors.Is(err, ErrDeclarationNotFound) {
			t.Errorf("incorrect error type: %v", err)
		}

		decls, err := one.RetrieveDeclarationItems(ctx, "baz")
		if err != nil {
			t.Fatal(err)
		}
		if have, want := len(decls), 1; have != want {
			t.Errorf("decl len wrong: have: %v, want: %v", have, want)
		}

	})

	t.Run("multi", func(t *testing.T) {
		s1, err := NewMockStore([]byte(testDecl), "abc", "baz")
		if err != nil {
			t.Fatal(err)
		}

		s2, err := NewMockStore([]byte(testDecl2), "abc", "baz")
		if err != nil {
			t.Fatal(err)
		}

		multi := NewMulti(s1, s2)

		// check that when we request a missing decl, we get the right error
		_, err = multi.RetrieveEnrollmentDeclarationJSON(ctx, "foo", "bar", "baz")
		if !errors.Is(err, ErrDeclarationNotFound) {
			t.Errorf("incorrect error type: %v", err)
		}

		// check that we get the declaration from one of the stores
		_, err = multi.RetrieveEnrollmentDeclarationJSON(ctx, "test_golang_9c6319a1-df3e-48db-aada-6ee944fc507b", "configuration", "baz")
		if errors.Is(err, ErrDeclarationNotFound) {
			t.Error("declaration should be found")
		} else if err != nil {
			t.Fatal(err)
		}

		// check that we get the declaration from the other store
		_, err = multi.RetrieveEnrollmentDeclarationJSON(ctx, "test_golang_bd23307c-4704-4a31-9b8f-a5e8406d8633", "configuration", "baz")
		if errors.Is(err, ErrDeclarationNotFound) {
			t.Error("declaration should be found")
		} else if err != nil {
			t.Fatal(err)
		}

		// check for correct number of declaration items
		decls, err := multi.RetrieveDeclarationItems(ctx, "baz")
		if err != nil {
			t.Fatal(err)
		}
		if have, want := len(decls), 2; have != want {
			t.Errorf("decl len wrong: have: %v, want: %v", have, want)
		}

		// make sure all declarations are found (as configured in the test storage)
		// for the declaration items
		ids := []string{"test_golang_9c6319a1-df3e-48db-aada-6ee944fc507b", "test_golang_bd23307c-4704-4a31-9b8f-a5e8406d8633"}
		for _, d := range decls {
			if !in(ids, d.Identifier) {
				t.Errorf("declaration not found in known set: %s", d.Identifier)
			}
		}

	})
}

func in(haystack []string, needle string) bool {
	for _, i := range haystack {
		if i == needle {
			return true
		}
	}
	return false
}
