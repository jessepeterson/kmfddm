package ddm

import (
	"fmt"
	"hash"
	"time"
)

// See https://developer.apple.com/documentation/devicemanagement/synchronizationtokens
type SynchronizationTokens struct {
	DeclarationsToken string
	Timestamp         time.Time
}

// See https://developer.apple.com/documentation/devicemanagement/tokensresponse
type TokensResponse struct {
	SyncTokens SynchronizationTokens
}

type TokensBuilder struct {
	TokensResponse
	hash.Hash
}

func NewTokensBuilder(newHash func() hash.Hash) *TokensBuilder {
	return &TokensBuilder{Hash: newHash()}
}

func (b *TokensBuilder) AddDeclarationData(d *Declaration) {
	tokenHashWrite(b.Hash, d)
}

func (b *TokensBuilder) Finalize() {
	b.TokensResponse.SyncTokens.DeclarationsToken = fmt.Sprintf("%x", b.Hash.Sum(nil))
	b.SyncTokens.Timestamp = time.Now().UTC().Truncate(time.Second)
}
