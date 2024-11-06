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

// ShardFunc computes a shard value. A string of a decimal value
// between 0 and 100 inclusive should be returned. I.e. "42".
type ShardFunc func(string) string

// FNV1Shard hashes input using FNV1 modulo 101.
func FNV1Shard(input string) string {
	hash := fnv.New32()
	hash.Write([]byte(input))
	result := hash.Sum32()
	return strconv.Itoa(int(result % 101))
}

// ShardStorage is a dynamic storage backend that synthesizes a shard declaration.
// The declaration is a management properties declaration that sets the "shard"
// property to the shard number for the enrollment. This can then be used in
// activation predicates.
// The shard number is computed from the enrollment ID.
type ShardStorage struct {
	shardFunc ShardFunc
}

type Option func(*ShardStorage)

// WithShardFunc sets the shard function to f.
func WithShardFunc(f ShardFunc) Option {
	if f == nil {
		panic("nil shard func")
	}
	return func(s *ShardStorage) {
		s.shardFunc = f
	}
}

// NewShardStorage creates a new shard storage.
// By default the shard function is hashed using [FNV1Shard].
// Use [WithShardFunc] to change it.
func NewShardStorage(opts ...Option) *ShardStorage {
	s := &ShardStorage{shardFunc: FNV1Shard}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *ShardStorage) serverToken(enrollmentID string) string {
	return "shard=" + s.shardFunc(enrollmentID) + ";version=1"
}

// RetrieveDeclarationItems synthesizes a dynamic shard declaration.
// Used for injection into the declaration items and sync tokens.
func (s *ShardStorage) RetrieveDeclarationItems(_ context.Context, enrollmentID string) ([]*ddm.Declaration, error) {
	return []*ddm.Declaration{{
		Type:        DeclarationType,
		Identifier:  DeclarationIdentifier,
		ServerToken: s.serverToken(enrollmentID),
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
		"shard": ` + s.shardFunc(enrollmentID) + `
	},
	"ServerToken": "` + s.serverToken(enrollmentID) + `"
}`
	return []byte(json), nil
}
