package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type ServeHTTP interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

func doReq(serve ServeHTTP, method, target string, body []byte) *http.Response {
	var buf io.Reader
	if len(body) > 0 {
		buf = bytes.NewBuffer(body)
	}
	req := httptest.NewRequest(method, target, buf)
	w := httptest.NewRecorder()
	serve.ServeHTTP(w, req)
	return w.Result()
}

func expectHTTP(t *testing.T, resp *http.Response, statusCode int) {
	t.Helper()

	if resp == nil {
		t.Fatal("nil response")
	}

	if have, want := resp.StatusCode, statusCode; statusCode > 0 && have != want {
		t.Errorf("status code: have: %v, want: %v", have, want)
	}
}

func expectHTTPStringSlice(t *testing.T, resp *http.Response, statusCode int, expected []string) {
	t.Helper()

	expectHTTP(t, resp, statusCode)

	if header := resp.Header.Get("Content-Type"); !strings.Contains(header, "application/json") {
		t.Errorf("header does not contain application/json: %s", header)
	}

	var s []string
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		t.Fatal(err)
	}
	if have, want := s, expected; !stringSlicesEqual(have, want) {
		t.Errorf("have: %v, want: %v", have, want)
	}
}
