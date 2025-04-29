package e2e

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/alexedwards/flow"
	"github.com/jessepeterson/kmfddm/http/api"
	"github.com/jessepeterson/kmfddm/storage"
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

type TestStorage interface {
	api.APIStorage
	storage.EnrollmentIDRetriever
}

func expectNotifierSlice(t *testing.T, n *captureNotifier, shouldHaveRun bool, want []string) {
	t.Helper()

	if shouldHaveRun && !n.called {
		t.Error("notifier should have been called but was not")
	} else if !shouldHaveRun && n.called {
		t.Error("notifier should not have been called but was")
	}

	have := n.getAndClear()

	if !stringSlicesEqual(have, want) {
		t.Errorf("have: %v, want: %v", have, want)
	}
}

func TestE2E(t *testing.T, ctx context.Context, storage TestStorage) {
	// setup
	mux := flow.New()
	n := &captureNotifier{store: storage}
	api.HandleAPIv1("/v1", mux, log.NopLogger, storage, n)

	// attempt to delete the not yet uploaded declaration
	resp := doReq(mux, "DELETE", "/v1/declarations/"+testID1, nil)
	expectHTTP(t, resp, 304)
	expectNotifierSlice(t, n, false, nil)

	// retrieve the (not yet uploaded) declaration
	resp = doReq(mux, "GET", "/v1/declarations/"+testID1, nil)
	expectHTTP(t, resp, 404)

	// upload our declaration
	resp = doReq(mux, "PUT", "/v1/declarations", []byte(testDecl1))
	expectHTTP(t, resp, 204)
	expectNotifierSlice(t, n, true, nil)

	// upload our declaration again
	resp = doReq(mux, "PUT", "/v1/declarations", []byte(testDecl1))
	// status should be 304 for an identically uploaded declaration
	expectHTTP(t, resp, 304)
	expectNotifierSlice(t, n, false, nil)

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
	expectNotifierSlice(t, n, false, nil)

	// retrieve the declarations for this (not yet existing) set
	resp = doReq(mux, "GET", "/v1/set-declarations/golang_test_set_854CC771FACE", nil)
	expectHTTPStringSlice(t, resp, 200, nil)

	// associate the declaration with the set
	resp = doReq(mux, "PUT", "/v1/set-declarations/golang_test_set_854CC771FACE?declaration="+testID1, nil)
	expectHTTP(t, resp, 204)
	expectNotifierSlice(t, n, true, nil)

	// attempt deletion of the declaration (should fail after set assoc, above)
	resp = doReq(mux, "DELETE", "/v1/declarations/"+testID1, nil)
	expectHTTP(t, resp, 500)
	expectNotifierSlice(t, n, false, nil)

	// retrieve the declarations for this (not yet existing) set
	resp = doReq(mux, "GET", "/v1/set-declarations/golang_test_set_854CC771FACE", nil)
	expectHTTPStringSlice(t, resp, 200, []string{"golang_test_decl_A711884F1270"})

	// first delete the enrollment association to make sure no change
	resp = doReq(mux, "DELETE", "/v1/enrollment-sets/golang_test_enr_775871FF5E47?set=golang_test_set_854CC771FACE", nil)
	expectHTTP(t, resp, 304)
	expectNotifierSlice(t, n, false, nil)

	// get the associations to make sure
	resp = doReq(mux, "GET", "/v1/enrollment-sets/golang_test_enr_775871FF5E47", nil)
	expectHTTPStringSlice(t, resp, 200, nil)

	// associate the set with the enrollment
	resp = doReq(mux, "PUT", "/v1/enrollment-sets/golang_test_enr_775871FF5E47?set=golang_test_set_854CC771FACE", nil)
	expectHTTP(t, resp, 204)
	expectNotifierSlice(t, n, true, []string{"golang_test_enr_775871FF5E47"})

	// retrieve the sets for this enrollment
	resp = doReq(mux, "GET", "/v1/enrollment-sets/golang_test_enr_775871FF5E47", nil)
	expectHTTPStringSlice(t, resp, 200, []string{"golang_test_set_854CC771FACE"})

	// remove the association
	resp = doReq(mux, "DELETE", "/v1/enrollment-sets/golang_test_enr_775871FF5E47?set=golang_test_set_854CC771FACE", nil)
	expectHTTP(t, resp, 204)
	expectNotifierSlice(t, n, true, []string{"golang_test_enr_775871FF5E47"})

	// first delete the association to make sure no change
	resp = doReq(mux, "GET", "/v1/enrollment-sets/golang_test_enr_775871FF5E47", nil)
	expectHTTPStringSlice(t, resp, 200, nil)

	// remove the association again
	resp = doReq(mux, "DELETE", "/v1/enrollment-sets/golang_test_enr_775871FF5E47?set=golang_test_set_854CC771FACE", nil)
	expectHTTP(t, resp, 304)
	expectNotifierSlice(t, n, false, nil)

	// remove the association
	resp = doReq(mux, "DELETE", "/v1/set-declarations/golang_test_set_854CC771FACE?declaration="+testID1, nil)
	expectHTTP(t, resp, 204)
	expectNotifierSlice(t, n, true, nil)

	// retrieve the declarations for this (not yet existing) set
	resp = doReq(mux, "GET", "/v1/set-declarations/golang_test_set_854CC771FACE", nil)
	expectHTTPStringSlice(t, resp, 200, nil)

	// remove the association again
	resp = doReq(mux, "DELETE", "/v1/set-declarations/golang_test_set_854CC771FACE?declaration="+testID1, nil)
	expectHTTP(t, resp, 304)
	expectNotifierSlice(t, n, false, nil)

	// delete the declaration
	resp = doReq(mux, "DELETE", "/v1/declarations/"+testID1, nil)
	expectHTTP(t, resp, 204)
	// shouldn't notify here because the storage should know that
	// we are not currently associated with any sets
	expectNotifierSlice(t, n, false, nil)

	// attempt to delete the declaration again
	resp = doReq(mux, "DELETE", "/v1/declarations/"+testID1, nil)
	expectHTTP(t, resp, 304)
	expectNotifierSlice(t, n, false, nil)
}
