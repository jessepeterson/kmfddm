package build

import (
	"hash"
	"time"

	"github.com/jessepeterson/kmfddm/ddm"
)

// TokensBuilder incrementally builds the DDM sync tokens structure for later serializing.
type TokensBuilder struct {
	hash         hash.Hash
	newTimestamp func() time.Time
	ddm.TokensResponse
}

// NewTokensBuilder constructs a new sync tokens builder.
// It will panic if provided with a nil hasher.
func NewTokensBuilder(newHash NewHash) *TokensBuilder {
	if newHash == nil {
		panic("nil hasher")
	}
	hash := newHash()
	if hash == nil {
		panic("nil hash")
	}
	return &TokensBuilder{
		hash:         hash,
		newTimestamp: func() time.Time { return time.Now().UTC() },
	}
}

// Add adds a declaration d to the sync tokens builder.
func (b *TokensBuilder) Add(d *ddm.Declaration) {
	tokenHashWrite(b.hash, d)
}

// Finalize finishes building the sync tokens by computing the final Declarations Token and timestamp.
func (b *TokensBuilder) Finalize() {
	b.TokensResponse.SyncTokens.DeclarationsToken = tokenHashFinalize(b.hash)
	b.SyncTokens.Timestamp = b.newTimestamp().Truncate(time.Second)
}
