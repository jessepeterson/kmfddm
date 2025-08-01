package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/jessepeterson/kmfddm/ddm"
	httpddm "github.com/jessepeterson/kmfddm/http/ddm"
	"github.com/jessepeterson/kmfddm/storage"
)

const testDecl2 = `{
	"Type": "com.apple.configuration.management.test",
	"Payload": {
		"Echo": "KMFDDM!"
	},
	"Identifier": "com.example.test"
}`

type errorJSONS struct {
	Path      string          `json:"path"`
	Error     json.RawMessage `json:"error"`
	Timestamp time.Time       `json:"timestamp"`
	StatusID  string          `json:"status_id,omitempty"`
}

func testStatus(t *testing.T, mux http.Handler, n *captureNotifier) {
	enrHdr := make(http.Header)

	// submit status for golang_test_enr_87C029C236E0

	statusBytes, err := os.ReadFile("../../test/e2e/testdata/status.1st.json")
	if err != nil {
		t.Fatal(err)
	}

	enrHdr.Set(httpddm.EnrollmentIDHeader, "golang_test_enr_87C029C236E0")

	resp := doReqHeader(mux, "PUT", "/status", enrHdr, statusBytes)
	expectHTTP(t, resp, 200)

	// test declaration status values

	resp = doReq(mux, "GET", "/v1/status-values/golang_test_enr_87C029C236E0", nil)
	expectHTTP(t, resp, 200)

	s := make(map[string][]storage.StatusValue)
	err = json.NewDecoder(resp.Body).Decode(&s)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := s["golang_test_enr_87C029C236E0"]; !ok {
		t.Fatal("id not present")
	}

	values := s["golang_test_enr_87C029C236E0"]

	// status value data from "testdata/status.1st.json"
	eValues := []storage.StatusValue{
		{Path: ".StatusItems.device.identifier.serial-number", Value: "ZRMXJQTTFX", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.device.identifier.udid", Value: "0000FE00-5FADE97DCBBBEDA5", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.device.model.family", Value: "Mac", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.device.operating-system.build-version", Value: "22E261", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.device.operating-system.family", Value: "macOS", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.device.operating-system.supplemental.extra-version", Value: "(a)", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.device.operating-system.version", Value: "13.3.1", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.declarations.activations", Value: "com.apple.activation.simple", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.declarations.configurations", Value: "com.apple.configuration.passcode.settings", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.declarations.configurations", Value: "com.apple.configuration.legacy", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.declarations.configurations", Value: "com.apple.configuration.management.status-subscriptions", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.declarations.configurations", Value: "com.apple.configuration.management.test", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.declarations.configurations", Value: "com.apple.configuration.legacy.interactive", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.declarations.management", Value: "com.apple.management.organization-info", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.declarations.management", Value: "com.apple.management.properties", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.declarations.management", Value: "com.apple.management.server-capabilities", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "management.client-capabilities", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "device.operating-system.version", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "device.identifier.udid", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "test.real-value", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "test.integer-value", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "test.error-value", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "test.dictionary-value", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "test.boolean-value", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "test.array-value", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "device.model.family", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "management.declarations", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "test.string-value", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "device.operating-system.supplemental.extra-version", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "device.operating-system.supplemental.build-version", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "device.operating-system.marketing-name", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "device.operating-system.family", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "device.operating-system.build-version", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "device.model.marketing-name", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "device.model.identifier", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "device.identifier.serial-number", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-versions", Value: "1.0.0", StatusID: "go_test_trace_id"},
	}

	sorter := func(values []storage.StatusValue) {
		sort.Slice(values, func(i, j int) bool {
			if values[i].Path != values[j].Path {
				return values[i].Path < values[j].Path
			} else if values[i].Value != values[j].Value {
				return values[i].Value < values[j].Value
			}
			return values[i].StatusID < values[j].StatusID
		})

	}

	// clear out any timestamps and optional Status ID fields (transient by nature)
	for i := range values {
		values[i].Timestamp = time.Time{}
		values[i].StatusID = ""
	}

	// clear out the optional StatusID fields
	for i := range eValues {
		eValues[i].StatusID = ""
	}

	sorter(eValues)
	sorter(values)

	// for _, s := range values {
	// 	fmt.Println(`{Path: "` + s.Path + `", Value: "` + s.Value + `", StatusID: "go_test_trace_id"},`)
	// }
	// fmt.Println("====")
	// for _, s := range eValues {
	// 	fmt.Println(`{Path: "` + s.Path + `", Value: "` + s.Value + `", StatusID: "go_test_trace_id"},`)
	// }
	// fmt.Println("====")

	if !reflect.DeepEqual(values, eValues) {
		t.Errorf("values: have: (%d) %v, want: (%d) %v", len(values), values, len(eValues), eValues)
	}

	// submit status for golang_test_enr_E4E7C11ECD86

	statusBytes, err = os.ReadFile("../../test/e2e/testdata/status.D0.error.json")
	if err != nil {
		t.Fatal(err)
	}

	enrHdr.Set(httpddm.EnrollmentIDHeader, "golang_test_enr_E4E7C11ECD86")

	resp = doReqHeader(mux, "PUT", "/status", enrHdr, statusBytes)
	expectHTTP(t, resp, 200)

	// test declaration errors

	resp = doReq(mux, "GET", "/v1/status-errors/golang_test_enr_E4E7C11ECD86", nil)
	expectHTTP(t, resp, 200)

	errorJSON := make(map[string][]errorJSONS)

	err = json.NewDecoder(resp.Body).Decode(&errorJSON)
	if err != nil {
		t.Fatal(err)
	}

	// clear any transient timestamp or optional status ID fields
	for k := range errorJSON {
		for i := range errorJSON[k] {
			errorJSON[k][i].Timestamp = time.Time{}
			errorJSON[k][i].StatusID = ""
		}
	}

	// error from "testdata/status.D0.error.json"
	myError := map[string][]errorJSONS{
		"golang_test_enr_E4E7C11ECD86": {{
			Path:  ".StatusItems.management.declarations.configurations",
			Error: []byte(`{"active":false,"identifier":"com.example.test","reasons":[{"code":"Info.NotReferencedByActivation","description":"Configuration “com.example.test:7c6d85989e823101” is not referenced by an activation.","details":{"Identifier":"com.example.test","ServerToken":"7c6d85989e823101"}}],"server-token":"7c6d85989e823101","valid":"unknown"}`),
			// StatusID: "go_test_trace_id",
		}},
	}

	if have, want := myError, errorJSON; !reflect.DeepEqual(have, want) {
		t.Errorf("error: have: (%d) %v, want: (%d) %v", len(have), have, len(want), want)
	}

	// test declaration status for enrollment with no status yet
	resp = doReq(mux, "GET", "/v1/declaration-status/golang_test_enr_4A5B529A3174", nil)
	expectHTTP(t, resp, 200)

	errorResp := &struct {
		Error string `json:"error"`
	}{}

	// decode delcaration status
	err = json.NewDecoder(resp.Body).Decode(errorResp)
	if err != nil {
		t.Fatal(err)
	}

	// should expect empty/no error
	if have, want := errorResp.Error, ""; have != want {
		t.Errorf("error: have: %q, want: %q", have, want)
	}

	// test declaration status

	resp = doReq(mux, "GET", "/v1/declaration-status/golang_test_enr_E4E7C11ECD86", nil)
	expectHTTP(t, resp, 200)

	dStatus := make(map[string][]ddm.DeclarationQueryStatus)

	// decode delcaration status
	err = json.NewDecoder(resp.Body).Decode(&dStatus)
	if err != nil {
		t.Fatal(err)
	}

	// clear out returned values that are transient or optional
	// to make our tests idempotent and work across backends
	// that do not implement the optional fields
	for k := range dStatus {
		for i := range dStatus[k] {
			dStatus[k][i].ServerToken = ""
			dStatus[k][i].StatusReceived = time.Time{}
			dStatus[k][i].StatusID = ""

			// should not be set?
			dStatus[k][i].ReasonsJSON = nil
			dStatus[k][i].ManifestType = ""
		}
	}

	// from the testdata file
	reasonsJSON := `[
              {
                "details" : {
                  "Identifier" : "com.example.test",
                  "ServerToken" : "7c6d85989e823101"
                },
                "description" : "Configuration “com.example.test:7c6d85989e823101” is not referenced by an activation.",
                "code" : "Info.NotReferencedByActivation"
              }
            ]`
	var reasons interface{}
	err = json.Unmarshal([]byte(reasonsJSON), &reasons)
	if err != nil {
		t.Fatal(err)
	}

	eStatus := map[string][]ddm.DeclarationQueryStatus{
		"golang_test_enr_E4E7C11ECD86": {
			{
				Current: false,
				DeclarationStatus: ddm.DeclarationStatus{
					Valid:      "unknown",
					Active:     false,
					Identifier: "com.example.test",
				},
				Reasons: reasons,
			},
		},
	}

	if have, want := dStatus, eStatus; !reflect.DeepEqual(have, want) {
		t.Errorf("declatation status: have: (%d) %v, want: (%d) %v", len(have), have, len(want), want)
	}

	//
	// test current=true
	//

	if n == nil {
		t.Fatal("nil notifier")
	}

	// upload our declaration
	resp = doReq(mux, "PUT", "/v1/declarations", []byte(testDecl2))
	expectHTTP(t, resp, 204)
	expectNotifierSlice(t, n, true, nil)

	// associate the set with the enrollment
	resp = doReq(mux, "PUT", "/v1/enrollment-sets/golang_test_enr_730E7C49E900?set=golang_test_set_793BBBD50EE9", nil)
	expectHTTP(t, resp, 204)
	expectNotifierSlice(t, n, true, []string{"golang_test_enr_730E7C49E900"})

	// associate the declaration with the set
	resp = doReq(mux, "PUT", "/v1/set-declarations/golang_test_set_793BBBD50EE9?declaration=com.example.test", nil)
	expectHTTP(t, resp, 204)
	expectNotifierSlice(t, n, true, []string{"golang_test_enr_730E7C49E900"})

	// retrieve the declaration to fetch the current server token
	resp = doReq(mux, "GET", "/v1/declarations/com.example.test", nil)
	expectHTTP(t, resp, 200)

	// decode the retrieved declaration
	rTestD := &TestDeclaration{}
	if err := json.NewDecoder(resp.Body).Decode(rTestD); err != nil {
		t.Fatal(err)
	}

	// replace the server token from "testdata/status.D0.error.json"
	// with whatever the backend has
	replacedStatusBytes := bytes.Replace(
		statusBytes,
		[]byte("7c6d85989e823101"),
		[]byte(rTestD.ServerToken),
		-1,
	)

	enrHdr.Set(httpddm.EnrollmentIDHeader, "golang_test_enr_730E7C49E900")

	resp = doReqHeader(mux, "PUT", "/status", enrHdr, replacedStatusBytes)
	expectHTTP(t, resp, 200)

	// retrieve the declaration status for both identifiers
	resp = doReq(mux, "GET", "/v1/declaration-status/golang_test_enr_E4E7C11ECD86,golang_test_enr_730E7C49E900", nil)
	expectHTTP(t, resp, 200)

	dStatus = make(map[string][]ddm.DeclarationQueryStatus)

	// decode delcaration status
	err = json.NewDecoder(resp.Body).Decode(&dStatus)
	if err != nil {
		t.Fatal(err)
	}

	// clear out unnecessary (to this test) fields
	for k := range dStatus {
		for i := range dStatus[k] {
			dStatus[k][i].Reasons = nil
			dStatus[k][i].StatusID = ""
			dStatus[k][i].StatusReceived = time.Time{}

			dStatus[k][i].ManifestType = ""
			dStatus[k][i].ReasonsJSON = nil
		}
	}

	eStatus = map[string][]ddm.DeclarationQueryStatus{
		"golang_test_enr_E4E7C11ECD86": {
			{
				Current: false,
				DeclarationStatus: ddm.DeclarationStatus{
					Valid:       "unknown",
					Active:      false,
					Identifier:  "com.example.test",
					ServerToken: "7c6d85989e823101",
				},
			},
		},
		"golang_test_enr_730E7C49E900": {
			{
				Current: true, // the main reason for this test is this true value
				DeclarationStatus: ddm.DeclarationStatus{
					Valid:       "unknown",
					Active:      false,
					Identifier:  "com.example.test",
					ServerToken: rTestD.ServerToken,
				},
			},
		},
	}

	if have, want := dStatus, eStatus; !reflect.DeepEqual(have, want) {
		t.Errorf("declatation status: have: (%d) %v, want: (%d) %v", len(have), have, len(want), want)
	}

	// remove the declaration-set association
	resp = doReq(mux, "DELETE", "/v1/set-declarations/golang_test_set_793BBBD50EE9?declaration=com.example.test", nil)
	expectHTTP(t, resp, 204)
	expectNotifierSlice(t, n, true, []string{"golang_test_enr_730E7C49E900"})

	// remove the set-enrollment association
	resp = doReq(mux, "DELETE", "/v1/enrollment-sets/golang_test_enr_730E7C49E900?set=golang_test_set_793BBBD50EE9", nil)
	expectHTTP(t, resp, 204)
	expectNotifierSlice(t, n, true, []string{"golang_test_enr_730E7C49E900"})

	// delete the declaration
	resp = doReq(mux, "DELETE", "/v1/declarations/com.example.test", nil)
	expectHTTP(t, resp, 204)
	// shouldn't notify here because the storage should know that
	// we are not currently associated with any sets
	expectNotifierSlice(t, n, false, nil)

}
