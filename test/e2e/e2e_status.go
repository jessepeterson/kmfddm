package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"sort"
	"testing"
	"time"

	httpddm "github.com/jessepeterson/kmfddm/http/ddm"
	"github.com/jessepeterson/kmfddm/storage"
)

func testStatus(t *testing.T, _ context.Context, mux http.Handler, _ storage.StatusStorer) {
	// statusBytes, err := os.ReadFile("../../storage/test/testdata/status.D0.error.json")
	statusBytes, err := os.ReadFile("../../storage/test/testdata/status.1st.json")
	if err != nil {
		t.Fatal(err)
	}

	enrHdr := make(http.Header)
	enrHdr.Add(httpddm.EnrollmentIDHeader, "golang_test_enr_87C029C236E0")

	resp := doReqHeader(mux, "PUT", "/status", enrHdr, statusBytes)
	expectHTTP(t, resp, 200)

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

	// clear out any timestamps as they're dynamically generated
	for i := range values {
		values[i].Timestamp = time.Time{}
	}

	// clear out the technically optional StatusID field
	for i := range values {
		values[i].StatusID = ""
	}
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

	// if !reflect.DeepEqual(values, eValues) {
	// 	t.Errorf("values: have: (%d) %v, want: (%d) %v", len(values), values, len(eValues), eValues)
	// }
}
