package test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jessepeterson/kmfddm/ddm"
	"github.com/jessepeterson/kmfddm/storage"
)

type statusStorage interface {
	storage.StatusStorage
	storage.DeclarationStorer
	storage.SetDeclarationStorage
	storage.EnrollmentSetStorer
	storage.StatusAPIStorage
}

const statusFile1 = "testdata/status.1st.json"
const statusFile2 = "testdata/status.D0.error.json"
const statusFileID1 = "A047820F-FC6B-4104-BED0-466876D82BB8"
const statusFileID2 = "D0463AF6-D0BF-5D06-BBBC-4A9A1386D613"

const testDecl2 = `{
    "Type": "com.apple.configuration.management.test",
    "Payload": {
        "Echo": "KMFDDM!"
    },
    "Identifier": "com.example.test"
}`

func getPathValue(values []storage.StatusValue, path string) string {
	for _, v := range values {
		if v.Path == path {
			return v.Value
		}
	}
	return ""
}

func TestBasicStatus(t *testing.T, pathToDDMTestdata string, storage statusStorage, ctx context.Context) {
	jsonBytes, err := os.ReadFile(filepath.Join(pathToDDMTestdata, statusFile1))
	if err != nil {
		t.Fatal(err)
	}

	status, err := ddm.ParseStatus(jsonBytes)
	if err != nil {
		t.Fatal(err)
	}

	err = storage.StoreDeclarationStatus(ctx, statusFileID1, status)
	if err != nil {
		t.Fatal(err)
	}

	enrollmentValues, err := storage.RetrieveStatusValues(ctx, []string{statusFileID1}, "")
	if err != nil {
		t.Fatal(err)
	}

	if enrollmentValues == nil {
		t.Fatal("nil")
	}

	values, ok := enrollmentValues[statusFileID1]
	if !ok {
		t.Error("enrollment ID not found")
	}

	if have, want := getPathValue(values, ".StatusItems.device.operating-system.family"), "macOS"; have != want {
		t.Errorf("have: %v, want: %v", have, want)
	}

	jsonBytes, err = os.ReadFile(filepath.Join(pathToDDMTestdata, statusFile2))
	if err != nil {
		t.Fatal(err)
	}

	status, err = ddm.ParseStatus(jsonBytes)
	if err != nil {
		t.Fatal(err)
	}

	// we have to setup and enable a declaration for it to show up in our declaration status

	d, err := ddm.ParseDeclaration([]byte(testDecl2))
	if err != nil {
		t.Fatal(err)
	}

	_, err = storage.StoreDeclaration(ctx, d)
	if err != nil {
		t.Fatal(err)
	}

	_, err = storage.StoreSetDeclaration(ctx, "default", d.Identifier)
	if err != nil {
		t.Fatal(err)
	}

	_, err = storage.StoreEnrollmentSet(ctx, statusFileID2, "default")
	if err != nil {
		t.Fatal(err)
	}

	// store declaration
	err = storage.StoreDeclarationStatus(ctx, statusFileID2, status)
	if err != nil {
		t.Fatal(err)
	}

	ddmErrorMap, err := storage.RetrieveStatusErrors(ctx, []string{statusFileID2}, 0, 10)
	if err != nil {
		t.Fatal(err)
	}

	if ddmErrorMap == nil {
		t.Fatal("nil map")
	}

	ddmErrors, ok := ddmErrorMap[statusFileID2]
	if !ok {
		t.Error("enrollment ID not found")
	}

	if len(ddmErrors) < 1 {
		t.Fatal("too few errors")
	}

	if have, want := ddmErrors[0].Path, ".StatusItems.management.declarations.configurations"; have != want {
		t.Errorf("have: %v, want: %v", have, want)
	}

	declStatuses, err := storage.RetrieveDeclarationStatus(ctx, []string{statusFileID2})
	if err != nil {
		t.Fatal(err)
	}

	if declStatuses == nil {
		t.Fatal("nil map")
	}

	declStatus, ok := declStatuses[statusFileID2]
	if !ok {
		t.Error("enrollment ID not found")
	}

	if len(declStatus) < 1 {
		t.Fatal("too few errors")
	}

	if have, want := declStatus[0].Identifier, "com.example.test"; have != want {
		t.Errorf("have: %v, want: %v", have, want)
	}

	if have, want := declStatus[0].Current, false; have != want {
		t.Errorf("have: %v, want: %v", have, want)
	}

	if have, want := declStatus[0].Active, false; have != want {
		t.Errorf("have: %v, want: %v", have, want)
	}

}
