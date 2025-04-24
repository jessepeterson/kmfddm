package storage

import "context"

type DeclarationSetRetriever interface {
	// RetrieveDeclarationSets retrieves the list of set names for declarationID.
	RetrieveDeclarationSets(ctx context.Context, declarationID string) (setNames []string, err error)
}

type SetDeclarationsRetriever interface {
	// RetrieveSetDeclarations retreives the list of declarations IDs for setName.
	RetrieveSetDeclarations(ctx context.Context, setName string) (declarationIDs []string, err error)
}

type SetDeclarationStorer interface {
	// StoreSetDeclaration associates setName and declarationID.
	// If the association is created true should be returned.
	// It should not be an error if the association does not exist.
	StoreSetDeclaration(ctx context.Context, setName, declarationID string) (bool, error)
}

type SetDeclarationRemover interface {
	// StoreSetDeclaration dissociates setName and declarationID.
	// If the association is removed true should be returned.
	// It should not be an error if the association does not exist.
	RemoveSetDeclaration(ctx context.Context, setName, declarationID string) (bool, error)
}

type SetRetreiver interface {
	// RetrieveSets returns the list of all sets.
	RetrieveSets(ctx context.Context) ([]string, error)
}
