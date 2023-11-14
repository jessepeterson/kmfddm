package build

import (
	"fmt"
	"hash"

	"github.com/jessepeterson/kmfddm/ddm"
)

// NewHash returns a newly instantiated hashing function.
type NewHash func() hash.Hash

// Builders construct declaration items and sync token structures.
// They operate similar to hashers.
type Builder interface {
	// Add incrementally adds d to the builder.
	Add(d *ddm.Declaration)

	// Finalize finishes composing the structure (i.e. computing server tokens).
	Finalize()
}

func tokenHashFinalize(h hash.Hash) string {
	return fmt.Sprintf("%x", h.Sum(nil))
}

func tokenHashWrite(h hash.Hash, d *ddm.Declaration) {
	h.Write([]byte(d.ServerToken))
}
