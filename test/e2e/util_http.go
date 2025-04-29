package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/jessepeterson/kmfddm/http/api"
	ddmhttp "github.com/jessepeterson/kmfddm/http/ddm"
	"github.com/jessepeterson/kmfddm/logkeys"
	"github.com/jessepeterson/kmfddm/storage"

	"github.com/micromdm/nanolib/log"
)

type DDMStorage interface {
	storage.TokensDeclarationItemsStorage
	storage.DeclarationJSONRetriever
}

func handleDDM(mux api.Mux, logger log.Logger, storage DDMStorage) {
	mux.Handle(
		"/declaration-items",
		ddmhttp.TokensOrDeclarationItemsHandler(storage, false, logger.With(logkeys.Handler, "declaration-items")),
		"GET",
	)

	mux.Handle(
		"/tokens",
		ddmhttp.TokensOrDeclarationItemsHandler(storage, true, logger.With(logkeys.Handler, "tokens")),
		"GET",
	)

	mux.Handle(
		"/declaration/:type/:id",
		http.StripPrefix("/declaration/",
			ddmhttp.DeclarationHandler(storage, logger.With(logkeys.Handler, "declaration")),
		),
		"GET",
	)
}

type ServeHTTP interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

func doReqHeader(serve ServeHTTP, method, target string, headers http.Header, body []byte) *http.Response {
	var buf io.Reader
	if len(body) > 0 {
		buf = bytes.NewBuffer(body)
	}
	req := httptest.NewRequest(method, target, buf)
	for k, vs := range headers {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	w := httptest.NewRecorder()
	serve.ServeHTTP(w, req)
	return w.Result()
}

func doReq(serve ServeHTTP, method, target string, body []byte) *http.Response {
	return doReqHeader(serve, method, target, nil, body)
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

// stringSlicesEqual checks if two string slices are equal by sorting them.
func stringSlicesEqual(expected, actual []string) bool {
	if len(expected) != len(actual) {
		return false
	}

	// Sort the string slices
	sort.Strings(expected)
	sort.Strings(actual)

	// Compare the sorted slices
	for i := range expected {
		if expected[i] != actual[i] {
			return false
		}
	}

	return true
}
