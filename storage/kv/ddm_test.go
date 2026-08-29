package kv

import (
	"context"
	"encoding/json"
	"hash"
	"hash/fnv"
	"reflect"
	"testing"

	"github.com/jessepeterson/kmfddm/ddm"

	"github.com/micromdm/nanolib/storage/kv/kvmap"
	"github.com/micromdm/nanolib/storage/kv/kvtxn"
)

const testDeclType = "com.apple.configuration.management.test"

func newTestKV() *KV {
	return New(
		func() hash.Hash { return fnv.New128() },
		kvtxn.New(kvmap.New()),
		kvtxn.New(kvmap.New()),
		kvtxn.New(kvmap.New()),
		kvtxn.New(kvmap.New()),
	)
}

func declarationsToken(t *testing.T, s *KV, ctx context.Context, enrollmentID string) string {
	t.Helper()

	tokensJSON, err := s.RetrieveTokensJSON(ctx, enrollmentID)
	if err != nil {
		t.Fatal(err)
	}

	var tokens ddm.TokensResponse
	if err = json.Unmarshal(tokensJSON, &tokens); err != nil {
		t.Fatal(err)
	}

	return tokens.SyncTokens.DeclarationsToken
}

// TestDeclarationItemsOrder tests that the declaration items of an enrollment
// are returned in the same order every time. The declarations token is a hash
// of the declarations in the order they are returned, so an unstable order
// means a token that differs on every request and enrolled devices that
// re-synchronize their declarations continuously.
func TestDeclarationItemsOrder(t *testing.T) {
	const enrollmentID = "golang_test_enrollment_order"
	const setName = "golang_test_set_order"

	ctx := context.Background()
	s := newTestKV()

	// more than one declaration: a single declaration only has one order.
	for _, id := range []string{"golang.test.d", "golang.test.b", "golang.test.a", "golang.test.c"} {
		d, err := ddm.ParseDeclaration([]byte(`{"Type":"` + testDeclType + `","Identifier":"` + id + `","Payload":{"Echo":"test"}}`))
		if err != nil {
			t.Fatal(err)
		}
		if _, err = s.StoreDeclaration(ctx, d); err != nil {
			t.Fatal(err)
		}
		if _, err = s.StoreSetDeclaration(ctx, setName, id); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := s.StoreEnrollmentSet(ctx, enrollmentID, setName); err != nil {
		t.Fatal(err)
	}

	// Go randomizes map iteration order, so comparing two retrievals could
	// pass by luck. Repeat enough that a random order is not plausible.
	const retrievals = 60

	var wantIDs []string
	var wantToken string

	for i := 0; i < retrievals; i++ {
		items, err := s.RetrieveDeclarationItems(ctx, enrollmentID)
		if err != nil {
			t.Fatal(err)
		}

		haveIDs := make([]string, 0, len(items))
		for _, d := range items {
			haveIDs = append(haveIDs, d.Identifier)
		}

		haveToken := declarationsToken(t, s, ctx, enrollmentID)

		if i == 0 {
			wantIDs = haveIDs
			wantToken = haveToken
			continue
		}

		if have, want := haveIDs, wantIDs; !reflect.DeepEqual(have, want) {
			t.Fatalf("retrieval %d: declaration item order: have: %v, want: %v", i, have, want)
		}

		if have, want := haveToken, wantToken; have != want {
			t.Fatalf("retrieval %d: declarations token: have: %v, want: %v", i, have, want)
		}
	}
}
