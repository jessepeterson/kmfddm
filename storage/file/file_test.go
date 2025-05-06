//go:build !windows

package file

import (
	"context"
	"hash"
	"hash/fnv"
	"testing"

	"github.com/jessepeterson/kmfddm/test/e2e"
)

func TestFile(t *testing.T) {
	s, err := New(t.TempDir(), func() hash.Hash { return fnv.New128() })
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	t.Run("TestE2E", func(t *testing.T) {
		e2e.TestE2E(t, ctx, s)
	})

}
