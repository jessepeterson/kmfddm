package ddm

import (
	"errors"
	"fmt"
	"strings"
)

// ParseDeclarationPath parses path to separate out the declaration type and identifier.
// Typically this will be a "a/b" path where "a" is the type and and "b"
// is the declaration.
func ParseDeclarationPath(path string) (string, string, error) {
	split := strings.SplitN(path, "/", 2)
	if len(split) != 2 {
		return "", "", fmt.Errorf("invalid path element count: %d", len(split))
	}
	if split[0] == "" || split[1] == "" {
		return "", "", errors.New("empty type or identifier path elements")
	}
	return split[0], split[1], nil
}
