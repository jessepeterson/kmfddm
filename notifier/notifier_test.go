package notifier

import (
	"context"
	"reflect"
	"testing"
)

type testEnqueuer struct {
	lastIDs    []string
	lastTokens []byte
}

func (e *testEnqueuer) EnqueueDMCommand(ctx context.Context, ids []string, tokensJSON []byte) error {
	e.lastIDs = ids
	e.lastTokens = tokensJSON
	return nil
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
		t.Error("tokens should not be present")
	}
}
