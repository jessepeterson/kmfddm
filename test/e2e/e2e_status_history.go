package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"testing"

	"github.com/alexedwards/flow"
	"github.com/jessepeterson/kmfddm/http/api"
	httpddm "github.com/jessepeterson/kmfddm/http/ddm"
	"github.com/micromdm/nanolib/http/trace"
	"github.com/micromdm/nanolib/log"
)

// testStatusIDHeader sets the trace ID (and thus the status ID) of a
// status report for muxes setup with newStatusIDMux.
const testStatusIDHeader = "X-Test-Status-ID"

// testErrorID identifies an error within a test status report.
type testErrorID struct {
	Report int `json:"report"`
	Error  int `json:"error"`
}

// statusReportWithErrors creates a status report for report number n that
// contains errorCount errors identifying themselves.
func statusReportWithErrors(t *testing.T, n, errorCount int) []byte {
	t.Helper()
	errs := make([]testErrorID, errorCount)
	for i := range errs {
		errs[i] = testErrorID{Report: n, Error: i}
	}
	b, err := json.Marshal(map[string]interface{}{
		"StatusItems": map[string]interface{}{},
		"Errors":      errs,
	})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// putStatusReport submits a status report for enrollmentID.
func putStatusReport(t *testing.T, mux http.Handler, enrollmentID, statusID string, report []byte) {
	t.Helper()
	hdr := make(http.Header)
	hdr.Set(httpddm.EnrollmentIDHeader, enrollmentID)
	if statusID != "" {
		hdr.Set(testStatusIDHeader, statusID)
	}
	resp := doReqHeader(mux, "PUT", "/status", hdr, report)
	expectHTTP(t, resp, 200)
}

// expectStatusReport retrieves a status report for enrollmentID using query
// and checks it is JSON-equivalent to want. A nil want expects a 404.
// The response is returned for further inspection.
func expectStatusReport(t *testing.T, mux http.Handler, enrollmentID, query string, want []byte) *http.Response {
	t.Helper()
	resp := doReq(mux, "GET", "/v1/status-report/"+enrollmentID+"?"+query, nil)
	if want == nil {
		expectHTTP(t, resp, 404)
		return resp
	}
	expectHTTP(t, resp, 200)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	// backends may normalize the stored JSON
	var have, wantV interface{}
	if err = json.Unmarshal(body, &have); err != nil {
		t.Fatalf("status report %s: %v", query, err)
	}
	if err = json.Unmarshal(want, &wantV); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(have, wantV) {
		t.Errorf("status report %s: have: %s, want: %s", query, body, want)
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))
	return resp
}

// expectStatusErrors retrieves the status errors for enrollmentID and
// checks that they are the test errors in want (in order).
func expectStatusErrors(t *testing.T, mux http.Handler, enrollmentID string, want []testErrorID) {
	t.Helper()
	resp := doReq(mux, "GET", "/v1/status-errors/"+enrollmentID, nil)
	expectHTTP(t, resp, 200)
	errorJSON := make(map[string][]errorJSONS)
	if err := json.NewDecoder(resp.Body).Decode(&errorJSON); err != nil {
		t.Fatal(err)
	}
	var have []testErrorID
	for _, e := range errorJSON[enrollmentID] {
		var id testErrorID
		if err := json.Unmarshal(e.Error, &id); err != nil {
			t.Fatal(err)
		}
		have = append(have, id)
	}
	if !reflect.DeepEqual(have, want) {
		t.Errorf("status errors: have: %v, want: %v", have, want)
	}
}

// expectedErrors returns the test errors for reports with errorCounts.
func expectedErrors(errorCounts []int) (errs []testErrorID) {
	for n, ct := range errorCounts {
		for i := 0; i < ct; i++ {
			errs = append(errs, testErrorID{Report: n, Error: i})
		}
	}
	return
}

// testStatusHistory tests multiple status reports for a single enrollment.
func testStatusHistory(t *testing.T, mux http.Handler) {
	const enrollmentID = "golang_test_enr_5E2A0C9B71D4"

	errorCounts := []int{2, 1, 2}
	var reports [][]byte
	for n, ct := range errorCounts {
		reports = append(reports, statusReportWithErrors(t, n, ct))
		putStatusReport(t, mux, enrollmentID, "", reports[n])
	}

	// index 0 is the most recent status report
	expectStatusReport(t, mux, enrollmentID, "index=0", reports[len(reports)-1])

	// errors accumulate across status reports
	expectStatusErrors(t, mux, enrollmentID, expectedErrors(errorCounts))
}

// newStatusIDMux creates a mux for storage that sets the status ID of
// submitted status reports from the testStatusIDHeader header.
func newStatusIDMux(storage TestStorage) http.Handler {
	mux := flow.New()
	logger := log.NopLogger
	api.HandleAPIv1("/v1", mux, logger, storage, &captureNotifier{store: storage})
	handleDDM(mux, logger, storage)
	return trace.NewTraceLoggingHandler(mux, logger, func(r *http.Request) string {
		return r.Header.Get(testStatusIDHeader)
	})
}

// TestStatusRetention tests that storage keeps only the most recent
// keepReports status reports and the most recent keepErrors errors per
// enrollment. It is intended for storage backends that support limiting
// retention and storage should be configured with these limits.
func TestStatusRetention(t *testing.T, _ context.Context, storage TestStorage, keepReports, keepErrors int) {
	const enrollmentID = "golang_test_enr_0B8E4F27C3A6"

	// reports carry a varying number of errors
	errorCounts := []int{3, 1, 2, 3, 1}
	want := expectedErrors(errorCounts)
	if keepReports < 1 || keepReports >= len(errorCounts) {
		t.Fatalf("report retention must be between 1 and %d", len(errorCounts)-1)
	}
	if keepErrors < 1 || keepErrors >= len(want) {
		t.Fatalf("error retention must be between 1 and %d", len(want)-1)
	}

	mux := newStatusIDMux(storage)
	statusID := func(n int) string { return fmt.Sprintf("golang_test_sts_%d", n) }

	var reports [][]byte
	for n, ct := range errorCounts {
		reports = append(reports, statusReportWithErrors(t, n, ct))
		putStatusReport(t, mux, enrollmentID, statusID(n), reports[n])
	}
	newest := len(reports) - 1

	for i := 0; i < keepReports; i++ {
		resp := expectStatusReport(t, mux, enrollmentID, "index="+strconv.Itoa(i), reports[newest-i])
		if have, want := resp.Header.Get("X-Status-Report-ID"), statusID(newest-i); have != want {
			t.Errorf("index %d: status ID: have: %q, want: %q", i, have, want)
		}
	}

	// pruned beyond retention
	expectStatusReport(t, mux, enrollmentID, "index="+strconv.Itoa(keepReports), nil)
	expectStatusReport(t, mux, enrollmentID, "status_id="+statusID(newest-keepReports), nil)

	// by status ID reports the index
	resp := expectStatusReport(t, mux, enrollmentID, "status_id="+statusID(newest-1), reports[newest-1])
	if have, want := resp.Header.Get("X-Status-Report-Index"), "1"; have != want {
		t.Errorf("status ID index: have: %q, want: %q", have, want)
	}

	// by both status ID and index must match both
	expectStatusReport(t, mux, enrollmentID, "status_id="+statusID(newest-1)+"&index=1", reports[newest-1])
	expectStatusReport(t, mux, enrollmentID, "status_id="+statusID(newest-1)+"&index=0", nil)

	// only the most recent errors are kept, regardless of which
	// status report they came from
	expectStatusErrors(t, mux, enrollmentID, want[len(want)-keepErrors:])
}
