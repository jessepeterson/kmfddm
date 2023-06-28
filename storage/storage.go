// Package storage defines shared types and interfaces for storage.
package storage

import (
	"context"
)

type Toucher interface {
	// TouchDeclaration forces a change only to a declaration's ServerToken.
	TouchDeclaration(ctx context.Context, declarationID string) error
}
