//go:build !windows

package file

import (
	"context"
	"hash"
	"testing"

	"github.com/cespare/xxhash"
	"github.com/jessepeterson/kmfddm/storage/test"
	"github.com/jessepeterson/kmfddm/test/e2e"
)

func TestFile(t *testing.T) {
	s, err := New(t.TempDir(), func() hash.Hash { return xxhash.New() })
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	test.TestBasic(t, s, ctx)
	test.TestBasicStatus(t, "../test", s, ctx)
	t.Run("TestE2E", func(t *testing.T) {
		e2e.TestE2E(t, ctx, s)
	})

}
