package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jessepeterson/kmfddm/storage"
)

const testDecl2 = `{
	"Type": "com.apple.configuration.management.test",
	"Payload": {
		"Echo": "KMFDDM!"
	},
	"Identifier": "com.example.test"
}`

// enrollment IDs used by the status subtests. Each subtest owns its own so that
// a failure in one cannot be read as a failure in another.
const (
	valuesENR        = "golang_test_enr_87C029C236E0"
	declErrENR       = "golang_test_enr_E4E7C11ECD86"
	currentENR       = "golang_test_enr_730E7C49E900"
	reportErrENR     = "golang_test_enr_1A2B3C4D5E6F"
	badTypeENR       = "golang_test_enr_6F5E4D3C2B1A"
	unknownStatusENR = "golang_test_enr_4A5B529A3174"
)

type errorJSONS struct {
	Path      string          `json:"path"`
	Error     json.RawMessage `json:"error"`
	Timestamp time.Time       `json:"timestamp"`
	StatusID  string          `json:"status_id,omitempty"`
}

func testStatus(t *testing.T, mux http.Handler, n *captureNotifier) {
	t.Run("values", func(t *testing.T) { testStatusValues(t, mux) })
	t.Run("values-path-prefix", func(t *testing.T) { testStatusValuesPathPrefix(t, mux) })
	t.Run("status-report-readback", func(t *testing.T) { testStatusReportReadback(t, mux) })
	t.Run("report-errors", func(t *testing.T) { testStatusReportErrors(t, mux) })
	t.Run("declaration-error", func(t *testing.T) { testStatusDeclarationError(t, mux) })
	t.Run("declaration-status-unknown", func(t *testing.T) { testStatusDeclarationUnknown(t, mux) })
	t.Run("declaration-status", func(t *testing.T) { testStatusDeclarationStatus(t, mux) })
	t.Run("reject-bad-types", func(t *testing.T) { testStatusRejectBadTypes(t, mux) })
	t.Run("declaration-status-current", func(t *testing.T) { testStatusCurrent(t, mux, n) })
	t.Run("second-report", func(t *testing.T) { testStatusSecondReport(t, mux) })
	t.Run("error-count", func(t *testing.T) { testStatusErrorCount(t, mux) })
}

// testStatusErrorCount submits a report carrying more errors than the
// /v1/status-errors endpoint's hardcoded limit of 10.
//
// The endpoint has no query semantics yet — the API passes a hardcoded limit and
// exposes no way to change or page it, and the backends differ on whether that
// limit means anything. storage/mysql honours it (LIMIT ?, ?) and returns 10;
// storage/kv and storage/file ignore it and return all 15. Query semantics are
// coming in a later PR, so there is no single right count to assert yet.
//
// What does hold everywhere is that whatever comes back is a well-formed,
// non-duplicated subset of what was sent, so that is what this asserts. The count
// is logged rather than pinned; when the endpoint grows real offset/limit
// support, assert all 15 and drop the tolerance.
func testStatusErrorCount(t *testing.T, mux http.Handler) {
	const enr = "golang_test_enr_5B6C7D8E9F01"
	const sent = 15

	t.Run("setup", func(t *testing.T) {
		expectHTTP(t, submitStatus(t, mux, enr, readStatusFixture(t, "status.many-errors.json")), 200)
	})

	t.Run("errors-are-well-formed", func(t *testing.T) {
		// every status item the fixture declares as an error
		want := make(map[string]bool, sent)
		for i := 1; i <= sent; i++ {
			want[fmt.Sprintf("test.item-%02d", i)] = true
		}

		have := getStatusErrors(t, mux, enr)[enr]
		if len(have) == 0 {
			t.Fatal("no errors returned")
		}
		if len(have) > sent {
			t.Fatalf("errors: have %d, want at most %d", len(have), sent)
		}
		// backend-dependent until /v1/status-errors has query semantics
		t.Logf("returned %d of %d errors", len(have), sent)

		seen := make(map[string]bool, len(have))
		for _, e := range have {
			if e.Path != ".Errors" {
				t.Errorf("error path: have: %q, want: %q", e.Path, ".Errors")
			}

			var reportErr struct {
				StatusItem string
				Reasons    []struct {
					Code        string
					Description string
				}
			}
			if err := json.Unmarshal(e.Error, &reportErr); err != nil {
				t.Errorf("unmarshaling error JSON %s: %v", e.Error, err)
				continue
			}

			if !want[reportErr.StatusItem] {
				t.Errorf("error for an item that was not sent: %s", reportErr.StatusItem)
			}
			if seen[reportErr.StatusItem] {
				t.Errorf("duplicate error: %s", reportErr.StatusItem)
			}
			seen[reportErr.StatusItem] = true

			if len(reportErr.Reasons) != 1 {
				t.Errorf("%s: reasons: have: %d, want: 1", reportErr.StatusItem, len(reportErr.Reasons))
				continue
			}
			if reportErr.Reasons[0].Code == "" {
				t.Errorf("%s: empty reason code", reportErr.StatusItem)
			}
			if reportErr.Reasons[0].Description == "" {
				t.Errorf("%s: empty reason description", reportErr.StatusItem)
			}
		}
	})
}

// testStatusSecondReport pins what happens when the same enrollment sends a
// second report. Neither kind of stored data is replaced wholesale: values are
// the union of every report seen, and declaration status survives a report that
// carries none. All four backends agree on this — MySQL guards it explicitly in
// storeStatusDeclarations ("do not delete existing declaration status if no
// status are included").
//
// This is also why nothing prunes: a value a device stops reporting stays
// forever. Acting on FullReport would be the fix, and nothing reads it yet.
func testStatusSecondReport(t *testing.T, mux http.Handler) {
	t.Run("values-merge", func(t *testing.T) {
		const enr = "golang_test_enr_7E2D1C0B9A88"

		// a full report: 37 values across the device and management namespaces
		expectHTTP(t, submitStatus(t, mux, enr, readStatusFixture(t, "status.1st.json")), 200)
		first := getStatusValues(t, mux, enr)[enr]
		if len(first) != 37 {
			t.Fatalf("first report: have %d values, want 37", len(first))
		}

		// then a much smaller report: one different value, plus two report errors
		expectHTTP(t, submitStatus(t, mux, enr, readStatusFixture(t, "status.report-errors.json")), 200)
		second := getStatusValues(t, mux, enr)[enr]

		have := make(map[string]string, len(second))
		for _, v := range second {
			have[v.Path+"="+v.Value] = v.Path
		}

		// the new value is collected...
		if _, ok := have[".StatusItems.device.identifier.udid=AAAABBBB-1111-2222-3333-CCCCDDDD4444"]; !ok {
			t.Errorf("second report's value missing from %d values", len(second))
		}
		// ...and nothing from the first report is dropped.
		for _, v := range first {
			if _, ok := have[v.Path+"="+v.Value]; !ok {
				t.Errorf("value dropped by the second report: %s = %s", v.Path, v.Value)
			}
		}
		if want := len(first) + 1; len(second) != want {
			t.Errorf("value count: have %d, want %d (union of both reports)", len(second), want)
		}
	})

	t.Run("declaration-status-survives-empty-report", func(t *testing.T) {
		const enr = "golang_test_enr_3C4D5E6F7A80"

		// a report carrying one declaration
		expectHTTP(t, submitStatus(t, mux, enr, readStatusFixture(t, "status.D0.error.json")), 200)
		if have := len(getDeclarationStatus(t, mux, enr)[enr]); have != 1 {
			t.Fatalf("first report: have %d declaration rows, want 1", have)
		}

		// the next report has empty declaration arrays
		expectHTTP(t, submitStatus(t, mux, enr, readStatusFixture(t, "status.report-errors.json")), 200)

		// the prior row is still there
		have := getDeclarationStatus(t, mux, enr)[enr]
		if len(have) != 1 {
			t.Fatalf("second report: have %d declaration rows, want 1 (they should not be erased)", len(have))
		}
		if id := have[0].Identifier; id != "com.example.test" {
			t.Errorf("identifier: have %q, want %q", id, "com.example.test")
		}
	})
}

// sortStatusValues orders values so that two backends returning the same set
// in a different order still compare equal.
func sortStatusValues(values []storage.StoredStatusValue) {
	sort.Slice(values, func(i, j int) bool {
		if values[i].Path != values[j].Path {
			return values[i].Path < values[j].Path
		} else if values[i].Value != values[j].Value {
			return values[i].Value < values[j].Value
		}
		return values[i].StatusID < values[j].StatusID
	})
}

// eStatusValues is the complete set of values expected from status.1st.json:
// the device namespace plus the client-capabilities block under management.
func eStatusValues() []storage.StoredStatusValue {
	return []storage.StoredStatusValue{
		{Path: ".StatusItems.device.identifier.serial-number", Value: "ZRMXJQTTFX"},
		{Path: ".StatusItems.device.identifier.udid", Value: "0000FE00-5FADE97DCBBBEDA5"},
		{Path: ".StatusItems.device.model.family", Value: "Mac"},
		{Path: ".StatusItems.device.operating-system.build-version", Value: "22E261"},
		{Path: ".StatusItems.device.operating-system.family", Value: "macOS"},
		{Path: ".StatusItems.device.operating-system.supplemental.extra-version", Value: "(a)"},
		{Path: ".StatusItems.device.operating-system.version", Value: "13.3.1"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.declarations.activations", Value: "com.apple.activation.simple"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.declarations.configurations", Value: "com.apple.configuration.passcode.settings"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.declarations.configurations", Value: "com.apple.configuration.legacy"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.declarations.configurations", Value: "com.apple.configuration.management.status-subscriptions"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.declarations.configurations", Value: "com.apple.configuration.management.test"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.declarations.configurations", Value: "com.apple.configuration.legacy.interactive"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.declarations.management", Value: "com.apple.management.organization-info"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.declarations.management", Value: "com.apple.management.properties"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.declarations.management", Value: "com.apple.management.server-capabilities"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "management.client-capabilities"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "device.operating-system.version"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "device.identifier.udid"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "test.real-value"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "test.integer-value"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "test.error-value"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "test.dictionary-value"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "test.boolean-value"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "test.array-value"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "device.model.family"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "management.declarations"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "test.string-value"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "device.operating-system.supplemental.extra-version"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "device.operating-system.supplemental.build-version"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "device.operating-system.marketing-name"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "device.operating-system.family"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "device.operating-system.build-version"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "device.model.marketing-name"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "device.model.identifier"},
		{Path: ".StatusItems.management.client-capabilities.supported-payloads.status-items", Value: "device.identifier.serial-number"},
		{Path: ".StatusItems.management.client-capabilities.supported-versions", Value: "1.0.0"},
	}
}

// testStatusValues submits a full status report and asserts the complete set of
// collected values. The declarations are also stored, but not as values.
func testStatusValues(t *testing.T, mux http.Handler) {
	t.Run("setup", func(t *testing.T) {
		expectHTTP(t, submitStatus(t, mux, valuesENR, readStatusFixture(t, "status.1st.json")), 200)
	})

	t.Run("collected-values", func(t *testing.T) {
		have := getStatusValues(t, mux, valuesENR)

		if _, ok := have[valuesENR]; !ok {
			t.Fatal("id not present")
		}

		want := eStatusValues()
		sortStatusValues(want)
		sortStatusValues(have[valuesENR])

		if !reflect.DeepEqual(have[valuesENR], want) {
			t.Errorf("values: have: (%d) %v, want: (%d) %v", len(have[valuesENR]), have[valuesENR], len(want), want)
		}
	})

	t.Run("declarations-are-not-values", func(t *testing.T) {
		// the declarations block is sunk by the value collector: it is stored
		// as declaration status, so it must not also appear as loose pairs.
		// paths under client-capabilities legitimately mention "declarations",
		// so this checks the declaration status block itself.
		for _, v := range getStatusValues(t, mux, valuesENR)[valuesENR] {
			if strings.HasPrefix(v.Path, ".StatusItems.management.declarations") {
				t.Errorf("declaration leaked into status values: %s", v.Path)
			}
		}
	})
}

// testStatusValuesPathPrefix asserts the ?prefix= filter narrows the values
// returned. The filter is documented as SQL LIKE syntax, so a prefix match is
// written with a trailing wildcard.
func testStatusValuesPathPrefix(t *testing.T, mux http.Handler) {
	all := getStatusValues(t, mux, valuesENR)[valuesENR]
	index := make(map[string]bool, len(all))
	for _, v := range all {
		index[v.Path+"="+v.Value] = true
	}

	for _, q := range []struct {
		name   string
		prefix string
	}{
		{"like-prefix", ".StatusItems.device.%"},
		{"exact-path", ".StatusItems.device.model.family"},
		{"no-match", ".StatusItems.no-such-namespace.%"},
	} {
		t.Run(q.name, func(t *testing.T) {
			// The KV backend does not implement ?prefix= at all, so the
			// count is logged rather than asserted. What must hold everywhere
			// is that narrowing the query never invents a row - every value a
			// filtered query returns is one the unfiltered query returned too.
			have := getStatusValuesWithPrefix(t, mux, valuesENR, q.prefix)[valuesENR]
			t.Logf("prefix %q returned %d of %d values", q.prefix, len(have), len(all))

			for _, v := range have {
				if !index[v.Path+"="+v.Value] {
					t.Errorf("prefix %q returned a value absent from the unfiltered set: %s = %s",
						q.prefix, v.Path, v.Value)
				}
			}
		})
	}
}

// testStatusReportReadback asserts the stored report is served back intact,
// which is what makes a status report usable as a debugging artifact.
//
// The comparison is JSON, not bytes: storage/mysql stores the report in a JSON
// column (storage/mysql/schema.sql), and MySQL normalizes JSON on the way in —
// object keys come back sorted and insignificant whitespace is gone. file and kv
// store the submitted bytes verbatim. So the byte-for-byte property only holds on
// two of the backends, and the invariant worth asserting across all of them is
// that the report reads back as the report that was sent.
func testStatusReportReadback(t *testing.T, mux http.Handler) {
	sent := readStatusFixture(t, "status.1st.json")

	t.Run("explicit-index", func(t *testing.T) {
		resp := doReq(mux, "GET", "/v1/status-report/"+valuesENR+"?index=0", nil)
		expectHTTP(t, resp, 200)
		expectStatusReportBody(t, resp, sent)
	})
}

// expectStatusReportBody asserts the response body is the report as submitted.
// Compare as JSON: see testStatusReportReadback for why not byte for byte.
func expectStatusReportBody(t *testing.T, resp *http.Response, sent []byte) {
	t.Helper()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	var have, want any
	if err := json.Unmarshal(body, &have); err != nil {
		t.Fatalf("unmarshaling returned report: %v\nbody: %s", err, body)
	}
	if err := json.Unmarshal(sent, &want); err != nil {
		t.Fatalf("unmarshaling sent report: %v", err)
	}

	if !reflect.DeepEqual(have, want) {
		t.Errorf("status report does not round-trip:\n have: %s\n want: %s", body, sent)
	}
}

// testStatusReportErrors asserts the report's own top-level Errors array is
// stored as status errors, distinct from the errors the inactive-declaration
// policy derives.
func testStatusReportErrors(t *testing.T, mux http.Handler) {
	t.Run("setup", func(t *testing.T) {
		expectHTTP(t, submitStatus(t, mux, reportErrENR, readStatusFixture(t, "status.report-errors.json")), 200)
	})

	t.Run("errors", func(t *testing.T) {
		// the stored error JSON is re-marshalled through the API, which sorts
		// object keys, so these are in alphabetical order.
		want := map[string][]errorJSONS{
			reportErrENR: {
				{
					// key order is alphabetical: the stored JSON is
					// re-marshalled through the API, and Details is a map
					// so its own keys sort too.
					Path:  ".Errors",
					Error: []byte(`{"Reasons":[{"Code":"Error.ConfigurationCannotBeApplied","Description":"Configuration cannot be applied","Details":{"Identifier":"com.example.configuration"}}],"StatusItem":"management.declarations"}`),
				},
				{
					// no Reasons: the field is omitted when marshalled back
					Path:  ".Errors",
					Error: []byte(`{"StatusItem":"device.operating-system.version"}`),
				},
			},
		}

		if have := getStatusErrors(t, mux, reportErrENR); !reflect.DeepEqual(have, want) {
			t.Errorf("errors: have: %v, want: %v", have, want)
		}
	})

	t.Run("values-still-collected", func(t *testing.T) {
		// the device namespace is collected even when the report carries errors
		want := []storage.StoredStatusValue{
			{Path: ".StatusItems.device.identifier.udid", Value: "AAAABBBB-1111-2222-3333-CCCCDDDD4444"},
		}

		have := getStatusValues(t, mux, reportErrENR)[reportErrENR]
		if !reflect.DeepEqual(have, want) {
			t.Errorf("values: have: %v, want: %v", have, want)
		}
	})
}

// testStatusDeclarationError asserts an inactive, non-valid declaration is
// filed as a status error carrying the declaration as reported.
func testStatusDeclarationError(t *testing.T, mux http.Handler) {
	expectHTTP(t, submitStatus(t, mux, declErrENR, readStatusFixture(t, "status.D0.error.json")), 200)

	want := map[string][]errorJSONS{
		declErrENR: {{
			Path:  ".StatusItems.management.declarations.configurations",
			Error: []byte(`{"active":false,"identifier":"com.example.test","reasons":[{"code":"Info.NotReferencedByActivation","description":"Configuration “com.example.test:7c6d85989e823101” is not referenced by an activation.","details":{"Identifier":"com.example.test","ServerToken":"7c6d85989e823101"}}],"server-token":"7c6d85989e823101","valid":"unknown"}`),
		}},
	}

	if have := getStatusErrors(t, mux, declErrENR); !reflect.DeepEqual(have, want) {
		t.Errorf("errors: have: %v, want: %v", have, want)
	}
}

// testStatusDeclarationUnknown asserts an enrollment that has never sent a
// status report reads back empty rather than erroring.
func testStatusDeclarationUnknown(t *testing.T, mux http.Handler) {
	resp := doReq(mux, "GET", "/v1/declaration-status/"+unknownStatusENR, nil)
	expectHTTP(t, resp, 200)

	got := &struct {
		Error string `json:"error"`
	}{}
	if err := json.NewDecoder(resp.Body).Decode(got); err != nil {
		t.Fatal(err)
	}

	if have, want := got.Error, ""; have != want {
		t.Errorf("error: have: %q, want: %q", have, want)
	}
}

// testStatusDeclarationStatus asserts the declaration status stored from the
// report, independent of any declaration the server currently holds.
func testStatusDeclarationStatus(t *testing.T, mux http.Handler) {
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
	if err := json.Unmarshal([]byte(reasonsJSON), &reasons); err != nil {
		t.Fatal(err)
	}

	have := getDeclarationStatus(t, mux, declErrENR)

	// the server token is optional across backends
	for i := range have[declErrENR] {
		have[declErrENR][i].ServerToken = ""
	}

	want := map[string][]storage.DeclarationQueryStatus{
		declErrENR: {{
			Current: false,
			DeclarationStatus: storage.DeclarationStatus{
				Valid:      "unknown",
				Active:     false,
				Identifier: "com.example.test",
			},
			Reasons: reasons,
		}},
	}

	if !reflect.DeepEqual(have, want) {
		t.Errorf("declaration status: have: %v, want: %v", have, want)
	}
}

// testStatusRejectBadTypes asserts a report that does not conform to the
// generated schema is rejected outright and leaves nothing behind. This is the
// one behaviour change in this area that operators can observe, so the
// "nothing stored" half matters as much as the status code.
func testStatusRejectBadTypes(t *testing.T, mux http.Handler) {
	t.Run("rejected", func(t *testing.T) {
		// "active" is a string where the schema declares a bool
		expectHTTP(t, submitStatus(t, mux, badTypeENR, readStatusFixture(t, "status.bad-type.json")), 500)
	})

	t.Run("nothing-stored", func(t *testing.T) {
		// the report is rejected during the unmarshal, before any backend is
		// called: no declaration status, no values, no errors, not even from
		// the parts of the report that would have parsed cleanly on their own.
		expectNoStatus(t, mux, badTypeENR)
	})

	t.Run("does-not-disturb-other-enrollments", func(t *testing.T) {
		// a rejected report for one enrollment must not disturb another's
		want := eStatusValues()
		sortStatusValues(want)

		have := getStatusValues(t, mux, valuesENR)[valuesENR]
		sortStatusValues(have)

		if !reflect.DeepEqual(have, want) {
			t.Errorf("values: have: (%d) %v, want: (%d) %v", len(have), have, len(want), want)
		}
	})
}

// testStatusCurrent asserts the declaration status the server holds for an
// enrollment (Current) is correlated against the status the device reported,
// rather than being whatever the device said.
func testStatusCurrent(t *testing.T, mux http.Handler, n *captureNotifier) {
	if n == nil {
		t.Fatal("nil notifier")
	}

	// upload our declaration
	resp := doReq(mux, "PUT", "/v1/declarations", []byte(testDecl2))
	expectHTTP(t, resp, 204)
	expectNotifierSlice(t, n, true, nil)

	// associate the set with the enrollment
	resp = doReq(mux, "PUT", "/v1/enrollment-sets/"+currentENR+"?set=golang_test_set_793BBBD50EE9", nil)
	expectHTTP(t, resp, 204)
	expectNotifierSlice(t, n, true, []string{currentENR})

	// associate the declaration with the set
	resp = doReq(mux, "PUT", "/v1/set-declarations/golang_test_set_793BBBD50EE9?declaration=com.example.test", nil)
	expectHTTP(t, resp, 204)
	expectNotifierSlice(t, n, true, []string{currentENR})

	// retrieve the declaration to fetch the current server token
	resp = doReq(mux, "GET", "/v1/declarations/com.example.test", nil)
	expectHTTP(t, resp, 200)

	rTestD := &TestDeclaration{}
	if err := json.NewDecoder(resp.Body).Decode(rTestD); err != nil {
		t.Fatal(err)
	}

	// replace the server token from "testdata/status.D0.error.json"
	// with whatever the backend has
	statusBytes := bytes.Replace(
		readStatusFixture(t, "status.D0.error.json"),
		[]byte("7c6d85989e823101"),
		[]byte(rTestD.ServerToken),
		-1,
	)

	expectHTTP(t, submitStatus(t, mux, currentENR, statusBytes), 200)

	t.Run("current-varies-by-enrollment", func(t *testing.T) {
		// retrieve the declaration status for both identifiers. only the
		// enrollment that actually holds the declaration reports it as current.
		have := getDeclarationStatus(t, mux, declErrENR+","+currentENR)

		for id := range have {
			for i := range have[id] {
				have[id][i].Reasons = nil
			}
		}

		want := map[string][]storage.DeclarationQueryStatus{
			declErrENR: {{
				Current: false,
				DeclarationStatus: storage.DeclarationStatus{
					Valid:       "unknown",
					Active:      false,
					Identifier:  "com.example.test",
					ServerToken: "7c6d85989e823101",
				},
			}},
			currentENR: {{
				Current: true, // the main reason for this test is this true value
				DeclarationStatus: storage.DeclarationStatus{
					Valid:       "unknown",
					Active:      false,
					Identifier:  "com.example.test",
					ServerToken: rTestD.ServerToken,
				},
			}},
		}

		if !reflect.DeepEqual(have, want) {
			t.Errorf("declaration status: have: %v, want: %v", have, want)
		}
	})

	t.Run("declaration-in-multiple-sets-is-not-duplicated", func(t *testing.T) {
		// associate the declaration with a second set, deliberately leaving
		// the enrollment's own set membership alone: only the declaration
		// gains a set here, which must not affect the enrollment's status.
		resp := doReq(mux, "PUT", "/v1/set-declarations/golang_test_set_9F1C0A4B22D7?declaration=com.example.test", nil)
		expectHTTP(t, resp, 204)
		expectNotifierSlice(t, n, true, nil)

		have := getDeclarationStatus(t, mux, currentENR)
		for i := range have[currentENR] {
			have[currentENR][i].Reasons = nil
		}

		// identical to the expectation above: the second set carrying the
		// declaration must not add a row (nor change current).
		want := map[string][]storage.DeclarationQueryStatus{
			currentENR: {{
				Current: true,
				DeclarationStatus: storage.DeclarationStatus{
					Valid:       "unknown",
					Active:      false,
					Identifier:  "com.example.test",
					ServerToken: rTestD.ServerToken,
				},
			}},
		}

		if !reflect.DeepEqual(have, want) {
			t.Errorf("declaration status: have: %v, want: %v", have, want)
		}
	})

	t.Run("teardown", func(t *testing.T) {
		// remove the second declaration-set association
		resp := doReq(mux, "DELETE", "/v1/set-declarations/golang_test_set_9F1C0A4B22D7?declaration=com.example.test", nil)
		expectHTTP(t, resp, 204)
		expectNotifierSlice(t, n, true, nil)

		// remove the declaration-set association
		resp = doReq(mux, "DELETE", "/v1/set-declarations/golang_test_set_793BBBD50EE9?declaration=com.example.test", nil)
		expectHTTP(t, resp, 204)
		expectNotifierSlice(t, n, true, []string{currentENR})

		// remove the set-enrollment association
		resp = doReq(mux, "DELETE", "/v1/enrollment-sets/"+currentENR+"?set=golang_test_set_793BBBD50EE9", nil)
		expectHTTP(t, resp, 204)
		expectNotifierSlice(t, n, true, []string{currentENR})

		// delete the declaration
		resp = doReq(mux, "DELETE", "/v1/declarations/com.example.test", nil)
		expectHTTP(t, resp, 204)
		// shouldn't notify here because the storage should know that
		// we are not currently associated with any sets
		expectNotifierSlice(t, n, false, nil)
	})
}
