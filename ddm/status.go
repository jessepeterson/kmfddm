package ddm

import (
	"fmt"
	"time"

	"github.com/jessepeterson/kmfddm/jsonpath"
	"github.com/valyala/fastjson"
)

const (
	pathDeclarations = ".StatusItems.management.declarations"
	pathManagement   = ".StatusItems.management."
	pathDevice       = ".StatusItems.device."
	pathErrors       = ".Errors"
)

// DeclarationStatus is a representation of the status of declarations.
// See https://developer.apple.com/documentation/devicemanagement/statusmanagementdeclarationsdeclarationobject
type DeclarationStatus struct {
	Identifier   string `json:"identifier"`
	Active       bool   `json:"active"`
	Valid        string `json:"valid"`
	ServerToken  string `json:"server-token"`
	ManifestType string `json:",omitempty"`
	ReasonsJSON  []byte `json:",omitempty"`
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

// StatusValue contains parsed status values. These are, essentially,
// just key-value pairs with the path.
type StatusValue struct {
	Path          string
	ContainerType string
	ValueType     string
	Value         []byte
}

// StatusError contains parsed status errors.
type StatusError struct {
	Path      string
	ErrorJSON []byte
}

// StatusReport is the combined parsed and raw status report.
type StatusReport struct {
	ID string

	Declarations []DeclarationStatus

	Errors []StatusError

	// the "raw" status report values not otherwise parsed
	Values []StatusValue

	// the raw JSON bytes of the status report
	Raw []byte
}

func parseStatusDeclarations(v *fastjson.Value) ([]DeclarationStatus, []StatusError, error) {
	o, err := v.Object()
	if err != nil {
		return nil, nil, err
	}
	var decls []DeclarationStatus
	var errs []StatusError
	o.Visit(func(k []byte, v *fastjson.Value) {
		var a []*fastjson.Value
		a, err = v.Array()
		for _, v := range a {
			var reasonsJSON []byte
			if rV := v.Get("reasons"); rV != nil {
				reasonsJSON = rV.MarshalTo(nil)
			}
			d := DeclarationStatus{
				ManifestType: string(k),
				Identifier:   string(v.GetStringBytes("identifier")),
				Active:       v.GetBool("active"),
				Valid:        string(v.GetStringBytes("valid")),
				ServerToken:  string(v.GetStringBytes("server-token")),
				ReasonsJSON:  reasonsJSON,
			}
			decls = append(decls, d)
			// consider non-active and non-valid declarations as "errors"
			if !d.Active && d.Valid != "valid" {
				e := StatusError{
					Path:      pathDeclarations + "." + string(k),
					ErrorJSON: v.MarshalTo(nil),
				}
				errs = append(errs, e)
			}
		}
	})
	return decls, errs, err
}

func parseErrors(v *fastjson.Value) ([]StatusError, error) {
	var errs []StatusError
	a, err := v.Array()
	if err != nil {
		return nil, err
	}
	for _, v := range a {
		errs = append(errs, StatusError{
			Path:      pathErrors,
			ErrorJSON: v.MarshalTo(nil),
		})
	}
	return errs, nil
}

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

func valueHandler(s *StatusReport) jsonpath.HandlerFunc {
	return func(path string, v *fastjson.Value) ([]string, error) {
		return nil, parseStatusReportValue(v, &s.Values, path, "object")
	}
}

func declarationHandler(s *StatusReport) jsonpath.HandlerFunc {
	return func(path string, v *fastjson.Value) ([]string, error) {
		declarationStatus, declarationErrors, err := parseStatusDeclarations(v)
		s.Declarations = declarationStatus
		s.Errors = append(s.Errors, declarationErrors...)
		return nil, err
	}
}

func errorHandler(s *StatusReport) jsonpath.HandlerFunc {
	return func(path string, v *fastjson.Value) ([]string, error) {
		statusErrors, err := parseErrors(v)
		s.Errors = append(s.Errors, statusErrors...)
		return nil, err
	}
}

// RegisterStatusHandlers attaches default jsonpath Mux handlers for s to mux.
// These default handlers populate the fields of s from a status report.
func RegisterStatusHandlers(mux *jsonpath.PathMux, s *StatusReport) {
	mux.Handle(pathDeclarations, declarationHandler(s))
	mux.Handle(pathManagement, valueHandler(s))
	mux.Handle(pathDevice, valueHandler(s))
	mux.Handle(pathErrors, errorHandler(s))
}

// ParseStatusUsingMux parses the raw status report from a DDM client using mux.
func ParseStatusUsingMux(raw []byte, mux *jsonpath.PathMux) ([]string, error) {
	if mux == nil {
		panic("mux is nil")
	}
	v, err := fastjson.ParseBytes(raw)
	if err != nil {
		return nil, fmt.Errorf("parsing json: %w", err)
	}
	unhandled, err := mux.JSONPath("", v)
	return unhandled, err
}

// ParseStatus parses the raw status report from a DDM client using mux.
func ParseStatus(raw []byte) ([]string, *StatusReport, error) {
	s := &StatusReport{Raw: raw}
	mux := jsonpath.NewPathMux()
	RegisterStatusHandlers(mux, s)
	unhandled, err := ParseStatusUsingMux(s.Raw, mux)
	return unhandled, s, err
}
