package ddm

import (
	"hash"
	"time"
)

// SynchronizationTokens contains the sync token for the set of declarations for a DDM client.
// See https://developer.apple.com/documentation/devicemanagement/synchronizationtokens
type SynchronizationTokens struct {
	DeclarationsToken string
	Timestamp         time.Time
}

// TokensResponse is the container for the sync tokens for serializing.
// See https://developer.apple.com/documentation/devicemanagement/tokensresponse
type TokensResponse struct {
	SyncTokens SynchronizationTokens
}

// TokensBuilder incrementally builds the DDM Sync Tokens structure for later serializing.
type TokensBuilder struct {
	hash         hash.Hash
	newTimestamp func() time.Time
	TokensResponse
}

// NewTokensBuilder constructs a new Sync Tokens builder.
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

// Add adds a declaration d to the Sync Tokens builder.
func (b *TokensBuilder) Add(d *Declaration) {
	tokenHashWrite(b.hash, d)
}

// Finalize finishes building the Sync Tokens by computing the final Declarations Token and timestamp.
func (b *TokensBuilder) Finalize() {
	b.TokensResponse.SyncTokens.DeclarationsToken = tokenHashFinalize(b.hash)
	b.SyncTokens.Timestamp = b.newTimestamp().Truncate(time.Second)
}
