package storage

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/jessepeterson/kmfddm/ddm"
	ddmstatus "github.com/jessepeterson/kmfddm/ddm/status"

	"github.com/micromdm/nanolib/log"
)

const (
	// statusFile1 is a status report from a macOS device.
	statusFile1 = "../ddm/testdata/A047820F-FC6B-4104-BED0-466876D82BB8.20220628-103225.001918.status.json"

	// statusFile2 is a "full report" (FullReport: true) that reports a
	// top-level error and whose declaration status keys are deliberately out
	// of order: declarations are always parsed in a fixed order regardless of
	// the order the client sent them in.
	statusFile2 = "../ddm/testdata/report-with-errors.status.json"
)

// parseReport parses raw into a ddm.StatusReport the way the HTTP status report
// handler does: strictly, against the generated DDM status structures.
func parseReport(raw []byte) (*ddm.StatusReport, error) {
	r := &ddm.StatusReport{Raw: raw, Status: new(ddmstatus.StatusReport)}
	err := json.Unmarshal(raw, r.Status)
	return r, err
}

// parseReportFile parses the status report in file and turns it into its write
// model, the way the default store handler does.
func parseReportFile(t *testing.T, file string) *StatusReport {
	t.Helper()
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	report, err := parseReport(raw)
	if err != nil {
		t.Fatal(err)
	}
	toStore, err := ParseStatusReport(context.Background(), report, log.NopLogger)
	if err != nil {
		t.Fatal(err)
	}
	return toStore
}

// TestStatusParseStruct asserts the report is reachable parsed against the
// generated status structures, that the write model is derived from those same
// structures, and that every declaration in a real captured report is read.
func TestStatusParseStruct(t *testing.T) {
	s := parseReportFile(t, statusFile1)
	if s.Status == nil {
		t.Fatal("no parsed status report")
	}

	device := s.Status.StatusItems.Device
	if device == nil || device.Model == nil {
		t.Fatal("no device model status item")
	}
	if have, want := *device.Model.Family, "iPhone"; have != want {
		t.Errorf("device model family: have: %q, want: %q", have, want)
	}

	// a real device report carries no top-level errors
	if want := 0; len(s.Errors) != want {
		t.Errorf("errors: have: %d, want: %d", len(s.Errors), want)
	}

	// the same declarations are reachable through both forms. this report has
	// an activation, two configurations, an asset and a management declaration.
	if want := 5; len(s.Declarations) != want {
		t.Fatalf("declaration count: have: %d, want: %d", len(s.Declarations), want)
	}
	if d := s.Status.StatusItems.Management.Declarations; d == nil {
		t.Fatal("no declaration status")
	} else if have, want := len(d.Activations)+len(d.Configurations)+len(d.Assets)+len(d.Management), 5; have != want {
		t.Errorf("declaration count from status: have: %d, want: %d", have, want)
	}

	// declarations are stored as declaration status, never as loose pairs
	for _, v := range s.Values {
		if strings.HasPrefix(v.Path, ".StatusItems.management.declarations") {
			t.Errorf("declaration leaked into status values: %s", v.Path)
		}
	}
}

// TestStatusParseDeclarations asserts declarations are parsed from the
// generated structures, in a fixed order, with the client's reasons.
func TestStatusParseDeclarations(t *testing.T) {
	s := parseReportFile(t, statusFile2)

	// note the report declares its keys as management, assets, configurations,
	// activations: they are always read back in this order.
	want := []DeclarationStatus{
		{ManifestType: "activations", Identifier: "com.example.activation", Active: true, Valid: "valid", ServerToken: "5e6f7a8b-9c0d-1e2f-3a4b-5c6d7e8f9a01"},
		{ManifestType: "configurations", Identifier: "com.example.configuration", Active: true, Valid: "invalid", ServerToken: "1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d",
			ReasonsJSON: json.RawMessage(`[{"code":"Error.ConfigurationCannotBeApplied","description":"Configuration cannot be applied","details":{"Error":"An internal error has occurred."}}]`)},
		{ManifestType: "configurations", Identifier: "com.example.unactivated", Active: false, Valid: "unknown", ServerToken: "2b3c4d5e-6f7a-8b9c-0d1e-2f3a4b5c6d7e",
			ReasonsJSON: json.RawMessage(`[{"code":"Info.NotReferencedByActivation","description":"Configuration is not referenced by an activation.","details":{"Identifier":"com.example.unactivated","ServerToken":"2b3c4d5e-6f7a-8b9c-0d1e-2f3a4b5c6d7e"}}]`)},
		{ManifestType: "assets", Identifier: "com.example.asset", Active: true, Valid: "valid", ServerToken: "3c4d5e6f-7a8b-9c0d-1e2f-3a4b5c6d7e8f"},
		{ManifestType: "management", Identifier: "com.example.management", Active: true, Valid: "valid", ServerToken: "4d5e6f7a-8b9c-0d1e-2f3a-4b5c6d7e8f90"},
	}
	if !reflect.DeepEqual(s.Declarations, want) {
		t.Errorf("declarations: have: %v, want: %v", s.Declarations, want)
	}
}

// TestStatusParseDeclarationNull asserts a null element in a declaration array
// is skipped rather than dereferenced.
//
// The generated declaration arrays are slices of pointers, and a JSON null
// unmarshals to a nil element with no error, so this used to panic inside
// parseDeclarations. In the e2e suite a panic here surfaces as a dead
// connection rather than a clean error.
func TestStatusParseDeclarationNull(t *testing.T) {
	for _, tt := range []struct {
		name string
		raw  string
	}{
		{"null activation", `{"StatusItems":{"management":{"declarations":{"activations":[null]}}}}`},
		{"null configuration", `{"StatusItems":{"management":{"declarations":{"configurations":[null]}}}}`},
		{"null asset", `{"StatusItems":{"management":{"declarations":{"assets":[null]}}}}`},
		{"null management", `{"StatusItems":{"management":{"declarations":{"management":[null]}}}}`},
		{"null among declarations", `{"StatusItems":{"management":{"declarations":{"configurations":[{"identifier":"com.example.a","active":true,"valid":"valid"},null]}}}}`},
		{"null declarations block", `{"StatusItems":{"management":{"declarations":null}}}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// the unmarshal is expected to succeed: null is not a type error
			report, err := parseReport([]byte(tt.raw))
			if err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			toStore, err := ParseStatusReport(context.Background(), report, log.NopLogger)
			if err != nil {
				t.Fatalf("ParseStatusReport: %v", err)
			}

			// only the declarations we can actually describe survive
			if len(toStore.Declarations) > 1 {
				t.Errorf("declarations: have: %d, want at most 1", len(toStore.Declarations))
			}
			for _, d := range toStore.Declarations {
				if d.Identifier == "" {
					t.Error("declaration with no identifier")
				}
			}
		})
	}
}

// TestStatusIsFullReport asserts the client's "full report" flag is readable.
func TestStatusIsFullReport(t *testing.T) {
	for _, tt := range []struct {
		file string
		want bool
	}{
		{file: statusFile2, want: true},  // FullReport: true
		{file: statusFile1, want: false}, // FullReport absent
	} {
		s := parseReportFile(t, tt.file)
		if have := s.IsFullReport(); have != tt.want {
			t.Errorf("%s: full report: have: %v, want: %v", tt.file, have, tt.want)
		}
	}
}

// TestStatusParseStrict asserts that a status report which does not conform to
// the generated status structures is rejected outright. The loose status
// values are collected leniently, but a declaration that cannot be understood
// is not silently coerced.
//
// This is the executable spec of what a device may and may not send. Every case
// here fails in encoding/json, before any backend is involved.
func TestStatusParseStrict(t *testing.T) {
	for _, tt := range []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{
			name:    "non-bool active",
			raw:     `{"StatusItems":{"management":{"declarations":{"configurations":[{"active":"yes"}]}}}}`,
			wantErr: true,
		},
		{
			name:    "non-array declarations",
			raw:     `{"StatusItems":{"management":{"declarations":{"configurations":{}}}}}`,
			wantErr: true,
		},
		{
			name:    "non-base64 push token",
			raw:     `{"StatusItems":{"mdm":{"push-token":"not base64!"}}}`,
			wantErr: true,
		},
		{
			name:    "string for int",
			raw:     `{"StatusItems":{"content-cache":{"status":{"port":"8080"}}}}`,
			wantErr: true,
		},
		{
			name: "null valid",
			raw:  `{"StatusItems":{"management":{"declarations":{"configurations":[{"valid":null,"active":true}]}}}}`,
		},
		{
			name: "unknown status item",
			raw:  `{"StatusItems":{"management":{"not-a-status-item":{"whatever":"ignored"}}}}`,
		},
		{
			name:    "not JSON",
			raw:     `{`,
			wantErr: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			report, err := parseReport([]byte(tt.raw))
			if have := err != nil; have != tt.wantErr {
				t.Fatalf("error: have: %v (%v), want error: %v", have, err, tt.wantErr)
			}
			if err != nil {
				return
			}
			// the report is always populated, even when it is mostly empty.
			if report.Status == nil {
				t.Error("no parsed status report")
			}
		})
	}
}

// TestUnhandledStatusPaths asserts the value collector reports the paths it does
// not walk, so that a status item namespace this package does not collect values
// for is visible in the logs. That return value is only ever logged: nothing
// serves it.
func TestUnhandledStatusPaths(t *testing.T) {
	// a real report is walked entirely: both collected namespaces are ours, so
	// nothing is left over.
	raw, err := os.ReadFile(statusFile1)
	if err != nil {
		t.Fatal(err)
	}
	r := &StatusReport{StatusReport: &ddm.StatusReport{Raw: raw}}
	if unhandled, err := r.collectStatusValues(); err != nil {
		t.Fatal(err)
	} else if len(unhandled) > 0 {
		t.Errorf("unhandled paths: have: %v, want none", unhandled)
	}

	// namespaces we do not walk are reported, in the order the client sent them
	raw = []byte(`{
		"StatusItems": {
			"passcode": {"is-present": true},
			"account": {"list": {"caldav": [{"identifier": "acct1", "removed": true}]}}
		},
		"Errors": []
	}`)
	r = &StatusReport{StatusReport: &ddm.StatusReport{Raw: raw}}
	unhandled, err := r.collectStatusValues()
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Values) > 0 {
		t.Errorf("values: have: %d, want none", len(r.Values))
	}
	if have, want := unhandled, []string{".StatusItems.passcode", ".StatusItems.account"}; !reflect.DeepEqual(have, want) {
		t.Errorf("unhandled paths: have: %v, want: %v", have, want)
	}
}

// TestParseStatusReportInput asserts the transform tolerates a report that
// carries no parsed structures, and rejects a missing report outright rather
// than panicking on the embedded pointer.
func TestParseStatusReportInput(t *testing.T) {
	t.Run("unparsed", func(t *testing.T) {
		// a report that was never passed through json.Unmarshal: no
		// declarations, no errors, but values are still collected from Raw.
		toStore, err := ParseStatusReport(context.Background(), &ddm.StatusReport{
			Raw: []byte(`{"StatusItems":{"device":{"model":{"family":"iPhone"}}}}`),
		}, log.NopLogger)
		if err != nil {
			t.Fatal(err)
		}
		if len(toStore.Declarations) != 0 || len(toStore.Errors) != 0 {
			t.Errorf("unexpected write model: %d declarations, %d errors",
				len(toStore.Declarations), len(toStore.Errors))
		}
		if want := 1; len(toStore.Values) != want {
			t.Errorf("values: have: %d, want: %d", len(toStore.Values), want)
		}
	})

	t.Run("nil", func(t *testing.T) {
		if _, err := ParseStatusReport(context.Background(), nil, log.NopLogger); err == nil {
			t.Error("expected an error, got none")
		}
	})
}

// TestParseStatusReportFullReport asserts the write model carries through the
// received report's "full report" flag.
func TestParseStatusReportFullReport(t *testing.T) {
	for _, tt := range []struct {
		name string
		raw  string
		want bool
	}{
		{name: "true", raw: `{"StatusItems":{},"Errors":[],"FullReport":true}`, want: true},
		{name: "false", raw: `{"StatusItems":{},"Errors":[],"FullReport":false}`},
		{name: "absent", raw: `{"StatusItems":{},"Errors":[]}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			report, err := parseReport([]byte(tt.raw))
			if err != nil {
				t.Fatal(err)
			}
			toStore, err := ParseStatusReport(context.Background(), report, log.NopLogger)
			if err != nil {
				t.Fatal(err)
			}
			if have := toStore.IsFullReport(); have != tt.want {
				t.Errorf("full report: have: %v, want: %v", have, tt.want)
			}
		})
	}
}
