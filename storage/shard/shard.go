// Package shard is a dynamic storage backend that synthesizes a shard declaration.
package shard

import (
	"context"
	"hash/fnv"
	"strconv"

	"github.com/jessepeterson/kmfddm/ddm"
	"github.com/jessepeterson/kmfddm/storage"
)

const (
	ManifestType          = "management"
	DeclarationType       = "com.apple.management.properties"
	DeclarationIdentifier = "com.github.jessepeterson.kmfddm.storage.shard.v1"
)

// ShardStorage is a dynamic storage backend that synthesizes a shard declaration.
// The declaration is a management properties declaration that sets the "shard"
// property to the shard number for the enrollment. This can then be used in
// activation predicates.
// The shard number is computed from the enrollment ID, hashed using FNV1 modulo 101.
type ShardStorage struct{}

// NewShardStorage creates a new shard storage.
func NewShardStorage() *ShardStorage {
	return new(ShardStorage)
}

func computeShard(input string) string {
	hash := fnv.New32()
	hash.Write([]byte(input))
	result := hash.Sum32()
	return strconv.Itoa(int(result % 101))
}

func serverToken(enrollmentID string) string {
	return "shard=" + computeShard(enrollmentID) + ";version=1"
}

// RetrieveDeclarationItems synthesizes a dynamic shard declaration.
// Used for injection into the declaration items and sync tokens.
func (s *ShardStorage) RetrieveDeclarationItems(_ context.Context, enrollmentID string) ([]*ddm.Declaration, error) {
	return []*ddm.Declaration{{
		Type:        DeclarationType,
		Identifier:  DeclarationIdentifier,
		ServerToken: serverToken(enrollmentID),
	}}, nil
}

// RetrieveEnrollmentDeclarationJSON synthesizes a dynamic shard declaration.
func (s *ShardStorage) RetrieveEnrollmentDeclarationJSON(_ context.Context, declarationID, declarationType, enrollmentID string) ([]byte, error) {
	if declarationID != DeclarationIdentifier || declarationType != ManifestType {
		// if caller hasn't targeted us exactly then bail as not found quickly.
		// if used in a multi-
		return nil, storage.ErrDeclarationNotFound
	}
	// avoid marshalling json by doing string concat as an optimization
	json := `{
	"Type": "` + DeclarationType + `",
	"Identifier": "` + DeclarationIdentifier + `",
	"Payload": {
		"shard": ` + computeShard(enrollmentID) + `
	},
	"ServerToken": "` + serverToken(enrollmentID) + `"
}`
	return []byte(json), nil
}
