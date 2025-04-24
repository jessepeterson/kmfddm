// Package diskv implements a NanoMDM storage backend using the diskv key-value store.
package diskv

import (
	"hash"
	"path/filepath"

	"github.com/jessepeterson/kmfddm/storage/kv"

	nlkv "github.com/micromdm/nanolib/storage/kv"
	"github.com/micromdm/nanolib/storage/kv/kvdiskv"
	"github.com/micromdm/nanolib/storage/kv/kvtxn"
	"github.com/peterbourgon/diskv/v3"
)

// Diskv is a storage backend that uses diskv.
type Diskv struct {
	*kv.KV
}

func newBucket(path, name string) nlkv.TxnBucketWithCRUD {
	return newBucketWithTransform(path, name, kvdiskv.FlatTransform)
}

func newBucketWithTransform(path, name string, transform diskv.TransformFunction) nlkv.TxnBucketWithCRUD {
	return kvtxn.New(kvdiskv.New(diskv.New(diskv.Options{
		BasePath:     filepath.Join(path, name),
		Transform:    transform,
		CacheSizeMax: 1024 * 1024,
	})))
}

// New creates a new storage backend that uses diskv.
func New(path string, newHash func() hash.Hash) *Diskv {
	return &Diskv{KV: kv.New(
		newHash,
		newBucket(path, "declarations"),
		newBucket(path, "sets"),
		newBucket(path, "enrollments"),
		newBucket(path, "status"),
	)}
}
