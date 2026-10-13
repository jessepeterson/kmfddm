package ddm

import (
	"context"

	ddmstatus "github.com/jessepeterson/kmfddm/ddm/status"
)

// StatusReport is a status report as it was received from a DDM client.
type StatusReport struct {
	// ID is the unique identifier of this report, if one was assigned.
	ID string

	// Status is the report parsed against the generated DDM status structures.
	//
	// Parsing uses the ordinary type checking of the json package: a value
	// whose JSON type does not match the structure fails the unmarshal.
	// Unknown keys, missing keys and nulls are not errors.
	Status *ddmstatus.StatusReport

	// Raw is the raw JSON bytes of the report.
	Raw []byte
}

// IsFullReport reports whether the client flagged this status report as
// containing the complete set of current status, rather than only the items
// that changed since the last report. A full report is sent periodically as a
// "safety sync".
// See https://developer.apple.com/documentation/devicemanagement/statusreport/fullreport
func (r *StatusReport) IsFullReport() bool {
	return r.Status != nil && r.Status.FullReport != nil && *r.Status.FullReport
}

// StatusHandler handles a status report received from a DDM client.
//
// A StatusReport has already been parsed by the time it reaches a handler. Any
// further detail about the request that carried it is available from ctx, or,
// where a handler cannot reasonably do without it, from the arguments.
type StatusHandler interface {
	HandleStatusReport(ctx context.Context, enrollmentID string, report *StatusReport) error
}

// StatusHandlerFunc adapts a function to a StatusHandler.
type StatusHandlerFunc func(ctx context.Context, enrollmentID string, report *StatusReport) error

// HandleStatusReport calls f(ctx, enrollmentID, report).
func (f StatusHandlerFunc) HandleStatusReport(ctx context.Context, enrollmentID string, report *StatusReport) error {
	return f(ctx, enrollmentID, report)
}
