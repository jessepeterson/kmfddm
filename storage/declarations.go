package storage

import (
	"context"
	"time"

	"github.com/jessepeterson/kmfddm/ddm"
)

type Toucher interface {
	// TouchDeclaration forces a change to a declaration's ServerToken only.
	TouchDeclaration(ctx context.Context, declarationID string) error
}

type DeclarationStorer interface {
	// StoreDeclaration stores a declaration.
	// If the declaration is new or has changed true should be returned.
	StoreDeclaration(ctx context.Context, d *ddm.Declaration) (bool, error)
}

type DeclarationDeleter interface {
	// DeleteDeclaration deletes a declaration.
	// If the declaration was deleted true should be returned.
	// Implementations should return an error if the declaration is associated with a set.
	DeleteDeclaration(ctx context.Context, declarationID string) (bool, error)
}

type DeclarationAPIRetriever interface {
	// RetrieveDeclaration retrieves a declaration from storage.
	RetrieveDeclaration(ctx context.Context, declarationID string) (*ddm.Declaration, error)

	// RetrieveDeclarationModTime retrieves the last modification time of the declaration.
	RetrieveDeclarationModTime(ctx context.Context, declarationID string) (time.Time, error)
}

type DeclarationsRetriever interface {
	// RetrieveDeclarations retrieves a list of all declarations.
	RetrieveDeclarations(ctx context.Context) ([]string, error)
}
