// Package kv implements a storage backend that uses key-value stores.
package kv

import (
	"hash"
	"strings"

	"github.com/micromdm/nanolib/storage/kv"
)

// KV is a storage backend that uses key-value stores.
type KV struct {
	newHash                                 func() hash.Hash
	declarations, sets, enrollments, status kv.TxnBucketWithCRUD
}

// New creates a new storage backend that uses key-value stores.
func New(newHash func() hash.Hash, declarations, sets, enrollments, status kv.TxnBucketWithCRUD) *KV {
	if newHash == nil {
		panic("nil hasher")
	}
	if declarations == nil || sets == nil || enrollments == nil || status == nil {
		panic("nil bucket")
	}

	return &KV{
		newHash:      newHash,
		declarations: declarations,
		sets:         sets,
		enrollments:  enrollments,
		status:       status,
	}
}

const (
	keySep   = "."
	valueSet = "1"
)

// join concatenates s together by placing [keySep] between segments.
// TODO: replace all occurances of this with simple string concat for performance?
func join(s ...string) string {
	return strings.Join(s, keySep)
}
