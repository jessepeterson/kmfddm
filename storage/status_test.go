package storage

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jessepeterson/kmfddm/ddm"
	ddmstatus "github.com/jessepeterson/kmfddm/ddm/status"

	"github.com/micromdm/nanolib/log"
)

// mockStatusStorer records the status report it was asked to store.
type mockStatusStorer struct {
	storeErr error

	called        bool
	enrollmentID  string
	report        *StatusReport
	ctxWasPresent bool
}

func (s *mockStatusStorer) StoreDeclarationStatus(ctx context.Context, enrollmentID string, status *StatusReport) error {
	s.called = true
	s.enrollmentID = enrollmentID
	s.report = status
	s.ctxWasPresent = ctx != nil
	return s.storeErr
}

// TestNewStatusStoreHandler asserts what the store handler hands the storer,
// and what it refuses to hand over at all.
func TestNewStatusStoreHandler(t *testing.T) {
	raw, err := os.ReadFile(statusFile1)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("write model", func(t *testing.T) {
		storer := new(mockStatusStorer)
		h := NewStatusStoreHandler(storer, log.NopLogger)

		report, err := parseReport(raw)
		if err != nil {
			t.Fatal(err)
		}
		report.ID = "status-id"

		ctx := context.Background()
		if err = h.HandleStatusReport(ctx, "ABC123", report); err != nil {
			t.Fatal(err)
		}
		if !storer.called {
			t.Fatal("status report was not stored")
		}
		if have, want := storer.enrollmentID, "ABC123"; have != want {
			t.Errorf("enrollment ID: have: %q, want: %q", have, want)
		}
		if storer.report == nil {
			t.Fatal("no write model")
		}
		// the storer gets the write model, and it still carries the received
		// report through.
		if storer.report.StatusReport != report {
			t.Errorf("report: have: %p, want: %p", storer.report.StatusReport, report)
		}
		if !storer.ctxWasPresent {
			t.Error("context was not passed through to the storer")
		}
	})

	t.Run("store error", func(t *testing.T) {
		storeErr := errors.New("no storage for you")
		storer := &mockStatusStorer{storeErr: storeErr}

		report, err := parseReport(raw)
		if err != nil {
			t.Fatal(err)
		}

		h := NewStatusStoreHandler(storer, log.NopLogger)
		if err = h.HandleStatusReport(context.Background(), "ABC123", report); !errors.Is(err, storeErr) {
			t.Errorf("error: have: %v, want: %v", err, storeErr)
		}
	})

	t.Run("unreadable values", func(t *testing.T) {
		storer := new(mockStatusStorer)

		// a report whose raw bytes are not JSON cannot have its values
		// collected, so it is never stored.
		report := &ddm.StatusReport{Status: new(ddmstatus.StatusReport), Raw: []byte("{")}
		h := NewStatusStoreHandler(storer, log.NopLogger)
		if err := h.HandleStatusReport(context.Background(), "ABC123", report); err == nil {
			t.Error("expected an error, got none")
		}
		if storer.called {
			t.Error("status report should not have been stored")
		}
	})
}
