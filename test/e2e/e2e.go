package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/alexedwards/flow"
	"github.com/jessepeterson/kmfddm/ddm"
	"github.com/jessepeterson/kmfddm/http/api"
	httpddm "github.com/jessepeterson/kmfddm/http/ddm"
	"github.com/jessepeterson/kmfddm/storage"
	"github.com/micromdm/nanolib/http/trace"
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
	DDMStorage
	storage.StatusStorer
}

var emptyDI = &ddm.DeclarationItems{
	Declarations: ddm.ManifestDeclarationItems{
		Activations:    []ddm.ManifestDeclaration{},
		Configurations: []ddm.ManifestDeclaration{},
		Assets:         []ddm.ManifestDeclaration{},
		Management:     []ddm.ManifestDeclaration{},
	},
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

func TestE2E(t *testing.T, _ context.Context, storage TestStorage) {
	// setup
	flowMux := flow.New()
	n := &captureNotifier{store: storage}
	logger := log.NopLogger
	api.HandleAPIv1("/v1", flowMux, logger, storage, n)
	handleDDM(flowMux, logger, storage)

	var mux http.Handler = flowMux
	mux = trace.NewTraceLoggingHandler(mux, logger.With("handler", "log"), func(*http.Request) string { return "go_test_trace_id" })

	t.Run("declaration-setup", func(t *testing.T) {

		// attempt to delete the not yet uploaded declaration
		resp := doReq(mux, "DELETE", "/v1/declarations/"+testID1, nil)
		expectHTTP(t, resp, 304)
		expectNotifierSlice(t, n, false, nil)

		// retrieve the (not yet uploaded) declaration
		resp = doReq(mux, "GET", "/v1/declarations/"+testID1, nil)
		expectHTTP(t, resp, 404)

		// retreive list of declarations (should be empty)
		resp = doReq(mux, "GET", "/v1/declarations", nil)
		expectHTTPStringSlice(t, resp, 200, nil)

		// upload our declaration
		resp = doReq(mux, "PUT", "/v1/declarations", []byte(testDecl1))
		expectHTTP(t, resp, 204)
		expectNotifierSlice(t, n, true, nil)

		// upload our declaration again
		resp = doReq(mux, "PUT", "/v1/declarations", []byte(testDecl1))
		// status should be 304 for an identically uploaded declaration
		expectHTTP(t, resp, 304)
		expectNotifierSlice(t, n, false, nil)

		// retreive list of declarations
		resp = doReq(mux, "GET", "/v1/declarations", nil)
		expectHTTPStringSlice(t, resp, 200, []string{"golang_test_decl_A711884F1270"})

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
		}

		// setup our expected struct
		eTestD := &TestDeclaration{
			Identifier: testID1,
			Type:       testType1,
			Payload: struct{ Echo string }{
				"test",
			},
			ServerToken: rTestD.ServerToken,
		}

		if !reflect.DeepEqual(eTestD, rTestD) {
			t.Errorf("declarations not equal: have: %v, want: %v", rTestD, eTestD)
		}

		// touch declaration
		resp = doReq(mux, "POST", "/v1/declarations/"+testID1+"/touch", nil)
		expectHTTP(t, resp, 204)
		expectNotifierSlice(t, n, true, nil)

		// retrieve the newly touched declaration
		resp = doReq(mux, "GET", "/v1/declarations/"+testID1, nil)
		expectHTTP(t, resp, 200)

		// decode the retrieved declaration
		rTestD = &TestDeclaration{}
		if err := json.NewDecoder(resp.Body).Decode(rTestD); err != nil {
			t.Fatal(err)
		}

		// make sure token has changed
		if have, notWant := rTestD.ServerToken, eTestD.ServerToken; have == notWant {
			t.Errorf("token should have changed: have: %v, notWant: %v", have, notWant)
		}

		// make the same for comparison otherwise
		eTestD.ServerToken = rTestD.ServerToken

		if !reflect.DeepEqual(rTestD, eTestD) {
			t.Errorf("declarations not equal: have: %v, want: %v", rTestD, eTestD)
		}
	})

	t.Run("set-declaration-setup", func(t *testing.T) {
		// delete the (not yet) set association
		resp := doReq(mux, "DELETE", "/v1/set-declarations/golang_test_set_854CC771FACE?declaration="+testID1, nil)
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

		// retreive list of declarations (should still be present)
		resp = doReq(mux, "GET", "/v1/declarations", nil)
		expectHTTPStringSlice(t, resp, 200, []string{"golang_test_decl_A711884F1270"})

		// retrieve the declarations for this (not yet existing) set
		resp = doReq(mux, "GET", "/v1/set-declarations/golang_test_set_854CC771FACE", nil)
		expectHTTPStringSlice(t, resp, 200, []string{"golang_test_decl_A711884F1270"})
	})

	t.Run("enrollment-set-setup", func(t *testing.T) {
		// first delete the enrollment association to make sure no change
		resp := doReq(mux, "DELETE", "/v1/enrollment-sets/golang_test_enr_775871FF5E47?set=golang_test_set_854CC771FACE", nil)
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
	})

	enrHdr := make(http.Header)
	enrHdr.Add(httpddm.EnrollmentIDHeader, "golang_test_enr_775871FF5E47")

	t.Run("ddm", func(t *testing.T) {
		// at this point we have a declaration associated to a set
		// which is associated to an enrollment.
		// the full Tokens, Items, Declarations should work now

		resp := doReqHeader(mux, "GET", "/declaration-items", enrHdr, nil)

		diT := &ddm.DeclarationItems{
			Declarations: ddm.ManifestDeclarationItems{
				Activations: []ddm.ManifestDeclaration{},
				Configurations: []ddm.ManifestDeclaration{
					{Identifier: testID1},
				},
				Assets:     []ddm.ManifestDeclaration{},
				Management: []ddm.ManifestDeclaration{},
			},
		}

		expectHTTPDI(t, resp, 200, diT)

		// attempt to retrieve the wrong type
		resp = doReqHeader(mux, "GET", "/declaration/INVALID/"+testID1, enrHdr, nil)
		expectHTTP(t, resp, 404)

		// attempt to retrieve the correct type, but wrong declaration
		resp = doReqHeader(mux, "GET", "/declaration/configuration/INVALID", enrHdr, nil)
		expectHTTP(t, resp, 404)

		// attempt to retrieve the correct declaration
		resp = doReqHeader(mux, "GET", "/declaration/configuration/"+testID1, enrHdr, nil)
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

		// touch declaration now that we have a set and declaration association
		resp = doReq(mux, "POST", "/v1/declarations/"+testID1+"/touch", nil)
		expectHTTP(t, resp, 204)
		// should now be notified of our enrollment id
		expectNotifierSlice(t, n, true, []string{"golang_test_enr_775871FF5E47"})

		// remove all sets for id.
		resp = doReq(mux, "DELETE", "/v1/enrollment-sets-all/sets/golang_test_enr_775871FF5E47", nil)
		expectHTTP(t, resp, 204)
		expectNotifierSlice(t, n, true, []string{"golang_test_enr_775871FF5E47"})

		// retrieve sets for enrollment (should be none)
		resp = doReq(mux, "GET", "/v1/enrollment-sets/golang_test_enr_775871FF5E47", nil)
		expectHTTPStringSlice(t, resp, 200, nil)

		// re-associate for the below tests
		resp = doReq(mux, "PUT", "/v1/enrollment-sets/golang_test_enr_775871FF5E47?set=golang_test_set_854CC771FACE", nil)
		expectHTTP(t, resp, 204)
		expectNotifierSlice(t, n, true, []string{"golang_test_enr_775871FF5E47"})
	})

	t.Run("enrollment-set-teardown", func(t *testing.T) {
		// remove the association
		resp := doReq(mux, "DELETE", "/v1/enrollment-sets/golang_test_enr_775871FF5E47?set=golang_test_set_854CC771FACE", nil)
		expectHTTP(t, resp, 204)
		expectNotifierSlice(t, n, true, []string{"golang_test_enr_775871FF5E47"})

		// retrieve sets for enrollment (should be none)
		resp = doReq(mux, "GET", "/v1/enrollment-sets/golang_test_enr_775871FF5E47", nil)
		expectHTTPStringSlice(t, resp, 200, nil)

		// remove the association again
		resp = doReq(mux, "DELETE", "/v1/enrollment-sets/golang_test_enr_775871FF5E47?set=golang_test_set_854CC771FACE", nil)
		expectHTTP(t, resp, 304)
		expectNotifierSlice(t, n, false, nil)

		resp = doReqHeader(mux, "GET", "/declaration-items", enrHdr, nil)
		expectHTTPDI(t, resp, 200, emptyDI)
	})

	t.Run("set-declaration-teardown", func(t *testing.T) {

		// remove the association
		resp := doReq(mux, "DELETE", "/v1/set-declarations/golang_test_set_854CC771FACE?declaration="+testID1, nil)
		expectHTTP(t, resp, 204)
		expectNotifierSlice(t, n, true, nil)

		// retrieve the declarations for this (not yet existing) set
		resp = doReq(mux, "GET", "/v1/set-declarations/golang_test_set_854CC771FACE", nil)
		expectHTTPStringSlice(t, resp, 200, nil)

		// remove the association again
		resp = doReq(mux, "DELETE", "/v1/set-declarations/golang_test_set_854CC771FACE?declaration="+testID1, nil)
		expectHTTP(t, resp, 304)
		expectNotifierSlice(t, n, false, nil)
	})

	t.Run("declaration-teardown", func(t *testing.T) {
		// delete the declaration
		resp := doReq(mux, "DELETE", "/v1/declarations/"+testID1, nil)
		expectHTTP(t, resp, 204)
		// shouldn't notify here because the storage should know that
		// we are not currently associated with any sets
		expectNotifierSlice(t, n, false, nil)

		// retreive list of declarations (should still be empty)
		resp = doReq(mux, "GET", "/v1/declarations", nil)
		expectHTTPStringSlice(t, resp, 200, nil)

		// attempt to delete the declaration again
		resp = doReq(mux, "DELETE", "/v1/declarations/"+testID1, nil)
		expectHTTP(t, resp, 304)
		expectNotifierSlice(t, n, false, nil)
	})

	t.Run("status", func(t *testing.T) {
		testStatus(t, mux, n)
	})
}
