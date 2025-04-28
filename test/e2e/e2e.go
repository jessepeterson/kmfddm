package e2e

import (
	"context"
	"encoding/json"
	"io"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/alexedwards/flow"
	"github.com/jessepeterson/kmfddm/http/api"
	"github.com/micromdm/nanolib/log"
)

const testID1 = "golang_test_decl_A711884F1270"
const testType1 = "com.apple.configuration.management.test"
const testDecl1 = `{
    "Type": "` + testType1 + `",
    "Payload": {
        "Echo": "test"
    },
    "Identifier": "` + testID1 + `"
}`

type TestDeclaration struct {
	ServerToken string
	Type        string
	Identifier  string
	Payload     struct {
		Echo string
	}
}

type nopNotifier struct{}

func (*nopNotifier) Changed(ctx context.Context, declarations []string, sets []string, ids []string) error {
	return nil
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

func TestE2E(t *testing.T, ctx context.Context, storage api.APIStorage) {
	// setup
	mux := flow.New()
	n := &nopNotifier{}
	api.HandleAPIv1("/v1", mux, log.NopLogger, storage, n)

	// attempt to delete the not yet uploaded declaration
	resp := doReq(mux, "DELETE", "/v1/declarations/"+testID1, nil)
	expectHTTP(t, resp, 304)

	// retrieve the (not yet uploaded) declaration
	resp = doReq(mux, "GET", "/v1/declarations/"+testID1, nil)
	expectHTTP(t, resp, 404)

	// upload our declaration
	resp = doReq(mux, "PUT", "/v1/declarations", []byte(testDecl1))
	expectHTTP(t, resp, 204)

	// upload our declaration again
	resp = doReq(mux, "PUT", "/v1/declarations", []byte(testDecl1))
	// status should be 304 for an identically uploaded declaration
	expectHTTP(t, resp, 304)

	// retrieve the declaration
	resp = doReq(mux, "GET", "/v1/declarations/"+testID1, nil)
	expectHTTP(t, resp, 200)

	// decode the retrieved declaration
	rTestD := &TestDeclaration{}
	if err := json.NewDecoder(resp.Body).Decode(rTestD); err != nil {
		t.Fatal(err)
	}

	if rTestD.ServerToken == "" {
		t.Error("ServerToken empty")
	} else {
		// set the server token to empty because we don't know what a
		// given backend's server token may look like. as long as its
		// not empty (checked above), should be fine for the following
		// tests.
		rTestD.ServerToken = ""
	}

	// setup our expected struct
	eTestD := &TestDeclaration{
		Identifier: testID1,
		Type:       testType1,
		Payload: struct{ Echo string }{
			"test",
		},
	}

	if !reflect.DeepEqual(eTestD, rTestD) {
		t.Errorf("have: %v, want: %v", rTestD, eTestD)
	}

	// delete the (not yet) set association
	resp = doReq(mux, "DELETE", "/v1/set-declarations/golang_test_set_854CC771FACE?declaration="+testID1, nil)
	expectHTTP(t, resp, 304)

	// retrieve the declarations for this (not yet existing) set
	resp = doReq(mux, "GET", "/v1/set-declarations/golang_test_set_854CC771FACE", nil)
	expectHTTP(t, resp, 200)

	// check that we have no association in the result
	bodyBytes, _ := io.ReadAll(resp.Body)
	if have, want := strings.TrimSpace(string(bodyBytes)), "null"; have != want {
		t.Errorf("have: %v, want: %v", have, want)
	}

	// associate the declaration with the set
	resp = doReq(mux, "PUT", "/v1/set-declarations/golang_test_set_854CC771FACE?declaration="+testID1, nil)
	expectHTTP(t, resp, 204)

	// retrieve the declarations for this (not yet existing) set
	resp = doReq(mux, "GET", "/v1/set-declarations/golang_test_set_854CC771FACE", nil)
	expectHTTP(t, resp, 200)

	// check that we have the proper association
	var s *[]string
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		t.Fatal(err)
	}
	if s == nil {
		t.Fatal("nil result")
	}
	if have, want := *s, []string{"golang_test_decl_A711884F1270"}; !stringSlicesEqual(have, want) {
		t.Errorf("have: %v, want: %v", have, want)
	}

	// remove the association
	resp = doReq(mux, "DELETE", "/v1/set-declarations/golang_test_set_854CC771FACE?declaration="+testID1, nil)
	expectHTTP(t, resp, 204)

	// remove the association again
	resp = doReq(mux, "DELETE", "/v1/set-declarations/golang_test_set_854CC771FACE?declaration="+testID1, nil)
	expectHTTP(t, resp, 304)

	// delete the declaration
	resp = doReq(mux, "DELETE", "/v1/declarations/"+testID1, nil)
	expectHTTP(t, resp, 204)

	// attempt to delete the declaration again
	resp = doReq(mux, "DELETE", "/v1/declarations/"+testID1, nil)
	expectHTTP(t, resp, 304)
}
