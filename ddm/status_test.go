package ddm

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

const statusFile1 = "testdata/A047820F-FC6B-4104-BED0-466876D82BB8.20220628-103225.001918.status.json"

func TestStatusParse(t *testing.T) {
	jsonBytes, err := os.ReadFile(statusFile1)
	if err != nil {
		t.Fatal(err)
	}
	_, s, err := ParseStatus(jsonBytes)
	if err != nil {
		t.Fatal(err)
	}
	for _, toFind := range [][]string{
		{".StatusItems.device.model.family", "iPhone"},
		{".StatusItems.management.client-capabilities.supported-payloads.status-items", "device.operating-system.family"},
	} {
		toFindPath := toFind[0]
		toFindValue := toFind[1]

		found := false

		for _, v := range s.Values {
			if toFindPath == v.Path && toFindValue == string(v.Value) {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("value not found at path: %s", toFindPath)
		}
	}

	for _, missingPath := range []string{
		".StatusItems.management.declarations",
	} {
		found := false

		for _, v := range s.Values {
			if strings.HasPrefix(v.Path, missingPath) {
				found = true
				break
			}
		}

		if found {
			t.Errorf("found path that should be missing: %s", missingPath)
		}
	}

	if want := 0; len(s.Errors) != want {
		t.Errorf("invalid number of errors: want %d, have %d", want, len(s.Errors))
	}

	for _, errorJSON := range []string{} {
		var found bool
		for _, v := range s.Errors {
			found = bytes.Contains(v.ErrorJSON, []byte(errorJSON))
			if found {
				break
			}
		}
		if !found {
			t.Errorf("error not found: %s", errorJSON)
		}
	}

	if want := 5; len(s.Declarations) != want {
		t.Errorf("invalid number of declarations: want %d, have %d", want, len(s.Declarations))
	}
}
