package ddm

import (
	"crypto/sha256"
	"hash"
	"testing"
)

func TestManifestType(t *testing.T) {
	for _, ts := range []struct {
		in, out string
	}{
		{"com.apple.configuration.management.test", "configuration"},
		{"com.apple..configuration.management.test", ""},
		{"com.apple", ""},
		{"com.apple.", ""},
		{"com.apple.c", "c"},
	} {
		if have, want := ManifestType(ts.in), ts.out; have != want {
			t.Errorf("have: %q, want: %q", have, want)
		}
	}
}

func TestBuilder(t *testing.T) {
	d := &Declaration{
		Identifier:  "com.foo.bar.a",
		Type:        "com.apple.configuration.test",
		ServerToken: "a",
	}
	d.ServerToken = "a"
	b := NewDIBuilder(func() hash.Hash { return sha256.New() })
	b.Add(d)
	b.Finalize()

	tokens1 := b.DeclarationsToken
	d.ServerToken = "b"
	d.Identifier = "com.foo.bar.b"
	b.Add(d)
	b.Finalize()

	if have, want := tokens1, b.DeclarationsToken; have == want {
		t.Errorf("tokens should be different: %q", have)
	}
}
