package notifier

import (
	"bytes"
	"context"
	"reflect"
	"testing"
)

type testEnqueuer struct {
	lastIDs    []string
	lastTokens []byte
	noMulti    bool
}

func (e *testEnqueuer) EnqueueDMCommand(ctx context.Context, ids []string, tokensJSON []byte) error {
	e.lastIDs = ids
	e.lastTokens = tokensJSON
	return nil
}

func (e *testEnqueuer) SupportsMultiCommands() bool {
	return !e.noMulti
}

type testStore struct {
	tokens []byte
}

func (s *testStore) RetrieveTokensJSON(ctx context.Context, enrollmentID string) ([]byte, error) {
	return s.tokens, nil
}

func (s *testStore) RetrieveEnrollmentIDs(ctx context.Context, declarations []string, sets []string, ids []string) ([]string, error) {
	return ids, nil
}

func TestNotifier(t *testing.T) {
	e := new(testEnqueuer)
	s := &testStore{tokens: []byte("hello")}
	n, err := New(e, s)
	if err != nil {
		t.Fatal(err)
	}
	err = n.Changed(context.Background(), nil, nil, []string{"id"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual([]string{"id"}, e.lastIDs) {
		t.Error("not deep equal")
	}
	if !reflect.DeepEqual([]byte("hello"), e.lastTokens) {
		t.Error("not deep equal")
	}
	err = n.Changed(context.Background(), nil, nil, []string{"id1", "id2"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual([]string{"id1", "id2"}, e.lastIDs) {
		t.Error("not deep equal")
	}
	if len(e.lastTokens) > 0 {
		t.Errorf("tokens should not be present: %s", e.lastTokens)
	}

	// set NO multi-targeted commands to true (i.e. disable multi-target)
	e.noMulti = true
	// resend a notifier
	err = n.Changed(context.Background(), nil, nil, []string{"id1", "id2"})
	if err != nil {
		t.Fatal(err)
	}
	// check the *last* ID to make sure its the only one queued individually
	if !reflect.DeepEqual([]string{"id2"}, e.lastIDs) {
		t.Error("not deep equal")
	}
	// because the tokens are sent individually now (no multi)
	// we should see some tokens
	if !bytes.Equal(e.lastTokens, []byte("hello")) {
		t.Errorf("tokens should be present and equal: %s", e.lastTokens)
	}

}
