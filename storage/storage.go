// Package storage defines shared types and interfaces for storage.
package storage

// DeclarationAPIStorage are storage interfaces relating to declarations.
type DeclarationAPIStorage interface {
	Toucher
	DeclarationStorer
	DeclarationDeleter
	DeclarationAPIRetriever
	DeclarationsRetriever
}

// SetStorage are storage interfaces related to sets.
type SetDeclarationStorage interface {
	DeclarationSetRetriever
	SetDeclarationsRetriever
	SetDeclarationStorer
	SetDeclarationRemover
}

// EnrollmentSetStorage are storage interfaces related to MDM enrollment IDs.
type EnrollmentSetStorage interface {
	EnrollmentSetsRetriever
	EnrollmentSetStorer
	EnrollmentSetRemover
}

// StatusAPIStorage are storage interfaces related to retrieving status channel data.
type StatusAPIStorage interface {
	StatusDeclarationsRetriever
	StatusErrorsRetriever
	StatusValuesRetriever
	StatusReportRetriever
}
