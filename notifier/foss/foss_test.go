package foss

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// TestEnqueueChunkedBodies checks that every chunked enqueue request
// carries the full raw command, and not just the first one.
func TestEnqueueChunkedBodies(t *testing.T) {
	rawCommand := []byte(`{"command":"test"}`)

	var mu sync.Mutex
	var bodies [][]byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			return
		}
		mu.Lock()
		bodies = append(bodies, body)
		mu.Unlock()
	}))
	defer srv.Close()

	m, err := NewFossMDM(srv.URL, "apiKey")
	if err != nil {
		t.Fatal(err)
	}

	// enough ids for more than one chunk
	var ids []string
	for i := 0; i < m.max*2+1; i++ {
		ids = append(ids, fmt.Sprintf("id%d", i))
	}
	wantChunks := len(chunk(ids, m.max))

	if err = m.Enqueue(context.Background(), ids, rawCommand); err != nil {
		t.Fatal(err)
	}

	if have, want := len(bodies), wantChunks; have != want {
		t.Fatalf("request count: have %d, want %d", have, want)
	}
	for i, body := range bodies {
		if have, want := len(body), len(rawCommand); have != want {
			t.Errorf("request %d: body length: have %d, want %d", i, have, want)
		}
	}
}
