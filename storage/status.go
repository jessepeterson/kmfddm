package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jessepeterson/kmfddm/ddm"
)

var (
	ErrStatusReportNotFound = errors.New("status report not found")
)

type StatusError struct {
	Path      string      `json:"path"`
	Error     interface{} `json:"error"`
	Timestamp time.Time   `json:"timestamp"`
	StatusID  string      `json:"status_id,omitempty"`
}

type StatusValue struct {
	Path      string    `json:"path"`
	Value     string    `json:"value"`
	Timestamp time.Time `json:"timestamp"`
	StatusID  string    `json:"status_id,omitempty"`
}

// StoredStatusReport represents a stored status report by StoreDeclarationStatus.
type StoredStatusReport struct {
	Raw       []byte    // the raw JSON bytes of the status report
	Timestamp time.Time // the date the status report was saved
	StatusID  string    // optional unique identifier of report. defined when report was saved.
	Index     int       // optional "index" for this enrollment's status reports.
}

// StatusReportQuery specifies search criteria for finding specific status reports for enrollments.
type StatusReportQuery struct {
	EnrollmentID string
	StatusID     *string
	Index        *int
}

// Valid performs basic sanity checks for querying for status reports.
func (q StatusReportQuery) Valid() error {
	if q.EnrollmentID == "" {
		return errors.New("missing enrollment ID")
	} else if (q.StatusID == nil || *q.StatusID == "") && q.Index == nil {
		return errors.New("status ID and index cannot both be empty")
	}
	return nil
}

type StatusStorer interface {
	// StoreDeclarationStatus stores the status report details.
	// For later retrieval by the StatusAPIStorage interface(s).
	StoreDeclarationStatus(ctx context.Context, enrollmentID string, status *ddm.StatusReport) error
}

type StatusDeclarationsRetriever interface {
	// RetrieveDeclarationStatus retrieves the status of the declarations for enrollmentIDs.
	RetrieveDeclarationStatus(ctx context.Context, enrollmentIDs []string) (map[string][]ddm.DeclarationQueryStatus, error)
}

type StatusErrorsRetriever interface {
	// RetrieveStatusErrors retrieves the collected errors for enrollmentIDs.
	RetrieveStatusErrors(ctx context.Context, enrollmentIDs []string, offset, limit int) (map[string][]StatusError, error)
}

type StatusValuesRetriever interface {
	// RetrieveStatusErrors retrieves the collected errors for enrollmentIDs.
	RetrieveStatusValues(ctx context.Context, enrollmentIDs []string, pathPrefix string) (map[string][]StatusValue, error)
}

type StatusReportRetriever interface {
	RetrieveStatusReport(ctx context.Context, q StatusReportQuery) (*StoredStatusReport, error)
}
