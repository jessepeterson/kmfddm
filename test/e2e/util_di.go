package e2e

import (
	"sort"
	"testing"

	"github.com/jessepeterson/kmfddm/ddm"
)

// di1 needs to have non-empty tokens
func diEqual(t *testing.T, di1, di2 *ddm.DeclarationItems) {
	t.Helper()

	if di1 == nil && di2 == nil {
		return
	}
	if di1 == nil || di2 == nil {
		t.Fatalf("declaration items not equal: d1 nil: %v, d2 nil: %v", di1 == nil, di2 == nil)
	}

	f := func(s []ddm.ManifestDeclaration) {
		sort.Slice(s, func(i, j int) bool { return s[i].Identifier < s[j].Identifier })
	}

	f(di1.Declarations.Activations)
	f(di1.Declarations.Configurations)
	f(di1.Declarations.Management)
	f(di1.Declarations.Assets)

	f(di1.Declarations.Activations)
	f(di1.Declarations.Configurations)
	f(di1.Declarations.Management)
	f(di1.Declarations.Assets)

	c := func(t *testing.T, typ string, md1, md2 []ddm.ManifestDeclaration) {
		t.Helper()
		if len(md1) != len(md2) {
			t.Fatalf("%s declarations: %v vs %v", typ, md1, md2)
		}
		for i := range md1 {
			if md1[i].ServerToken == "" {
				t.Errorf("%s declaration (%d) missing ServerToken: %v", typ, i, md1[i])
			}
			if md1[i].Identifier != md2[i].Identifier {
				t.Fatalf("%s declaration (%d): %v vs %v", typ, i, md1[i], md2[i])
			}
		}
	}

	if di1.DeclarationsToken == "" {
		t.Error("empty declarations token")
	}

	c(t, "activation", di1.Declarations.Activations, di2.Declarations.Activations)
	c(t, "configuration", di1.Declarations.Configurations, di2.Declarations.Configurations)
	c(t, "assets", di1.Declarations.Assets, di2.Declarations.Assets)
	c(t, "management", di1.Declarations.Management, di2.Declarations.Management)
}
