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

	eValues := []storage.StatusValue{
		{Path: ".StatusItems.device.operating-system.family", Value: "macOS", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.device.operating-system.supplemental.extra-version", Value: "(a)", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.declarations.management", Value: "com.apple.management.server-capabilities", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.device.operating-system.version", Value: "13.3.1", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.device.operating-system.build-version", Value: "22E261", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-versions", Value: "1.0.0", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.declarations.configurations", Value: "com.apple.configuration.passcode.settings", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.device.identifier.udid", Value: "0000FE00-5FADE97DCBBBEDA5", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.declarations.activations", Value: "com.apple.activation.simple", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.device.model.family", Value: "Mac", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "test.string-value", StatusID: "go_test_trace_id"},
		{Path: ".StatusItems.device.identifier.serial-number", Value: "ZRMXJQTTFX", StatusID: "go_test_trace_id"},
	}

	sorter := func(values []storage.StatusValue) {
		sort.Slice(values, func(i, j int) bool {
			return values[i].Path < values[j].Path
		})

	}

	// clear out any timestamps as they're dynamically generated
	for i, _ := range values {
		values[i].Timestamp = time.Time{}
	}

	sorter(eValues)
	sorter(values)

	// for _, s := range values {
	// 	fmt.Println(s.Path)
	// }
	// fmt.Println("====")
	// for _, s := range eValues {
	// 	fmt.Println(s.Path)
	// }
	// fmt.Println("====")

	// if !reflect.DeepEqual(values, eValues) {
	// 	t.Errorf("values: have: (%d) %v, want: (%d) %v", len(values), values, len(eValues), eValues)
	// }
}
