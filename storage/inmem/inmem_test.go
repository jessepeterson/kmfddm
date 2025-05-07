package inmem

import (
	"context"
	"hash"
	"hash/fnv"
	"testing"

	"github.com/jessepeterson/kmfddm/test/e2e"
)

func TestInMem(t *testing.T) {
	s := New(func() hash.Hash { return fnv.New128() })
	ctx := context.Background()

	t.Run("TestE2E", func(t *testing.T) {
		e2e.TestE2E(t, ctx, s)
	})
}
