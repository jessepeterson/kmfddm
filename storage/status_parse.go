package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jessepeterson/kmfddm/ddm"
	ddmstatus "github.com/jessepeterson/kmfddm/ddm/status"
	"github.com/jessepeterson/kmfddm/jsonpath"
	"github.com/jessepeterson/kmfddm/logkeys"
	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanolib/log/ctxlog"
	"github.com/valyala/fastjson"
)

// Paths within a DDM status report that the default value collector walks. A
// path ending in a period is a "wildcard" path: the collector descends beneath
// it.
const (
	pathDeclarations = ".StatusItems.management.declarations"
	pathManagement   = ".StatusItems.management."
	pathDevice       = ".StatusItems.device."
	pathErrors       = ".Errors"
)

// declarationValid is the "valid" value of a declaration status that reports a
// successfully processed declaration.
const declarationValid = "valid"

// ParseStatusReport turns a received ddm.StatusReport into the write model the
// storage layer stores: its declaration status, its collected status values,
// and its status errors.
//
// The declaration status and the report's own errors are read from the received
// report, so they are whatever the json package parsed into the generated
// structures. The status values are collected differently, by walking the raw
// report: a status item this package does not model is collected or skipped on
// the shape of the JSON alone, and never an error. Paths the walk does not
// collect are logged to logger.
//
// Returns nil on error.
func ParseStatusReport(ctx context.Context, report *ddm.StatusReport, logger log.Logger) (*StatusReport, error) {
	if report == nil {
		return nil, errors.New("nil status report")
	}
	if logger == nil {
		logger = log.NopLogger
	}

	toStore := &StatusReport{StatusReport: report}
	toStore.parseDeclarations()
	toStore.parseReportErrors()

	unhandled, err := toStore.collectStatusValues()
	if err != nil {
		return nil, err
	}

	ctxLogger := ctxlog.Logger(ctx, logger)
	for _, u := range unhandled {
		ctxLogger.Debug(logkeys.Message, "unhandled status path", "path", u)
	}
	return toStore, nil
}

// collectStatusValues fills r.Values from the report's raw bytes, and returns
// the paths of the report that were not walked.
func (r *StatusReport) collectStatusValues() ([]string, error) {
	v, err := fastjson.ParseBytes(r.Raw)
	if err != nil {
		return nil, fmt.Errorf("parsing json: %w", err)
	}

	var values []StatusValue
	mux := jsonpath.NewPathMux()

	// The declaration status and the report's own errors are stored in their own
	// structured form. Sinking them here keeps them from being collected a
	// second time as loose key-value pairs.
	mux.Handle(pathDeclarations, sinkHandler())
	mux.Handle(pathErrors, sinkHandler())
	mux.Handle(pathManagement, valueHandler(&values))
	mux.Handle(pathDevice, valueHandler(&values))

	unhandled, err := mux.JSONPath("", v)
	if err != nil {
		return nil, err
	}

	r.Values = values
	return unhandled, nil
}

// valueHandler returns a jsonpath handler that collects the values it is given
// into values. It is a "wildcard" handler: register it for a path ending in a
// period to collect the leaves of every path beneath it.
func valueHandler(values *[]StatusValue) jsonpath.HandlerFunc {
	return func(path string, v *fastjson.Value) ([]string, error) {
		return nil, parseStatusReportValue(v, values, path, "object")
	}
}

// sinkHandler returns a jsonpath handler that consumes the value it is given
// without recording anything. Registering one for a path prevents the wildcard
// handlers above it from descending into that subtree.
func sinkHandler() jsonpath.HandlerFunc {
	return func(string, *fastjson.Value) ([]string, error) { return nil, nil }
}

// parseStatusReportValue walks the JSON value v, collecting its scalar leaves
// into values. path is the path v was found at and container is the kind of
// container v was found in ("object" or "array"), which is recorded as the
// container type of each value found.
//
// The walk is generic: it does not know anything about status reports. Arrays
// are flattened — each element shares the path of the array — and objects and
// objects-within-arrays alike produce a path per leaf.
func parseStatusReportValue(v *fastjson.Value, values *[]StatusValue, path, container string) error {
	switch v.Type() {
	case fastjson.TypeObject:
		o, err := v.Object()
		if err != nil {
			return err
		}
		o.Visit(func(k []byte, v *fastjson.Value) {
			newPath := path + "." + string(k)
			err = parseStatusReportValue(v, values, newPath, "object")
		})
		if err != nil {
			return err
		}
	case fastjson.TypeArray:
		a, err := v.Array()
		if err != nil {
			return err
		}
		for _, v := range a {
			err = parseStatusReportValue(v, values, path, "array")
			if err != nil {
				return err
			}
		}
	case fastjson.TypeString:
		*values = append(*values, StatusValue{
			Path:          path,
			ContainerType: container,
			ValueType:     "string",
			Value:         v.GetStringBytes(),
		})
	case fastjson.TypeNumber:
		*values = append(*values, StatusValue{
			Path:          path,
			ContainerType: container,
			ValueType:     "number",
			Value:         v.MarshalTo(nil),
		})
	case fastjson.TypeTrue:
		*values = append(*values, StatusValue{
			Path:          path,
			ContainerType: container,
			ValueType:     "boolean",
			Value:         []byte("true"),
		})
	case fastjson.TypeFalse:
		*values = append(*values, StatusValue{
			Path:          path,
			ContainerType: container,
			ValueType:     "boolean",
			Value:         []byte("false"),
		})
	}
	return nil
}

// addDeclaration appends the status of a single processed declaration to r.
// The declaration itself is only used (marshaled) should the declaration be
// flagged as an error below.
func (r *StatusReport) addDeclaration(manifestType, identifier, serverToken string, active bool, valid string, reasons *[]*ddmstatus.ManagementDeclarationsActivationsReasons, declaration any) {
	d := DeclarationStatus{
		ManifestType: manifestType,
		Identifier:   identifier,
		Active:       active,
		Valid:        valid,
		ServerToken:  serverToken,
	}
	// marshaling cannot fail for this type; leave ReasonsJSON nil (omitted) if
	// the client did not report any reasons.
	if reasons != nil {
		d.ReasonsJSON, _ = json.Marshal(reasons)
	}
	r.Declarations = append(r.Declarations, d)

	// consider non-active and non-valid declarations as "errors"
	if !d.Active && d.Valid != declarationValid {
		// the declaration is marshaled back out of the generated structures.
		// should never fail: the values came out of a successful unmarshal.
		errJSON, err := json.Marshal(declaration)
		if err == nil {
			r.Errors = append(r.Errors, StatusError{
				Path:      pathDeclarations + "." + manifestType,
				ErrorJSON: errJSON,
			})
		}
	}
}

// parseDeclarations fills r.Declarations, and r.Errors, from the declaration
// status of the received report.
func (r *StatusReport) parseDeclarations() {
	if r.Status == nil {
		return
	}
	management := r.Status.StatusItems.Management
	if management == nil || management.Declarations == nil {
		return
	}
	d := management.Declarations
	for _, decl := range d.Activations {
		r.addDeclaration("activations", decl.Identifier, decl.ServerToken, decl.Active, string(decl.Valid), decl.Reasons, decl)
	}
	for _, decl := range d.Configurations {
		r.addDeclaration("configurations", decl.Identifier, decl.ServerToken, decl.Active, string(decl.Valid), decl.Reasons, decl)
	}
	for _, decl := range d.Assets {
		r.addDeclaration("assets", decl.Identifier, decl.ServerToken, decl.Active, string(decl.Valid), decl.Reasons, decl)
	}
	for _, decl := range d.Management {
		r.addDeclaration("management", decl.Identifier, decl.ServerToken, decl.Active, string(decl.Valid), decl.Reasons, decl)
	}
}

// parseReportErrors appends a StatusError to r for each error the client
// reported at the top level of the received report.
//
// The stored JSON is the error marshalled back out of the generated schema
// types, not the bytes the client sent, so it is a model of the error rather
// than a copy of it. For anything matching Apple's schema that is lossless in
// substance; the differences are:
//
//   - keys come back in Go struct order (and sorted within Details)
//   - unmodelled keys are dropped: an error carrying a key the generated types
//     do not declare is stored without it, which for a wholly off-schema error
//     leaves an empty object
//   - numbers inside Details pass through float64, so a large integer loses
//     precision
//
// Callers needing the report exactly as submitted can read it back from
// /v1/status-report, which serves the stored report unchanged.
func (r *StatusReport) parseReportErrors() {
	if r.Status == nil {
		return
	}
	for _, e := range r.Status.Errors {
		errJSON, err := json.Marshal(e)
		if err != nil {
			continue
		}
		r.Errors = append(r.Errors, StatusError{
			Path:      pathErrors,
			ErrorJSON: errJSON,
		})
	}
}
