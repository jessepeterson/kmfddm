package foss

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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

func TestConcatURL(t *testing.T) {
	for _, tc := range []struct {
		base string
		ids  []string
		want string // empty if an error is expected
	}{
		{"http://mdm.example.com/v1/enqueue/", []string{"ABC-123"}, "http://mdm.example.com/v1/enqueue/ABC-123"},
		{"http://mdm.example.com/v1/enqueue/", []string{"00008030-001A2B3C4D5E6F70"}, "http://mdm.example.com/v1/enqueue/00008030-001A2B3C4D5E6F70"},
		{"http://mdm.example.com/v1/enqueue/", []string{"A", "B", "C"}, "http://mdm.example.com/v1/enqueue/A,B,C"},
		// user-channel IDs contain a colon and must not be treated as a scheme
		{"http://mdm.example.com/v1/enqueue/", []string{"00008030-001A2B3C:2F1E-USER"}, "http://mdm.example.com/v1/enqueue/00008030-001A2B3C:2F1E-USER"},
		{"http://mdm.example.com/v1/enqueue/", []string{"A1B2C3D4E5:4F3A-USER"}, "http://mdm.example.com/v1/enqueue/A1B2C3D4E5:4F3A-USER"},
		{"http://mdm.example.com/v1/enqueue/", []string{"A", "B1:U"}, "http://mdm.example.com/v1/enqueue/A,B1:U"},
		// the ids are unescaped: other characters are escaped, not interpreted
		{"http://mdm.example.com/v1/enqueue/", []string{"x?y=1"}, "http://mdm.example.com/v1/enqueue/x%3Fy=1"},
		{"http://mdm.example.com/v1/enqueue/", []string{"x#f"}, "http://mdm.example.com/v1/enqueue/x%23f"},
		{"http://mdm.example.com/v1/enqueue/", []string{"a b"}, "http://mdm.example.com/v1/enqueue/a%20b"},
		{"http://mdm.example.com/v1/enqueue/", []string{"50%off"}, "http://mdm.example.com/v1/enqueue/50%25off"},
		{"http://mdm.example.com/v1/enqueue/", []string{"..%2f..%2fx"}, "http://mdm.example.com/v1/enqueue/..%252f..%252fx"},
		{"http://mdm.example.com/v1/enqueue/", []string{"%2e%2e"}, "http://mdm.example.com/v1/enqueue/%252e%252e"},
		{"http://mdm.example.com/v1/enqueue/", []string{"http:x"}, "http://mdm.example.com/v1/enqueue/http:x"},
		// URL-looking ids are rejected
		{"http://mdm.example.com/v1/enqueue/", []string{"http://observer/leak"}, ""},
		{"http://mdm.example.com/v1/enqueue/", []string{"//observer/leak"}, ""},
		{"http://mdm.example.com/v1/enqueue/", []string{"//user@observer/leak"}, ""},
		{"http://mdm.example.com/v1/enqueue/", []string{"A", "http://observer/leak"}, ""},
		// empty, ".", "..", and ids containing "/" or "," are rejected
		{"http://mdm.example.com/v1/enqueue/", nil, ""},
		{"http://mdm.example.com/v1/enqueue/", []string{""}, ""},
		{"http://mdm.example.com/v1/enqueue/", []string{"A", ""}, ""},
		{"http://mdm.example.com/v1/enqueue/", []string{"."}, ""},
		{"http://mdm.example.com/v1/enqueue/", []string{".."}, ""},
		{"http://mdm.example.com/v1/enqueue/", []string{"../../v1/pushcert"}, ""},
		{"http://mdm.example.com/v1/enqueue/", []string{"/abs"}, ""},
		{"http://mdm.example.com/v1/enqueue/", []string{"a/b"}, ""},
		{"http://mdm.example.com/v1/enqueue/", []string{"A,B"}, ""},
		// the base query and fragment are dropped
		{"http://mdm.example.com/v1/enqueue/?k=v#f", []string{"A"}, "http://mdm.example.com/v1/enqueue/A"},
		// the base userinfo and escaped path are kept
		{"http://u:p@mdm.example.com/v%201/enqueue/", []string{"A"}, "http://u:p@mdm.example.com/v%201/enqueue/A"},
		// without a trailing slash the last path element is replaced
		{"http://mdm.example.com/v1/enqueue", []string{"A"}, "http://mdm.example.com/v1/A"},
		{"http://mdm.example.com", []string{"A"}, "http://mdm.example.com/A"},
	} {
		base, err := url.Parse(tc.base)
		if err != nil {
			t.Fatal(err)
		}
		have, err := concatURL(base, tc.ids)
		if tc.want == "" {
			if err == nil {
				t.Errorf("%s %q: expected error, have: %q", tc.base, tc.ids, have)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s %q: %v", tc.base, tc.ids, err)
		} else if have != tc.want {
			t.Errorf("%s %q: URL mismatch: have: %q, want: %q", tc.base, tc.ids, have, tc.want)
		}
	}
}

// recordingDoer records the URLs of requests it receives.
type recordingDoer struct {
	urls []string
}

func (d *recordingDoer) Do(r *http.Request) (*http.Response, error) {
	d.urls = append(d.urls, r.URL.String())
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
}

// TestInvalidIDsNotSent checks that requests for invalid enrollment IDs
// are never sent (with the MDM server credentials) to any URL.
func TestInvalidIDsNotSent(t *testing.T) {
	m, err := NewFossMDM("https://mdm.example.com/v1/enqueue", "apiKey", WithPush("https://mdm.example.com/v1/push"))
	if err != nil {
		t.Fatal(err)
	}
	d := &recordingDoer{}
	m.client = d

	for _, id := range []string{
		"http://observer/leak",
		"//observer/leak",
		"../../v1/pushcert",
		"..",
		"",
	} {
		if err = m.Enqueue(context.Background(), []string{id}, []byte("cmd")); err != nil {
			t.Fatal(err)
		}
		if err = m.Push(context.Background(), []string{id}); err != nil {
			t.Fatal(err)
		}
	}
	if len(d.urls) > 0 {
		t.Errorf("requests should not have been sent: %q", d.urls)
	}

	// sanity check that valid ids are still sent to the configured URLs
	if err = m.Enqueue(context.Background(), []string{"A", "B:U"}, []byte("cmd")); err != nil {
		t.Fatal(err)
	}
	if err = m.Push(context.Background(), []string{"A"}); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"https://mdm.example.com/v1/enqueue/A,B:U",
		"https://mdm.example.com/v1/push/A",
	}
	if fmt.Sprint(d.urls) != fmt.Sprint(want) {
		t.Errorf("URL mismatch: have: %q, want: %q", d.urls, want)
	}
}
