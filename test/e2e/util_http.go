package e2e

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func expectHTTP(t *testing.T, resp *http.Response, statusCode int) {
	t.Helper()

	if resp == nil {
		t.Fatal("nil response")
	}

	if have, want := resp.StatusCode, statusCode; statusCode > 0 && have != want {
		t.Errorf("status code: have: %v, want: %v", have, want)
	}
}

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
