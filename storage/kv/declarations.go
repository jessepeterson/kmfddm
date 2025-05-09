package kv

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"strconv"
	"strings"
	"time"

	"github.com/jessepeterson/kmfddm/ddm"
	"github.com/jessepeterson/kmfddm/storage"

	"github.com/micromdm/nanolib/storage/kv"
)

const (
	keyPfxDcl = "dcl"
	keyPfxIdx = "idx"

	keyDeclarationServerToken = "token"
	keyDeclarationTouch       = "touch"
	keyDeclarationCreated     = "created"
	keyDeclarationModified    = "modified"

	keyDeclarationType    = "type"
	keyDeclarationPayload = "payload"
)

func genServerToken(d *ddm.Declaration, touch string, created []byte, newHash func() hash.Hash) string {
	h := newHash()
	h.Write(append(append(d.Payload, created...), []byte(d.Identifier+d.Type+touch)...))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func encodeTime(t time.Time) []byte {
	return []byte(strconv.FormatInt(t.UnixMicro(), 10))
}

func decodeTime(s []byte) (time.Time, error) {
	i, err := strconv.ParseInt(string(s), 10, 64)
	return time.UnixMicro(i), err
}

// StoreDeclaration stores a declaration.
// If the declaration is new or has changed true should be returned.
//
// Note that a storage backend may try to create relations
// based on the the ddm.IdentifierRefs field.
func (s *KV) StoreDeclaration(ctx context.Context, d *ddm.Declaration) (changed bool, err error) {
	err = kv.PerformCRUDBucketTxn(ctx, s.declarations, func(ctx context.Context, b kv.CRUDBucket) error {
		var touch string
		now := time.Now()
		created := now

		var found bool
		dMap, err := kv.GetMap(ctx, b, []string{
			join(keyPfxDcl, d.Identifier, keyDeclarationTouch),
			join(keyPfxDcl, d.Identifier, keyDeclarationCreated),
			join(keyPfxDcl, d.Identifier, keyDeclarationServerToken),
		})
		if errors.Is(err, kv.ErrKeyNotFound) {
			// if any key is not found, consider the whole declaration not found
			touch = "0"
		} else if err != nil {
			// if any other error, then bail on it
			return err
		} else {
			// otherwise assume we have an existing declaration
			// setup some things to be able to compare the existing
			// declaration with the one being stored.
			found = true
			touch = string(dMap[join(keyPfxDcl, d.Identifier, keyDeclarationTouch)])
			created, err = decodeTime(dMap[join(keyPfxDcl, d.Identifier, keyDeclarationCreated)])
			if err != nil {
				return err
			}
		}

		// (re-)generate the server token based on our new (or existing) data
		serverToken := genServerToken(d, touch, encodeTime(created), s.newHash)

		if found && serverToken == string(dMap[join(keyPfxDcl, d.Identifier, keyDeclarationServerToken)]) {
			// if they match, then just bail
			return nil
		}

		// if we get here then they don't match
		changed = true

		// save it all to the kv store
		return kv.SetMap(ctx, b, map[string][]byte{
			join(keyPfxDcl, d.Identifier, keyDeclarationTouch):       []byte(touch),
			join(keyPfxDcl, d.Identifier, keyDeclarationCreated):     encodeTime(created),
			join(keyPfxDcl, d.Identifier, keyDeclarationModified):    encodeTime(now),
			join(keyPfxDcl, d.Identifier, keyDeclarationServerToken): []byte(serverToken),
			join(keyPfxDcl, d.Identifier, keyDeclarationType):        []byte(d.Type),
			join(keyPfxDcl, d.Identifier, keyDeclarationPayload):     d.Payload,
		})
	})
	return
}

// TouchDeclaration forces a change to a declaration's ServerToken only.
func (s *KV) TouchDeclaration(ctx context.Context, declarationID string) error {
	return kv.PerformCRUDBucketTxn(ctx, s.declarations, func(ctx context.Context, b kv.CRUDBucket) error {
		// first retrieve the key bits from the kv store
		dMap, err := kv.GetMap(ctx, b, []string{
			join(keyPfxDcl, declarationID, keyDeclarationType),
			join(keyPfxDcl, declarationID, keyDeclarationPayload),
			join(keyPfxDcl, declarationID, keyDeclarationTouch),
			join(keyPfxDcl, declarationID, keyDeclarationCreated),
		})
		if errors.Is(err, kv.ErrKeyNotFound) {
			// wrap kv error in the proper storage error
			return fmt.Errorf("%w: %v", storage.ErrDeclarationNotFound, err)
		}

		// bump the touch index number
		touch, err := strconv.Atoi(string(dMap[join(keyPfxDcl, declarationID, keyDeclarationTouch)]))
		if err != nil {
			return err
		}
		touch++

		// assemble enough of a declaration to compute the new server token
		d := &ddm.Declaration{
			Identifier: declarationID,
			Type:       string(dMap[join(keyPfxDcl, declarationID, keyDeclarationType)]),
			Payload:    dMap[join(keyPfxDcl, declarationID, keyDeclarationPayload)],
		}

		// save the changes back to the kv store
		return kv.SetMap(ctx, b, map[string][]byte{
			join(keyPfxDcl, declarationID, keyDeclarationTouch):       []byte(strconv.Itoa(touch)),
			join(keyPfxDcl, declarationID, keyDeclarationServerToken): []byte(genServerToken(d, strconv.Itoa(touch), dMap[join(keyPfxDcl, declarationID, keyDeclarationCreated)], s.newHash)),
			join(keyPfxDcl, declarationID, keyDeclarationModified):    encodeTime(time.Now()),
		})
	})
}

// DeleteDeclaration deletes a declaration.
// If the declaration was deleted true should be returned.
//
// Implementations should return an error if there are declarations
// that depend on it or if the declaration is associated with a set.
func (s *KV) DeleteDeclaration(ctx context.Context, declarationID string) (changed bool, err error) {
	err = kv.PerformBucketTxn(ctx, s.declarations, func(ctx context.Context, b kv.Bucket) error {
		// first check if the declaration exists
		if found, err := b.Has(ctx, join(keyPfxDcl, declarationID, keyDeclarationType)); err != nil {
			return err
		} else if found {
			changed = true

			// then check if we're in any sets
			sets, err := getDeclarationSets(ctx, s.sets, declarationID)
			if err != nil {
				return err
			}
			if len(sets) > 0 {
				return fmt.Errorf("declaration is referenced by %d sets", len(sets))
			}
		}

		// finally just unconditionally clear everything out of the kv store
		return kv.DeleteSlice(ctx, b, []string{
			join(keyPfxDcl, declarationID, keyDeclarationTouch),
			join(keyPfxDcl, declarationID, keyDeclarationCreated),
			join(keyPfxDcl, declarationID, keyDeclarationModified),
			join(keyPfxDcl, declarationID, keyDeclarationServerToken),
			join(keyPfxDcl, declarationID, keyDeclarationType),
			join(keyPfxDcl, declarationID, keyDeclarationPayload),
		})
	})
	return
}

// RetrieveDeclaration retrieves a declaration from storage.
func (s *KV) RetrieveDeclaration(ctx context.Context, declarationID string) (*ddm.Declaration, error) {
	dMap, err := kv.GetMap(ctx, s.declarations, []string{
		join(keyPfxDcl, declarationID, keyDeclarationServerToken),
		join(keyPfxDcl, declarationID, keyDeclarationType),
		join(keyPfxDcl, declarationID, keyDeclarationPayload),
	})
	if errors.Is(err, kv.ErrKeyNotFound) {
		// wrap kv error in the proper storage error
		return nil, fmt.Errorf("%w: %v", storage.ErrDeclarationNotFound, err)
	} else if err != nil {
		return nil, err
	}

	// assemble the declaration from the get map
	d := &ddm.Declaration{
		ServerToken: string(dMap[join(keyPfxDcl, declarationID, keyDeclarationServerToken)]),
		Identifier:  declarationID,
		Type:        string(dMap[join(keyPfxDcl, declarationID, keyDeclarationType)]),
		Payload:     dMap[join(keyPfxDcl, declarationID, keyDeclarationPayload)],
	}

	// assemble a faux declaration for JSON marshalling
	dm := &struct {
		ServerToken string
		Identifier  string
		Type        string
		Payload     json.RawMessage
	}{
		ServerToken: d.ServerToken,
		Identifier:  d.Identifier,
		Type:        d.Type,
		Payload:     d.Payload,
	}
	d.Raw, err = json.MarshalIndent(dm, "", "\t")

	return d, err
}

// RetrieveDeclarationModTime retrieves the last modification time of the declaration.
func (s *KV) RetrieveDeclarationModTime(ctx context.Context, declarationID string) (time.Time, error) {
	modTime, err := s.declarations.Get(ctx, join(keyPfxDcl, declarationID, keyDeclarationModified))
	if errors.Is(err, kv.ErrKeyNotFound) {
		// wrap kv error in the proper storage error
		return time.Time{}, fmt.Errorf("%w: %v", storage.ErrDeclarationNotFound, err)
	}
	return decodeTime(modTime)
}

// RetrieveDeclarations retrieves a list of all declarations.
func (s *KV) RetrieveDeclarations(ctx context.Context) (declarations []string, _ error) {
	for key := range s.declarations.Keys(ctx, nil) {
		if strings.HasPrefix(key, keyPfxDcl+keySep) && strings.HasSuffix(key, keySep+keyDeclarationType) {
			declarations = append(declarations, key[len(keyPfxDcl+keySep):len(key)-len(keySep+keyDeclarationType)])
		}
	}
	return
}
