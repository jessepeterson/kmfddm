// Package ddm supports the core Apple Declarative Device Management protocol.
package ddm

import (
	"encoding/json"
	"errors"
)

var ErrInvalidDeclaration = errors.New("invalid declaration")

// Declaration represents an Apple Declarative Management declaration.
// See https://developer.apple.com/documentation/devicemanagement/declarationbase
// See https://github.com/apple/device-management/blob/release/declarative/declarations/declarationbase.yaml
type Declaration struct {
	Identifier  string          `json:"Identifier"`
	Type        string          `json:"Type"`
	Payload     json.RawMessage `json:"Payload"`
	ServerToken string          `json:"ServerToken"`

	Raw []byte `json:"-"`
}

// Valid performs basic sanity checks.
func (d *Declaration) Valid() bool {
	if d == nil || d.Identifier == "" || d.Type == "" {
		return false
	}
	if len(d.Payload) < 2 {
		// a valid object must at least be `{}`
		return false
	}
	return true
}

// ParseDeclaration parses raw into a Declaration structure.
func ParseDeclaration(raw []byte) (*Declaration, error) {
	d := &Declaration{Raw: raw}
	return d, json.Unmarshal(d.Raw, d)
}
