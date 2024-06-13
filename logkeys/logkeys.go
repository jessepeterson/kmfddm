// Package logkeys defines some static logging keys for consistent structured logging output.
// Mostly exists as a mental aid when drafting log messages.
package logkeys

const (
	Message = "msg" // type: string
	Error   = "err" // type: error

	// an MDM enrollment ID. i.e. a UDID, EnrollmentID, etc.
	EnrollmentID = "id" // type: string

	// in cases where we might need to log multiple enrollment IDs but only
	// want to log the first (to avoid massive lists in logs).
	FirstEnrollmentID = "id_first" // type: string

	// MDM "v1" command UUID
	CommandUUID = "command_uuid" // type: string

	// a context-dependent numerical count/length of something
	GenericCount = "count" // type: int

	DeclarationID   = "declaration_id"   // type: string
	DeclarationType = "declaration_type" // type: string

	// status report type counts
	DeclarationCount = "declaration_count" // type: int
	ErrorCount       = "error_count"       // type: int
	ValueCount       = "value_count"       // type: int

	// HTTP handler
	Handler = "handler" // type: string

	Changed = "changed" // type: bool
	Notify  = "notify"  // type: bool
)
