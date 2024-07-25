package build

import (
	"crypto/sha256"
	"hash"
	"testing"

	"github.com/jessepeterson/kmfddm/ddm"
)

func TestBuilder(t *testing.T) {
	d := &ddm.Declaration{
		Identifier:  "com.foo.bar.a",
		Type:        "com.apple.configuration.test",
		ServerToken: "a",
	}
	d.ServerToken = "a"
	b := NewDIBuilder(func() hash.Hash { return sha256.New() })
	b.Add(d)
	b.Finalize()

	if have, want := len(b.DeclarationItems.Declarations.Configurations), 1; have != want {
		t.Errorf("wrong number of declarations (configurations): have: %v, want: %v", have, want)
	}

	tokens1 := b.DeclarationsToken
	d.ServerToken = "b"
	d.Identifier = "com.foo.bar.b"
	b.Add(d)
	b.Finalize()

	if have, want := len(b.DeclarationItems.Declarations.Configurations), 2; have != want {
		t.Errorf("wrong number of declarations (configurations): have: %v, want: %v", have, want)
	}

	if have, want := tokens1, b.DeclarationsToken; have == want {
		t.Errorf("tokens should be different: %q", have)
	}

	// "invalid" type (not one of the four) - should not be added to the
	// built declaration items
	d.Type = "com.example.fubar"

	b2 := NewDIBuilder(func() hash.Hash { return sha256.New() })
	b2.Add(d)
	b2.Finalize()

	if have, want := len(b2.DeclarationItems.Declarations.Activations), 0; have != want {
		t.Errorf("wrong number of declarations (activations): have: %v, want: %v", have, want)
	}

	if have, want := len(b2.DeclarationItems.Declarations.Assets), 0; have != want {
		t.Errorf("wrong number of declarations (assets): have: %v, want: %v", have, want)
	}

	if have, want := len(b2.DeclarationItems.Declarations.Configurations), 0; have != want {
		t.Errorf("wrong number of declarations (configurations): have: %v, want: %v", have, want)
	}

	if have, want := len(b2.DeclarationItems.Declarations.Management), 0; have != want {
		t.Errorf("wrong number of declarations (management): have: %v, want: %v", have, want)
	}
}
