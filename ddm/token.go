package ddm

import (
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
