package shard

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jessepeterson/kmfddm/ddm"
)

func TestShard(t *testing.T) {
	ctx := context.Background()

	s := NewShardStorage()

	j, err := s.RetrieveEnrollmentDeclarationJSON(ctx, DeclarationIdentifier, ManifestType, "baz")
	if err != nil {
		t.Fatal(err)
	}

	d, err := ddm.ParseDeclaration(j)
	if err != nil {
		t.Fatal(err)
	}

	// make sure the identifier and type are what they should be
	if have, want := d.Identifier, DeclarationIdentifier; have != want {
		t.Errorf("declaration identifier: have=%v, want=%v", have, want)
	}
	if have, want := d.Type, DeclarationType; have != want {
		t.Errorf("declaration identifier: have=%v, want=%v", have, want)
	}

	type shardPayload struct {
		Shard int `json:"shard"`
	}

	p := &shardPayload{}

	if err = json.Unmarshal(d.PayloadJSON, p); err != nil {
		t.Fatal(err)
	}

	if p.Shard < 0 || p.Shard > 100 {
		t.Error("invalid shard value")
	}

	decls, err := s.RetrieveDeclarationItems(ctx, "baz")
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(decls), 1; have != want {
		t.Fatalf("declaration item len: have=%v, want=%v", have, want)
	}

	manifestDecl := decls[0]

	// make sure our identifiers and server tokens match between
	// declaration items and declaration.
	if have, want := manifestDecl.Identifier, d.Identifier; have != want {
		t.Errorf("declaration identifier: have=%v, want=%v", have, want)
	}
	if have, want := manifestDecl.ServerToken, d.ServerToken; have != want {
		t.Errorf("declaration identifier: have=%v, want=%v", have, want)
	}
}
