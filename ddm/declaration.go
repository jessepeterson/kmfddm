// Package ddm supports the core Apple Declarative Device Management protocol.
package ddm

import (
	"errors"
	"fmt"

	"github.com/valyala/fastjson"
)

var ErrInvalidDeclaration = errors.New("invalid declaration")

type Declaration struct {
	Identifier  string
	Type        string
	PayloadJSON []byte `json:"-"`
	ServerToken string `json:",omitempty"`

	// References to to other declaration identifiers parsed
	// from this declaration's payload.
	IdentifierRefs []string `json:"-"`

	Raw []byte `json:"-"`
}

// Valid performs basic sanity checks.
func (d *Declaration) Valid() bool {
	if d == nil || d.Identifier == "" || d.Type == "" {
		return false
	}
	if len(d.PayloadJSON) < 2 {
		// a valid object must at least be `{}`
		return false
	}
	return true
}

// findIDRefs traverses the payload using IdentifierRefs to find any dependent declarations.
func findIDRefs(v *fastjson.Value, declarationType string) []string {
	if _, ok := IdentifierRefs[declarationType]; !ok {
		return nil
	}
	var idRefs []string
	var curValue *fastjson.Value
	for _, path := range IdentifierRefs[declarationType] {
		curValue = v
		for i, pathElem := range path {
			valueAtPath := curValue.Get(pathElem)
			if valueAtPath == nil {
				// if the relative path key doesn't exist then bail
				break
			}

			if i < (len(path) - 1) {
				// not yet at a leaf element
				if valueAtPath.Type() != fastjson.TypeObject {
					// if the relative path key is not an object then bail
					break
				}
				// move on to next path element
				curValue = valueAtPath
				continue
			}

			switch valueAtPath.Type() {
			case fastjson.TypeString:
				idRefs = append(idRefs, string(valueAtPath.GetStringBytes()))
			case fastjson.TypeArray:
				arr, err := valueAtPath.Array()
				if err != nil {
					break
				}
				for _, v := range arr {
					if v.Type() == fastjson.TypeString {
						idRefs = append(idRefs, string(v.GetStringBytes()))
					}
				}
			}
		}

	}
	return idRefs
}

func parseDeclarationValue(v *fastjson.Value, d *Declaration) error {
	d.Identifier = string(v.GetStringBytes("Identifier"))
	payloadValue := v.Get("Payload")
	if payloadValue == nil {
		d.PayloadJSON = []byte("{}")
	} else if payloadValue.Type() != fastjson.TypeObject {
		return errors.New("payload not an object")
	} else {
		d.PayloadJSON = payloadValue.MarshalTo(nil)
	}
	d.ServerToken = string(v.GetStringBytes("ServerToken"))
	d.Type = string(v.GetStringBytes("Type"))
	d.IdentifierRefs = findIDRefs(payloadValue, d.Type)
	return nil
}

// ParseDeclaration parses raw into a Declaration structure.
func ParseDeclaration(raw []byte) (*Declaration, error) {
	v, err := fastjson.ParseBytes(raw)
	if err != nil {
		return nil, fmt.Errorf("parsing json: %w", err)
	}
	d := new(Declaration)
	err = parseDeclarationValue(v, d)
	d.Raw = raw
	return d, err
}
