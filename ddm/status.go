package ddm

import (
	"fmt"
	"time"

	"github.com/valyala/fastjson"
)

const (
	pathDeclarations = ".StatusItems.management.declarations"
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
	Declarations []DeclarationStatus

	Errors []StatusError

	// the "raw" status report values not otherwise parsed
	Values []StatusValue

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

func parseStatusReportValue(v *fastjson.Value, s *StatusReport, path, container string) error {
	switch path {
	case pathDeclarations:
		var err error
		var errs []StatusError
		s.Declarations, errs, err = parseStatusDeclarations(v)
		s.Errors = append(s.Errors, errs...)
		return err
	case pathErrors:
		errs, err := parseErrors(v)
		s.Errors = append(s.Errors, errs...)
		return err
	}
	vType := v.Type()
	switch vType {
	case fastjson.TypeObject:
		o, err := v.Object()
		if err != nil {
			return err
		}
		o.Visit(func(k []byte, v *fastjson.Value) {
			newPath := path + "." + string(k)
			err = parseStatusReportValue(v, s, newPath, "object")
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
			err = parseStatusReportValue(v, s, path, "array")
			if err != nil {
				return err
			}
		}
	case fastjson.TypeString:
		s.Values = append(s.Values, StatusValue{
			Path:          path,
			ContainerType: container,
			ValueType:     "string",
			Value:         v.GetStringBytes(),
		})
	case fastjson.TypeNumber:
		s.Values = append(s.Values, StatusValue{
			Path:          path,
			ContainerType: container,
			ValueType:     "number",
			Value:         v.MarshalTo(nil),
		})
	case fastjson.TypeTrue:
		s.Values = append(s.Values, StatusValue{
			Path:          path,
			ContainerType: container,
			ValueType:     "boolean",
			Value:         []byte("true"),
		})
	case fastjson.TypeFalse:
		s.Values = append(s.Values, StatusValue{
			Path:          path,
			ContainerType: container,
			ValueType:     "boolean",
			Value:         []byte("false"),
		})
	}
	return nil
}

// ParseStatus parses the status report from a DDM client.
func ParseStatus(raw []byte) (*StatusReport, error) {
	v, err := fastjson.ParseBytes(raw)
	if err != nil {
		return nil, fmt.Errorf("parsing json: %w", err)
	}
	s := &StatusReport{Raw: raw}
	return s, parseStatusReportValue(v, s, "", "root")
}
