// Package inmem implements an in-memory storage backend.
package inmem

import (
	"hash"

	"github.com/jessepeterson/kmfddm/storage/kv"

	"github.com/micromdm/nanolib/storage/kv/kvmap"
	"github.com/micromdm/nanolib/storage/kv/kvtxn"
)

// InMem is an in-memory storage backend.
type InMem struct {
	*kv.KV
}

// New creates a new in-memory storage backend.
func New(newHash func() hash.Hash) *InMem {
	return &InMem{KV: kv.New(
		newHash,
		kvtxn.New(kvmap.New()),
		kvtxn.New(kvmap.New()),
		kvtxn.New(kvmap.New()),
		kvtxn.New(kvmap.New()),
	)}
}
