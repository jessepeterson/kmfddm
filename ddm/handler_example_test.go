package ddm_test

import (
	"context"
	"fmt"
	"hash"
	"hash/fnv"
	"net/http/httptest"
	"strings"

	"github.com/jessepeterson/kmfddm/ddm"
	httpddm "github.com/jessepeterson/kmfddm/http/ddm"
	"github.com/jessepeterson/kmfddm/jsonpath"
	"github.com/jessepeterson/kmfddm/storage"
	"github.com/jessepeterson/kmfddm/storage/inmem"

	"github.com/micromdm/nanolib/log"
	"github.com/valyala/fastjson"
)

// A status report carrying a FileVault status item.
const reportJSON = `{
  "StatusItems": {
    "diskmanagement": {
      "filevault": {"enabled": true}
    }
  },
  "Errors": []
}`

// ExampleStatusHandler wraps the default handler to do something of your own with
// each status report.
//
// A wrapper takes the next handler and returns a handler, so wrappers compose:
// nest them to stack behavior. Where you call next decides when your work runs.
// Calling it first, as below, runs your work after the report has been stored —
// what you want for an action that should only follow a successful store.
// Calling it last runs your work first, before storage.
func ExampleStatusHandler() {
	statusStorer := inmem.New(func() hash.Hash { return fnv.New128() })
	store := storage.NewStatusStoreHandler(statusStorer, nil)

	// a wrapper: takes the next handler, returns a handler
	logFileVault := func(next ddm.StatusHandler) ddm.StatusHandler {
		return ddm.StatusHandlerFunc(func(ctx context.Context, enrollmentID string, report *ddm.StatusReport) error {
			if err := next.HandleStatusReport(ctx, enrollmentID, report); err != nil {
				return err
			}

			// The report is parsed against Apple's schemas, so a status item the
			// schema covers is already a field.
			if fv := report.Status.StatusItems.DiskManagement; fv != nil &&
				fv.FileVault != nil && fv.FileVault.Enabled != nil {
				fmt.Println("filevault enabled:", *fv.FileVault.Enabled)
			}
			return nil
		})
	}

	req := httptest.NewRequest("PUT", "/status", strings.NewReader(reportJSON))
	req.Header.Set(httpddm.EnrollmentIDHeader, "ABC123")
	rec := httptest.NewRecorder()

	httpddm.NewStatusReportHandler(logFileVault(store), log.NopLogger).ServeHTTP(rec, req)

	if rec.Code != 200 {
		fmt.Println("unexpected response:", rec.Code)
		return
	}
	fmt.Println("response:", rec.Code)

	// Output:
	// filevault enabled: true
	// response: 200
}

// ExampleStatusHandler_raw wraps the default handler the same way, but reads the
// FileVault status from the report's raw bytes with a jsonpath.PathMux instead
// of a typed field.
//
// Prefer the typed field in ExampleStatusHandler whenever the generated
// structures cover what you need. Walking the raw bytes is for status items the
// schema doesn't model, or a shape it doesn't express — at the cost of a second
// parse of the report.
func ExampleStatusHandler_raw() {
	statusStorer := inmem.New(func() hash.Hash { return fnv.New128() })
	store := storage.NewStatusStoreHandler(statusStorer, nil)

	// a wrapper that reads the report's raw bytes instead of a typed field
	logFileVaultRaw := func(next ddm.StatusHandler) ddm.StatusHandler {
		return ddm.StatusHandlerFunc(func(ctx context.Context, enrollmentID string, report *ddm.StatusReport) error {
			if err := next.HandleStatusReport(ctx, enrollmentID, report); err != nil {
				return err
			}

			v, err := fastjson.ParseBytes(report.Raw)
			if err != nil {
				return err
			}
			var fileVault bool
			mux := jsonpath.NewPathMux()
			mux.HandleFunc(".StatusItems.diskmanagement.filevault.enabled.", func(_ string, v *fastjson.Value) ([]string, error) {
				fileVault = v.GetBool()
				return nil, nil
			})
			if _, err = mux.JSONPath("", v); err != nil {
				return err
			}
			fmt.Println("filevault enabled:", fileVault)

			return nil
		})
	}

	req := httptest.NewRequest("PUT", "/status", strings.NewReader(reportJSON))
	req.Header.Set(httpddm.EnrollmentIDHeader, "ABC123")
	rec := httptest.NewRecorder()

	httpddm.NewStatusReportHandler(logFileVaultRaw(store), log.NopLogger).ServeHTTP(rec, req)

	if rec.Code != 200 {
		fmt.Println("unexpected response:", rec.Code)
		return
	}
	fmt.Println("response:", rec.Code)

	// Output:
	// filevault enabled: true
	// response: 200
}
