package storage

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jessepeterson/kmfddm/ddm"
	"github.com/jessepeterson/kmfddm/logkeys"

	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanolib/log/ctxlog"
)

var (
	ErrStatusReportNotFound = errors.New("status report not found")
)

// StatusReport is a status report together with the "write model" derived from
// it: the rows storing it produces.
type StatusReport struct {
	*ddm.StatusReport

	// The write model. These are projections of the received report, populated
	// by ParseStatusReport.
	Declarations []DeclarationStatus
	Values       []StatusValue
	Errors       []StatusError
}

// DeclarationStatus is a representation of the status of declarations.
// See https://developer.apple.com/documentation/devicemanagement/statusmanagementdeclarationsdeclarationobject
type DeclarationStatus struct {
	Identifier   string          `json:"identifier"`
	Active       bool            `json:"active"`
	Valid        string          `json:"valid"`
	ServerToken  string          `json:"server-token"`
	ManifestType string          `json:",omitempty"`
	ReasonsJSON  json.RawMessage `json:",omitempty"`
}

// DeclarationQueryStatus is additional detail given to the as-reported DeclarationStatus.
// This is populated from the storage backend's knowledge the last status update
// and current declarations.
type DeclarationQueryStatus struct {
	DeclarationStatus
	Current        bool        `json:"current"`
	StatusReceived time.Time   `json:"status_received"`
	StatusID       string      `json:"status_id,omitempty"`
	Reasons        interface{} `json:"reasons,omitempty"`
}

// StatusError is a status error as it is stored. The error itself is held as
// the JSON it is serialized to.
//
// ErrorJSON is the error marshalled from the generated schema types rather than
// the bytes the client sent: see parseReportErrors for what that costs. A caller
// wanting the report exactly as submitted can read it from /v1/status-report.
type StatusError struct {
	Path      string
	ErrorJSON []byte
}

// StatusValue is a single parsed status value as it is stored. These are,
// essentially, just key-value pairs with the path.
type StatusValue struct {
	Path          string
	ContainerType string
	ValueType     string
	Value         []byte
}

// StoredStatusError is a status error as it is read back out of storage.
type StoredStatusError struct {
	Path      string      `json:"path"`
	Error     interface{} `json:"error"`
	Timestamp time.Time   `json:"timestamp"`
	StatusID  string      `json:"status_id,omitempty"`
}

// StoredStatusValue is a status value as it is read back out of storage.
type StoredStatusValue struct {
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

	// Index is the optional "index" of this report among the enrollment's
	// status reports. Its meaning depends on the storage backend: see the
	// backend's RetrieveStatusReport documentation.
	Index int
}

// StatusReportQuery specifies search criteria for finding specific status reports for enrollments.
type StatusReportQuery struct {
	EnrollmentID string
	StatusID     *string

	// Index is reverse-chronological: 0 is the most recent status report,
	// 1 the one received before that, and so on.
	Index *int
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
	//
	// The write model of the report is expected to be populated.
	StoreDeclarationStatus(ctx context.Context, enrollmentID string, status *StatusReport) error
}

// NewStatusStoreHandler returns the default ddm.StatusHandler: it turns report
// into the write model it stores — its declaration status, its collected status
// values and its status errors — and then stores it with s.
//
// logger receives a debug line describing each stored report, and the paths of
// any status items it does not collect. It is combined with any key-value
// functions the caller attached to the context, so a logger scoped to the
// handler is a reasonable thing to pass. Pass nil for none.
//
// This is a single linear step. To do something with a status report before or
// after it is stored, or alongside it, wrap this handler in a ddm.StatusHandler
// of your own. Such a handler has the parsed report and its raw bytes, and can
// walk those raw bytes with its own jsonpath.PathMux.
//
// Panics if s is nil.
func NewStatusStoreHandler(s StatusStorer, logger log.Logger) ddm.StatusHandler {
	if s == nil {
		panic("nil status storer")
	}
	if logger == nil {
		logger = log.NopLogger
	}
	return ddm.StatusHandlerFunc(func(ctx context.Context, enrollmentID string, report *ddm.StatusReport) error {
		toStore, err := ParseStatusReport(ctx, report, logger)
		if err != nil {
			return err
		}
		logger := ctxlog.Logger(ctx, logger).With(
			logkeys.DeclarationCount, len(toStore.Declarations),
			logkeys.ErrorCount, len(toStore.Errors),
			logkeys.ValueCount, len(toStore.Values),
			logkeys.FullReport, toStore.IsFullReport(),
			logkeys.StatusID, toStore.ID,
		)
		if err = s.StoreDeclarationStatus(ctx, enrollmentID, toStore); err != nil {
			logger.Info(logkeys.Message, "storing status report", logkeys.Error, err)
			return err
		}
		logger.Debug(logkeys.Message, "stored status report")
		return nil
	})
}

type StatusDeclarationsRetriever interface {
	// RetrieveDeclarationStatus retrieves the status of the declarations for enrollmentIDs.
	RetrieveDeclarationStatus(ctx context.Context, enrollmentIDs []string) (map[string][]DeclarationQueryStatus, error)
}

type StatusErrorsRetriever interface {
	// RetrieveStatusErrors retrieves the collected errors for enrollmentIDs.
	RetrieveStatusErrors(ctx context.Context, enrollmentIDs []string, offset, limit int) (map[string][]StoredStatusError, error)
}

type StatusValuesRetriever interface {
	// RetrieveStatusErrors retrieves the collected errors for enrollmentIDs.
	RetrieveStatusValues(ctx context.Context, enrollmentIDs []string, pathPrefix string) (map[string][]StoredStatusValue, error)
}

type StatusReportRetriever interface {
	// RetrieveStatusReport retrieves the status report matching q.
	// A nil report and nil error means no report matched. Backends
	// differ in which queries they support and in the meaning of the
	// returned report's Index: see each backend's documentation.
	RetrieveStatusReport(ctx context.Context, q StatusReportQuery) (*StoredStatusReport, error)
}
