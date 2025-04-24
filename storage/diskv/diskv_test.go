package diskv

import (
	"context"
	"hash"
	"testing"

	"github.com/cespare/xxhash"
	"github.com/jessepeterson/kmfddm/storage/test"
)

func TestDiskv(t *testing.T) {
	s := New(t.TempDir(), func() hash.Hash { return xxhash.New() })
	ctx := context.Background()

	t.Run("TestBasic", func(t *testing.T) {
		test.TestBasic(t, s, ctx)
	})
	t.Run("TestBasicStatus", func(t *testing.T) {
		test.TestBasicStatus(t, "../test", s, ctx)
	})
}
