package ddm

import (
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
