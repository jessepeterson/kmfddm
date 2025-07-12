package kv

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jessepeterson/kmfddm/ddm"
	"github.com/jessepeterson/kmfddm/storage"
	"github.com/micromdm/nanolib/storage/kv"
)

const (
	keyPfxStaRaw = "rs"
	keyPfxStaDcl = "ds"
	keyPfxStaVal = "vs"
	keyPfxStaErr = "es"

	keySfxStaEnrIdx = "index"

	keySfxStaRawRaw = "raw"
	keySfxStaRawID  = "id"
	keySfxStaRawTS  = "ts"

	keySfxStaDclID  = "id"
	keySfxStaDclIdx = "index"
	keySfxStaDclJso = "json"
	keySfxStaDclTS  = "ts"

	keySfxStaValPth = "pt"
	keySfxStaValTyp = "vt"
	keySfxStaValCty = "ct"
	keySfxStaValVal = "va"
	keySfxStaValID  = "id"
	keySfxStaValTS  = "ts"

	keySfxStaErrErr = "error"
	keySfxStaErrPth = "path"
	keySfxStaErrTS  = "ts"
	keySfxStaErrID  = "id"
)

func fromTime(t time.Time) []byte {
	return []byte(strconv.FormatInt(t.UnixMicro(), 10))
}

func toTime(b []byte) (time.Time, error) {
	i, err := strconv.ParseInt(string(b), 10, 64)
	return time.UnixMicro(i), err
}

// StoreDeclarationStatus stores the status report details.
// For later retrieval by the StatusAPIStorage interface(s).
func (s *KV) StoreDeclarationStatus(ctx context.Context, enrollmentID string, status *ddm.StatusReport) error {
	var idx int
	now := time.Now()
	// write the raw status report
	err := kv.PerformCRUDBucketTxn(ctx, s.status, func(ctx context.Context, b kv.CRUDBucket) error {
		idx, err := bumpIdx(ctx, b, join(keyPfxStaRaw, enrollmentID, keySfxStaEnrIdx))
		if err != nil {
			return err
		}

		// store the full status report
		pfx := join(keyPfxStaRaw, enrollmentID, strconv.Itoa(idx))
		return kv.SetMap(ctx, b, map[string][]byte{
			join(pfx, keySfxStaRawRaw): status.Raw,
			join(pfx, keySfxStaRawID):  []byte(status.ID), // TODO: id is optional
			join(pfx, keySfxStaRawTS):  fromTime(now),
		})
	})
	if err != nil {
		return err
	}
	// write the declaration status
	err = kv.PerformCRUDBucketTxn(ctx, s.status, func(ctx context.Context, b kv.CRUDBucket) error {
		if len(status.Declarations) > 0 {
			dStatusJSON, err := json.Marshal(status.Declarations)
			if err != nil {
				return fmt.Errorf("marshal declaration status: %w", err)
			}

			err = kv.SetMap(ctx, b, map[string][]byte{
				join(keyPfxStaDcl, enrollmentID, keySfxStaDclJso): dStatusJSON,
				join(keyPfxStaDcl, enrollmentID, keySfxStaDclIdx): []byte(strconv.Itoa(idx)),
				join(keyPfxStaDcl, enrollmentID, keySfxStaDclTS):  fromTime(now),
			})
			if err != nil {
				return err
			}

			if status.ID != "" {
				err = b.Set(ctx, join(keyPfxStaDcl, enrollmentID, keySfxStaDclID), []byte(status.ID))
				if err != nil {
					return err
				}
			} else {
				err = b.Delete(ctx, join(keyPfxStaDcl, enrollmentID, keySfxStaDclID))
				if err != nil {
					return err
				}
			}
		}
		// don't delete anything
		//  else {
		// 	// this means all status reports must contain declarations or we'll
		// 	// wipe out declaration status each time a report does not contain
		// 	// them.
		// 	return kv.DeleteSlice(ctx, b, []string{
		// 		join(keyPfxStaDcl, enrollmentID, keySfxStaDclJso),
		// 		join(keyPfxStaDcl, enrollmentID, keySfxStaDclIdx),
		// 		join(keyPfxStaDcl, enrollmentID, keySfxStaDclTS),
		// 		join(keyPfxStaDcl, enrollmentID, keySfxStaDclID),
		// 	})
		// }
		return nil
	})
	if err != nil {
		return err
	}
	// write the status values
	err = kv.PerformCRUDBucketTxn(ctx, s.status, func(ctx context.Context, b kv.CRUDBucket) error {
		for _, value := range status.Values {
			pkHash := s.newHash()
			pkHash.Write([]byte(value.Path + value.ContainerType + value.ValueType + string(value.Value)))
			pk := fmt.Sprintf("%x", pkHash.Sum(nil))
			err = kv.SetMap(ctx, b, map[string][]byte{
				join(keyPfxStaVal, enrollmentID, keySfxStaValPth, pk): []byte(value.Path),
				join(keyPfxStaVal, enrollmentID, keySfxStaValCty, pk): []byte(value.ContainerType),
				join(keyPfxStaVal, enrollmentID, keySfxStaValTyp, pk): []byte(value.ValueType),
				join(keyPfxStaVal, enrollmentID, keySfxStaValVal, pk): value.Value,
				join(keyPfxStaVal, enrollmentID, keySfxStaValTS, pk):  fromTime(now),
			})
			if err != nil {
				return err
			}

			if status.ID != "" {
				err = b.Set(ctx, join(keyPfxStaVal, enrollmentID, keySfxStaValID, pk), []byte(status.ID))
				if err != nil {
					return err
				}
			} else {
				err = b.Delete(ctx, join(keyPfxStaVal, enrollmentID, keySfxStaValID, pk))
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	// write the status errors
	return kv.PerformCRUDBucketTxn(ctx, s.status, func(ctx context.Context, b kv.CRUDBucket) error {
		for _, statusError := range status.Errors {
			idx, err := bumpIdx(ctx, b, join(keyPfxStaErr, enrollmentID))
			if err != nil {
				return fmt.Errorf("bumping index for error status: %w", err)
			}

			pfx := join(keyPfxStaErr, enrollmentID, strconv.Itoa(idx))
			err = kv.SetMap(ctx, b, map[string][]byte{
				join(pfx, keySfxStaErrErr): statusError.ErrorJSON,
				join(pfx, keySfxStaErrPth): []byte(statusError.Path),
				join(pfx, keySfxStaErrTS):  fromTime(now),
			})
			if err != nil {
				return err
			}

			if status.ID != "" {
				err = b.Set(ctx, join(pfx, keySfxStaErrID), []byte(status.ID))
				if err != nil {
					return err
				}
			} else {
				err = b.Delete(ctx, join(pfx, keySfxStaErrID))
				if err != nil {
					return err
				}
			}

		}
		return nil
	})
}

// bumpIdx reads the value at key in b, increments it by one, and writes it back out to b.
// The value should be a string representation of an integer or is assumed to be 0.
// The returned int is same as is written back out to b.
func bumpIdx(ctx context.Context, b kv.CRUDBucket, key string) (int, error) {
	var v int
	if raw, err := b.Get(ctx, key); err != nil && !errors.Is(err, kv.ErrKeyNotFound) {
		return v, err
	} else if v, err = strconv.Atoi(string(raw)); err == nil {
		// only increment if no errors, any errors should keep v=0.
		v++
	}
	return v, b.Set(ctx, key, []byte(strconv.Itoa(v)))
}

// retrIdx retrieves the value at key and attempts to convert to an interger.
// Errors with the conversion are ignored (returning a 0).
func retrIdx(ctx context.Context, b kv.CRUDBucket, key string) (int, error) {
	raw, err := b.Get(ctx, key)
	if err != nil && !errors.Is(err, kv.ErrKeyNotFound) {
		return 0, err
	}
	v, _ := strconv.Atoi(string(raw))
	return v, nil
}

// RetrieveDeclarationStatus retrieves the status of the declarations for enrollmentIDs.
func (s *KV) RetrieveDeclarationStatus(ctx context.Context, enrollmentIDs []string) (map[string][]ddm.DeclarationQueryStatus, error) {
	r := make(map[string][]ddm.DeclarationQueryStatus)
	for _, id := range enrollmentIDs {
		// ignoring errors as we don't technically need this to serve our status reply
		enrDIs, _ := s.RetrieveDeclarationItems(ctx, id)

		dMap, err := kv.GetMap(ctx, s.status, []string{
			join(keyPfxStaDcl, id, keySfxStaDclJso),
			join(keyPfxStaDcl, id, keySfxStaDclIdx),
			join(keyPfxStaDcl, id, keySfxStaDclTS),
		})
		if errors.Is(err, kv.ErrKeyNotFound) {
			continue
		} else if err != nil {
			return nil, err
		}

		var statusID string
		if rawID, err := s.status.Get(ctx, join(keyPfxStaDcl, id, keySfxStaDclID)); err != nil && !errors.Is(err, kv.ErrKeyNotFound) {
			return nil, err
		} else if err == nil && len(rawID) > 0 {
			statusID = string(rawID)
		}

		var dcls []ddm.DeclarationStatus
		if len(dMap[join(keyPfxStaDcl, id, keySfxStaDclJso)]) > 0 {
			err = json.Unmarshal(dMap[join(keyPfxStaDcl, id, keySfxStaDclJso)], &dcls)
			if err != nil {
				return nil, err
			}
		}

		ts, err := toTime(dMap[join(keyPfxStaDcl, id, keySfxStaDclTS)])
		if err != nil {
			return nil, err
		}

		var dqss []ddm.DeclarationQueryStatus

		for _, ds := range dcls {
			dqs := ddm.DeclarationQueryStatus{
				DeclarationStatus: ds,
				StatusID:          statusID,
				StatusReceived:    ts,
				Current:           false,
			}

			// now loop through our IDs and see if we're current
			for _, enrDI := range enrDIs {
				if enrDI.Identifier == ds.Identifier {
					if enrDI.ServerToken == ds.ServerToken {
						dqs.Current = true
					}
				}
			}

			if len(ds.ReasonsJSON) > 0 {
				err := json.Unmarshal(ds.ReasonsJSON, &dqs.Reasons)
				if err != nil {
					return nil, err
				}
			}

			dqss = append(dqss, dqs)
		}

		if len(dqss) > 0 {
			r[id] = dqss
		}
	}
	return r, nil
}

// RetrieveStatusErrors retrieves the collected errors for enrollmentIDs.
func (s *KV) RetrieveStatusErrors(ctx context.Context, enrollmentIDs []string, offset, limit int) (map[string][]storage.StatusError, error) {
	r := make(map[string][]storage.StatusError)
	for _, id := range enrollmentIDs {
		idx, err := retrIdx(ctx, s.status, join(keyPfxStaErr, id))
		if err != nil {
			continue
		}

		var errs []storage.StatusError
		for i := 0; i <= idx; i++ {
			// loop through all of the stored errors for this id

			pfx := join(keyPfxStaErr, id, strconv.Itoa(i))
			eMap, err := kv.GetMap(ctx, s.status, []string{
				join(pfx, keySfxStaErrErr),
				join(pfx, keySfxStaErrPth),
				join(pfx, keySfxStaErrTS),
			})
			if errors.Is(err, kv.ErrKeyNotFound) {
				continue
			} else if err != nil {
				return nil, fmt.Errorf("error retrieving nth stored error: %d for id: %s: %w", i, id, err)
			}

			statusError := storage.StatusError{
				Path: string(eMap[join(pfx, keySfxStaErrPth)]),
			}

			err = json.Unmarshal(eMap[join(pfx, keySfxStaErrErr)], &statusError.Error)
			if err != nil {
				return nil, fmt.Errorf("error retrieving nth stored error: %d for id: %s: %w", i, id, err)
			}

			statusError.Timestamp, err = toTime(eMap[join(pfx, keySfxStaErrTS)])
			if err != nil {
				return nil, fmt.Errorf("error retrieving nth stored error: %d for id: %s: %w", i, id, err)
			}

			rawID, err := s.status.Get(ctx, join(pfx, keySfxStaErrID))
			if err != nil && !errors.Is(err, kv.ErrKeyNotFound) {
				return nil, fmt.Errorf("error retrieving nth stored error: %d for id: %s: %w", i, id, err)
			} else if err == nil && len(rawID) > 0 {
				statusError.StatusID = string(rawID)
			}

			errs = append(errs, statusError)
		}

		if len(errs) > 0 {
			r[id] = errs
		}
	}
	return r, nil
}

// RetrieveStatusValues retrieves the collected errors for enrollmentIDs.
func (s *KV) RetrieveStatusValues(ctx context.Context, enrollmentIDs []string, pathPrefix string) (map[string][]storage.StatusValue, error) {
	r := make(map[string][]storage.StatusValue)
	for _, id := range enrollmentIDs {

		rawKeys := kv.AllKeysPrefix(ctx, s.status, join(keyPfxStaVal, id, keySfxStaValPth)+keySep)

		var values []storage.StatusValue

		for _, k := range rawKeys {
			pk := k[len(join(keyPfxStaVal, id, keySfxStaValPth)+keySep):]

			pMap, err := kv.GetMap(ctx, s.status, []string{
				join(keyPfxStaVal, id, keySfxStaValPth, pk),
				join(keyPfxStaVal, id, keySfxStaValVal, pk),
				// not used in query, but stored
				// join(keyPfxStaVal, id, keySfxStaValTyp, pk),
				// join(keyPfxStaVal, id, keySfxStaValCty, pk),
				join(keyPfxStaVal, id, keySfxStaValTS, pk),
			})
			if err != nil {
				return nil, err
			}
			v := storage.StatusValue{
				Path:  string(pMap[join(keyPfxStaVal, id, keySfxStaValPth, pk)]),
				Value: string(pMap[join(keyPfxStaVal, id, keySfxStaValVal, pk)]),
			}

			v.Timestamp, _ = toTime(pMap[join(keyPfxStaVal, id, keySfxStaValTS, pk)])

			// retrieve the optional status ID
			if statusID, err := s.status.Get(ctx, join(keyPfxStaVal, id, keySfxStaValID, pk)); err != nil && !errors.Is(err, kv.ErrKeyNotFound) {
				return nil, err
			} else if err == nil {
				v.StatusID = string(statusID)
			}

			values = append(values, v)
		}

		if len(values) > 0 {
			r[id] = values
		}
	}
	return r, nil
}

// RetrieveStatusReport retrieves an enrollment's raw status report that matches q.
func (s *KV) RetrieveStatusReport(ctx context.Context, q storage.StatusReportQuery) (*storage.StoredStatusReport, error) {
	if q.EnrollmentID == "" {
		return nil, errors.New("empty enrollment ID")
	}
	if q.StatusID != nil {
		return nil, errors.New("cannot search by status id")
	}
	if q.Index == nil {
		return nil, errors.New("empty index")
	}
	if *q.Index < 0 {
		return nil, fmt.Errorf("index out of range: too low (%d)", *q.Index)
	}
	i, err := retrIdx(ctx, s.status, join(keyPfxStaRaw, q.EnrollmentID, keySfxStaEnrIdx))
	if err != nil {
		return nil, err
	}
	if *q.Index > i {
		return nil, fmt.Errorf("index out of range: too high (%d)", *q.Index)
	}

	// TODO: this presumes we never delete reports. which we don't, currently
	aIdx := i - *q.Index

	pfx := join(keyPfxStaRaw, q.EnrollmentID, strconv.Itoa(aIdx))
	sMap, err := kv.GetMap(ctx, s.status, []string{
		join(pfx, keySfxStaRawRaw),
		join(pfx, keySfxStaRawTS),
	})
	if err != nil {
		return nil, err
	}
	statusID, err := s.status.Get(ctx, join(pfx, keySfxStaRawID))
	if err != nil && !errors.Is(err, kv.ErrKeyNotFound) {
		return nil, err
	}
	ts, err := toTime(sMap[join(pfx, keySfxStaRawTS)])
	if err != nil {
		return nil, err
	}
	return &storage.StoredStatusReport{
		Raw:       sMap[join(pfx, keySfxStaRawRaw)],
		StatusID:  string(statusID),
		Timestamp: ts,
		Index:     aIdx, // nominally optional, but required in this implementation
	}, nil
}
