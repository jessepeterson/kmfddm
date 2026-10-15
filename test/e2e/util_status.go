package e2e

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	httpddm "github.com/jessepeterson/kmfddm/http/ddm"
	"github.com/jessepeterson/kmfddm/storage"
)

// statusFixtureDir is the directory holding the status report fixtures. The
// e2e suite is driven from each backend's own package directory, so the path
// has to climb back out to test/e2e.
const statusFixtureDir = "../../test/e2e/testdata"

// readStatusFixture reads the named status report fixture.
func readStatusFixture(t *testing.T, name string) []byte {
	t.Helper()

	buf, err := os.ReadFile(filepath.Join(statusFixtureDir, name))
	if err != nil {
		t.Fatal(err)
	}
	return buf
}

// submitStatus submits body as the status report for enrollmentID.
func submitStatus(t *testing.T, mux http.Handler, enrollmentID string, body []byte) *http.Response {
	t.Helper()

	hdr := make(http.Header)
	hdr.Set(httpddm.EnrollmentIDHeader, enrollmentID)
	return doReqHeader(mux, "PUT", "/status", hdr, body)
}

// getStatusValues retrieves the collected status values for target, which may
// be a comma-separated list of enrollment IDs.
func getStatusValues(t *testing.T, mux http.Handler, target string) map[string][]storage.StoredStatusValue {
	t.Helper()
	return getStatusValuesWithPrefix(t, mux, target, "")
}

// getStatusValuesWithPrefix is getStatusValues restricted to the values whose
// path begins with prefix.
func getStatusValuesWithPrefix(t *testing.T, mux http.Handler, target, prefix string) map[string][]storage.StoredStatusValue {
	t.Helper()

	u := "/v1/status-values/" + target
	if prefix != "" {
		// the prefix is SQL LIKE syntax and carries a literal '%', which has
		// to be escaped or it is not a valid query string
		u += "?prefix=" + url.QueryEscape(prefix)
	}

	resp := doReq(mux, "GET", u, nil)
	expectHTTP(t, resp, 200)

	have := make(map[string][]storage.StoredStatusValue)
	if err := json.NewDecoder(resp.Body).Decode(&have); err != nil {
		t.Fatal(err)
	}

	// the timestamp is assigned at store time and the status ID is optional:
	// clear both so the comparison holds across backends.
	for _, values := range have {
		for i := range values {
			values[i].Timestamp = time.Time{}
			values[i].StatusID = ""
		}
	}
	return have
}

// getStatusErrors retrieves the collected status errors for target, which may
// be a comma-separated list of enrollment IDs.
func getStatusErrors(t *testing.T, mux http.Handler, target string) map[string][]errorJSONS {
	t.Helper()

	resp := doReq(mux, "GET", "/v1/status-errors/"+target, nil)
	expectHTTP(t, resp, 200)

	have := make(map[string][]errorJSONS)
	if err := json.NewDecoder(resp.Body).Decode(&have); err != nil {
		t.Fatal(err)
	}

	for _, errs := range have {
		for i := range errs {
			errs[i].Timestamp = time.Time{}
			errs[i].StatusID = ""
		}
	}
	return have
}

// getDeclarationStatus retrieves the declaration status for target, which may
// be a comma-separated list of enrollment IDs.
func getDeclarationStatus(t *testing.T, mux http.Handler, target string) map[string][]storage.DeclarationQueryStatus {
	t.Helper()

	resp := doReq(mux, "GET", "/v1/declaration-status/"+target, nil)
	expectHTTP(t, resp, 200)

	have := make(map[string][]storage.DeclarationQueryStatus)
	if err := json.NewDecoder(resp.Body).Decode(&have); err != nil {
		t.Fatal(err)
	}

	// these are either assigned by the backend at read time or optional
	// across backends. ServerToken and Reasons are left alone: the tests that
	// care about them assert them directly.
	for _, statuses := range have {
		for i := range statuses {
			statuses[i].StatusReceived = time.Time{}
			statuses[i].StatusID = ""
			statuses[i].ManifestType = ""
			statuses[i].ReasonsJSON = nil
		}
	}
	return have
}

// expectNoStatus asserts that nothing at all was stored for target: no values,
// no errors, no declaration status.
func expectNoStatus(t *testing.T, mux http.Handler, target string) {
	t.Helper()

	if have := getStatusValues(t, mux, target); len(have) != 0 {
		t.Errorf("status values: have: %v, want: none", have)
	}
	if have := getStatusErrors(t, mux, target); len(have) != 0 {
		t.Errorf("status errors: have: %v, want: none", have)
	}
	if have := getDeclarationStatus(t, mux, target); len(have) != 0 {
		t.Errorf("declaration status: have: %v, want: none", have)
	}
}
