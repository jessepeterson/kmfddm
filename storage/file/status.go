package file

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/jessepeterson/kmfddm/ddm"
	"github.com/jessepeterson/kmfddm/storage"
)

func (s *File) storeStatusDeclarations(enrollmentID string, declarations []ddm.DeclarationStatus) error {
	if len(declarations) < 1 {
		return nil
	}

	now := time.Now()
	nowText, err := now.MarshalText()
	if err != nil {
		return fmt.Errorf("marshal time to text: %w", err)
	}

	csvFile, err := os.OpenFile(s.csvFilename(csvFilenameDeclarations, enrollmentID), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("opening declaration CSV: %w", err)
	}
	defer csvFile.Close()
	writer := csv.NewWriter(csvFile)

	var records [][]string
	for _, d := range declarations {
		records = append(records, []string{
			string(nowText),
			d.Identifier,
			strconv.FormatBool(d.Active),
			d.Valid,
			d.ServerToken,
			d.ManifestType,
			base64.StdEncoding.EncodeToString(d.ReasonsJSON),
		})

	}

	if err = writer.WriteAll(records); err != nil {
		return fmt.Errorf("writing records: %w", err)
	}

	return nil
}

func (s *File) readStatusValues(enrollmentID string) ([]ddm.StatusValue, error) {
	csvFile, err := os.Open(s.csvFilename(csvFilenameValues, enrollmentID))
	if errors.Is(err, os.ErrNotExist) {
		// no status values (yet)
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("opening status value CSV: %w", err)
	}
	defer csvFile.Close()
	reader := csv.NewReader(csvFile)

	var ret []ddm.StatusValue
	for {
		// read a record
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, fmt.Errorf("reading CSV record: %w", err)
		}

		// record is a set length
		if len(record) != 4 {
			return nil, fmt.Errorf("record fields: %d", len(record))
		}

		ret = append(ret, ddm.StatusValue{
			Path:          record[0],
			ContainerType: record[1],
			ValueType:     record[2],
			Value:         []byte(record[3]),
		})
	}

	return ret, nil
}

func containsValue(values []ddm.StatusValue, v ddm.StatusValue) int {
	for i, value := range values {
		if v.Path == value.Path && v.ValueType == value.ValueType && v.ContainerType == value.ContainerType && bytes.Equal(v.Value, value.Value) {
			return i
		}
	}
	return -1
}

func mergeStatusValues(dst, src []ddm.StatusValue) (ret []ddm.StatusValue) {
	ret = append([]ddm.StatusValue{}, dst...)
	for _, srcValue := range src {
		if containsValue(ret, srcValue) < 0 {
			ret = append(ret, srcValue)
		}
	}
	return
}

func (s *File) storeStatusValues(enrollmentID string, values []ddm.StatusValue) error {
	if len(values) < 1 {
		return nil
	}

	curValues, err := s.readStatusValues(enrollmentID)
	if err != nil {
		return fmt.Errorf("reading values: %w", err)
	}

	// merge in the new values
	values = mergeStatusValues(curValues, values) // TODO: curValues, values

	if len(values) < 1 {
		// nothing to save
		return nil
	}

	csvFile, err := os.OpenFile(s.csvFilename(csvFilenameValues, enrollmentID), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("opening declaration CSV: %w", err)
	}
	defer csvFile.Close()
	writer := csv.NewWriter(csvFile)

	var records [][]string
	for _, v := range values {
		records = append(records, []string{
			v.Path,
			v.ContainerType,
			v.ValueType,
			string(v.Value),
		})
	}

	if err = writer.WriteAll(records); err != nil {
		return fmt.Errorf("writing records: %w", err)
	}

	return nil
}

const (
	csvFilenameErrors       = "status.errors"
	csvFilenameDeclarations = "status.declarations"
	csvFilenameValues       = "status.values"
)

func (s *File) csvFilename(name, enrollmentID string) string {
	return path.Join(s.path, enrollmentID, name+".csv")
}

func (s *File) errorsCSVFilename(enrollmentID string) string {
	return s.csvFilename(csvFilenameErrors, enrollmentID)
}

func (s *File) storeStatusErrors(enrollmentID string, ddmErrors []ddm.StatusError) error {
	if len(ddmErrors) < 1 {
		return nil
	}

	csvFile, err := os.OpenFile(s.errorsCSVFilename(enrollmentID), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("opening error CSV: %w", err)
	}
	defer csvFile.Close()
	writer := csv.NewWriter(csvFile)

	now := time.Now()
	nowText, err := now.MarshalText()
	if err != nil {
		return fmt.Errorf("marshal time to text: %w", err)
	}

	var records [][]string
	for _, ddmError := range ddmErrors {
		records = append(records, []string{
			string(nowText),
			ddmError.Path,
			base64.StdEncoding.EncodeToString(ddmError.ErrorJSON),
		})

	}

	if err = writer.WriteAll(records); err != nil {
		return fmt.Errorf("writing records: %w", err)
	}

	return nil
}

func (s *File) StoreDeclarationStatus(_ context.Context, enrollmentID string, status *ddm.StatusReport) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := s.assureEnrollmentDirExists(enrollmentID)
	if err != nil {
		return fmt.Errorf("assuring enrollment directory exists: %w", err)
	}

	// save a copy of the last complete status report, independent of our status updates.
	if err = os.WriteFile(path.Join(s.path, enrollmentID, "status.last.json"), status.Raw, 0644); err != nil {
		return fmt.Errorf("writing last status: %w", err)
	}

	if err = s.storeStatusDeclarations(enrollmentID, status.Declarations); err != nil {
		return fmt.Errorf("storing declaration status: %w", err)
	}

	if err = s.storeStatusValues(enrollmentID, status.Values); err != nil {
		return fmt.Errorf("storing status values: %w", err)
	}

	if err = s.storeStatusErrors(enrollmentID, status.Errors); err != nil {
		return fmt.Errorf("storing status errors: %w", err)
	}

	return nil
}

func (s *File) RetrieveDeclarationStatus(_ context.Context, enrollmentIDs []string) (map[string][]ddm.DeclarationQueryStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ret := make(map[string][]ddm.DeclarationQueryStatus)
	for _, enrollmentID := range enrollmentIDs {
		di := new(ddm.DeclarationItems)

		f, err := os.Open(s.declarationItemsFilename(enrollmentID))
		if errors.Is(err, os.ErrNotExist) {
			// no declaration items for this enrollment (yet)
			continue
		} else if err != nil {
			return nil, fmt.Errorf("opening declaration items: %w", err)
		}
		defer f.Close()

		if err = json.NewDecoder(f).Decode(di); err != nil {
			return nil, fmt.Errorf("decoding declaration items json: %w", err)
		}

		if di == nil || (len(di.Declarations.Activations) < 1 && len(di.Declarations.Assets) < 1 && len(di.Declarations.Configurations) < 1 && len(di.Declarations.Management) < 1) {
			// no declarations or empty response; move on.
			continue
		}

		// deconstruct the declaration items into a slice of configured declarations
		var manifestDeclarations []ddm.ManifestDeclaration
		manifestDeclarations = append(manifestDeclarations, di.Declarations.Activations...)
		manifestDeclarations = append(manifestDeclarations, di.Declarations.Assets...)
		manifestDeclarations = append(manifestDeclarations, di.Declarations.Configurations...)
		manifestDeclarations = append(manifestDeclarations, di.Declarations.Management...)

		// generate the "placeholder" output for all of our declaration items
		manifestMap := make(map[string]ddm.DeclarationQueryStatus)
		for _, manifestDeclaration := range manifestDeclarations {
			manifestMap[manifestDeclaration.Identifier] = ddm.DeclarationQueryStatus{
				DeclarationStatus: ddm.DeclarationStatus{
					Identifier:  manifestDeclaration.Identifier,
					ServerToken: manifestDeclaration.ServerToken,
				},
			}
		}

		csvFile, err := os.Open(s.csvFilename(csvFilenameDeclarations, enrollmentID))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		defer csvFile.Close()
		reader := csv.NewReader(csvFile)

		for {
			// read a record
			record, err := reader.Read()
			if errors.Is(err, io.EOF) {
				break
			} else if err != nil {
				return nil, fmt.Errorf("reading CSV record: %w", err)
			}

			// record is a set length
			if len(record) != 7 {
				return nil, fmt.Errorf("record fields: %d", len(record))
			}

			// attempt to decode the b64 JSON
			jsonBytes, err := base64.StdEncoding.DecodeString(record[6])
			if err != nil {
				return nil, fmt.Errorf("decoding base64: %w", err)
			}
			var ddmError interface{}
			if len(jsonBytes) > 0 {
				if err = json.Unmarshal(jsonBytes, &ddmError); err != nil {
					return nil, fmt.Errorf("unmarshal reason json: %w", err)
				}
			}

			// decode the timestamp
			var ts time.Time
			if err = ts.UnmarshalText([]byte(record[0])); err != nil {
				return nil, fmt.Errorf("unmarshal time: %w", err)
			}

			active, err := strconv.ParseBool(record[2])
			if err != nil {
				return nil, fmt.Errorf("parse bool: %w", err)
			}

			_, ok := manifestMap[record[1]]
			if !ok {
				// we only want to report on those declarations that are configured
				// i.e. set in our declartion-items
				continue
			}

			// replace placeholder with a "full" declaration query status
			manifestMap[record[1]] = ddm.DeclarationQueryStatus{
				DeclarationStatus: ddm.DeclarationStatus{
					Identifier:   record[1],
					Active:       active,
					Valid:        record[3],
					ServerToken:  record[4],
					ManifestType: record[5],
				},
				Reasons:        ddmError,
				StatusReceived: ts,
				Current:        record[4] == manifestMap[record[1]].ServerToken,
			}
		}
		// turn back into a list
		ddmStatuses := make([]ddm.DeclarationQueryStatus, 0, len(manifestMap))
		for k := range manifestMap {
			ddmStatuses = append(ddmStatuses, manifestMap[k])
		}
		ret[enrollmentID] = ddmStatuses
	}
	return ret, nil
}

// RetrieveStatusErrors reads DDM errors from CSV.
func (s *File) RetrieveStatusErrors(_ context.Context, enrollmentIDs []string, offset, limit int) (map[string][]storage.StatusError, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// TODO: implement offset/limit?

	ret := make(map[string][]storage.StatusError)
	for _, enrollmentID := range enrollmentIDs {
		csvFile, err := os.Open(s.errorsCSVFilename(enrollmentID))
		if errors.Is(err, os.ErrNotExist) {
			// no errors for this enrollment
			continue
		} else if err != nil {
			return nil, err
		}
		defer csvFile.Close()

		reader := csv.NewReader(csvFile)

		var ddmErrors []storage.StatusError
		for {
			// read a record
			record, err := reader.Read()
			if errors.Is(err, io.EOF) {
				break
			} else if err != nil {
				return nil, fmt.Errorf("reading CSV record: %w", err)
			}

			// must be 3 columns wide
			if len(record) != 3 {
				return nil, fmt.Errorf("record fields: %d", len(record))
			}

			// attempt to decode the b64 JSON
			jsonBytes, err := base64.StdEncoding.DecodeString(record[2])
			if err != nil {
				return nil, fmt.Errorf("decoding base64: %w", err)
			}
			var ddmError interface{}
			if err = json.Unmarshal(jsonBytes, &ddmError); err != nil {
				return nil, fmt.Errorf("unmarshal json: %w", err)
			}

			// decode the timestamp
			var ts time.Time
			if err = ts.UnmarshalText([]byte(record[0])); err != nil {
				return nil, fmt.Errorf("unmarshal time: %w", err)
			}

			// assemble and append the record
			ddmErrors = append(ddmErrors, storage.StatusError{
				Path:      record[1],
				Error:     ddmError,
				Timestamp: ts,
			})
		}
		ret[enrollmentID] = ddmErrors
	}

	return ret, nil
}

func filterPathPrefix(values []ddm.StatusValue, pathPrefix string) (ret []ddm.StatusValue) {
	for _, v := range values {
		var found bool
		if hasPre, hasSuf := strings.HasPrefix(pathPrefix, "%"), strings.HasSuffix(pathPrefix, "%"); len(v.Path) >= 3 && hasPre && hasSuf {
			found = strings.Contains(v.Path, pathPrefix[1:len(pathPrefix)-1])
		} else if hasPre {
			found = strings.HasSuffix(v.Path, pathPrefix[1:])
		} else if hasSuf {
			found = strings.HasPrefix(v.Path, pathPrefix[:len(pathPrefix)-1])
		} else {
			found = v.Path == pathPrefix
		}
		if found {
			ret = append(ret, v)
		}
	}
	return
}

func (s *File) RetrieveStatusValues(_ context.Context, enrollmentIDs []string, pathPrefix string) (map[string][]storage.StatusValue, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ret := make(map[string][]storage.StatusValue)
	for _, enrollmentID := range enrollmentIDs {
		values, err := s.readStatusValues(enrollmentID)
		if err != nil {
			return nil, fmt.Errorf("reading status values: %w", err)
		}
		if pathPrefix != "" {
			values = filterPathPrefix(values, pathPrefix)
		}
		var sValues []storage.StatusValue
		for _, v := range values {
			sValues = append(sValues, storage.StatusValue{
				Path:  v.Path,
				Value: string(v.Value),
			})
		}
		if len(sValues) > 0 {
			ret[enrollmentID] = sValues
		}
	}
	return ret, nil
}
