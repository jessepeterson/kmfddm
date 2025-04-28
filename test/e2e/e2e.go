package e2e

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/alexedwards/flow"
	"github.com/jessepeterson/kmfddm/http/api"
	"github.com/micromdm/nanolib/log"
)

const testID1 = "golang_test_D526087C-3B18-4782-99FA-A711884F1270"
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

func TestE2E(t *testing.T, ctx context.Context, storage api.APIStorage) {
	// setup
	mux := flow.New()
	n := &nopNotifier{}
	api.HandleAPIv1("/v1", mux, log.NopLogger, storage, n)

	// attempt to delete the not yet uploaded declaration
	resp := doReq(mux, "DELETE", "/v1/declarations/"+testID1, nil)
	expectHTTP(t, resp, 304)

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

}
