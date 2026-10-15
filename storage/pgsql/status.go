package pgsql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jessepeterson/kmfddm/ddm"
	"github.com/jessepeterson/kmfddm/storage"
	"github.com/jessepeterson/kmfddm/storage/pgsql/sqlc"
)

// storeStatusDeclarations will completely remove and replace the set of declaration status for an enrollmentID with declarations.
// Exits early if no declarations are present.
func (s *PgSQLStorage) storeStatusDeclarations(ctx context.Context, enrollmentID, statusID string, declarations []ddm.DeclarationStatus) error {
	if len(declarations) < 1 {
		// do not delete existing declaration status if no status are included.
		return nil
	}
	now := time.Now().UnixMilli()
	return tx(ctx, s.db, s.q, func(ctx context.Context, tx *sql.Tx, qtx *sqlc.Queries) error {
		err := qtx.RemoveDeclarationStatus(ctx, enrollmentID)
		if err != nil {
			return err
		}
		for _, ds := range declarations {
			err = qtx.PutDeclarationStatus(ctx, sqlc.PutDeclarationStatusParams{
				Now:                   now,
				EnrollmentID:          enrollmentID,
				ItemType:              ds.ManifestType,
				DeclarationIdentifier: ds.Identifier,
				Active:                ds.Active,
				Valid:                 ds.Valid,
				ServerToken:           ds.ServerToken,
				Reasons:               nullEmptyString(string(ds.ReasonsJSON)),
				StatusID:              nullEmptyString(statusID),
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *PgSQLStorage) storeStatusValues(ctx context.Context, enrollmentID, statusID string, values []ddm.StatusValue) error {
	if len(values) < 1 {
		return nil
	}
	params := sqlc.PutStatusValuesParams{
		Now:            time.Now().UnixMilli(),
		EnrollmentID:   enrollmentID,
		StatusID:       nullEmptyString(statusID),
		Paths:          make([]string, len(values)),
		ContainerTypes: make([]string, len(values)),
		ValueTypes:     make([]string, len(values)),
		Vals:           make([]string, len(values)),
	}
	for i, v := range values {
		params.Paths[i] = v.Path
		params.ContainerTypes[i] = v.ContainerType
		params.ValueTypes[i] = v.ValueType
		params.Vals[i] = string(v.Value)
	}
	return s.q.PutStatusValues(ctx, params)
}

func (s *PgSQLStorage) storeStatusErrors(ctx context.Context, enrollmentID, statusID string, errors []ddm.StatusError) error {
	if len(errors) < 1 {
		return nil
	}
	now := time.Now().UnixMilli()
	err := tx(ctx, s.db, s.q, func(ctx context.Context, _ *sql.Tx, qtx *sqlc.Queries) error {
		for _, e := range errors {
			err := qtx.InsertStatusError(ctx, sqlc.InsertStatusErrorParams{
				Now:          now,
				EnrollmentID: enrollmentID,
				Path:         e.Path,
				Error:        string(e.ErrorJSON),
				StatusID:     nullEmptyString(statusID),
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil || s.errDel < 1 {
		return err
	}
	// deletion is separate from (and after) storing the errors: if it
	// fails the next status report will delete them.
	err = s.q.DeleteStatusErrors(ctx, sqlc.DeleteStatusErrorsParams{
		EnrollmentID: enrollmentID,
		Keep:         int32(s.errDel),
	})
	if err != nil {
		return fmt.Errorf("deleting status errors: %w", err)
	}
	return nil
}

func (s *PgSQLStorage) storeStatusReport(ctx context.Context, enrollmentID, statusID string, raw []byte) error {
	if len(raw) < 1 {
		return errors.New("empty raw status report")
	}
	err := s.q.InsertStatusReport(ctx, sqlc.InsertStatusReportParams{
		Now:          time.Now().UnixMilli(),
		EnrollmentID: enrollmentID,
		StatusID:     nullEmptyString(statusID),
		StatusReport: string(raw),
	})
	if err != nil || s.stsDel < 1 {
		return err
	}
	// deletion is separate from (and after) storing the report: if it
	// fails the next status report will delete them.
	err = s.q.DeleteStatusReports(ctx, sqlc.DeleteStatusReportsParams{
		EnrollmentID: enrollmentID,
		Keep:         int32(s.stsDel),
	})
	if err != nil {
		return fmt.Errorf("deleting status reports: %w", err)
	}
	return nil
}

// StoreDeclarationStatus stores the status report from enrollmentID.
// The raw report is stored as sent. Declaration status, values and errors
// are stored in text and jsonb columns, which PostgreSQL cannot store \u0000
// in, so a report with \u0000 in those parts fails after the raw report is
// stored.
// See also the storage package for documentation on the storage interfaces.
func (s *PgSQLStorage) StoreDeclarationStatus(ctx context.Context, enrollmentID string, status *ddm.StatusReport) error {
	var err error
	if !s.noSts {
		err = s.storeStatusReport(ctx, enrollmentID, status.ID, status.Raw)
		if err != nil {
			return fmt.Errorf("storing status report: %w", err)
		}
	}
	err = s.storeStatusDeclarations(ctx, enrollmentID, status.ID, status.Declarations)
	if err != nil {
		return fmt.Errorf("storing declaration status: %w", err)
	}
	err = s.storeStatusValues(ctx, enrollmentID, status.ID, status.Values)
	if err != nil {
		return fmt.Errorf("storing status values: %w", err)
	}
	err = s.storeStatusErrors(ctx, enrollmentID, status.ID, status.Errors)
	if err != nil {
		return fmt.Errorf("storing status errors: %w", err)
	}
	return nil
}

// RetrieveDeclarationStatus retrieves the status of declarations for enrollmentIDs.
// See also the storage package for documentation on the storage interfaces.
func (s *PgSQLStorage) RetrieveDeclarationStatus(ctx context.Context, enrollmentIDs []string) (map[string][]ddm.DeclarationQueryStatus, error) {
	if len(enrollmentIDs) < 1 {
		return nil, errors.New("no enrollment IDs provided")
	}

	// storeStatusDeclarations removes and replaces an enrollment's rows for
	// every status report, so they are already scoped to that enrollment's
	// latest report.
	rows, err := s.q.GetDeclarationStatus(ctx, enrollmentIDs)
	if err != nil {
		return nil, err
	}
	resp := make(map[string][]ddm.DeclarationQueryStatus)
	for _, row := range rows {
		dqs := ddm.DeclarationQueryStatus{
			DeclarationStatus: ddm.DeclarationStatus{
				Identifier:  row.DeclarationIdentifier,
				Active:      row.Active,
				Valid:       row.Valid,
				ServerToken: row.ServerToken,
			},
			Current:        row.Current,
			StatusReceived: time.UnixMilli(row.UpdatedAt),
		}
		if row.Reasons.Valid {
			dqs.ReasonsJSON = []byte(row.Reasons.String)
			_ = json.Unmarshal(dqs.ReasonsJSON, &dqs.Reasons)
		}
		resp[row.EnrollmentID] = append(resp[row.EnrollmentID], dqs)
	}
	return resp, err
}

// RetrieveStatusErrors retrieves the reported status errors for enrollmentIDs.
// See also the storage package for documentation on the storage interfaces.
func (s *PgSQLStorage) RetrieveStatusErrors(ctx context.Context, enrollmentIDs []string, offset, limit int) (map[string][]storage.StatusError, error) {
	rows, err := s.q.SelectStatusErrors(ctx, sqlc.SelectStatusErrorsParams{
		Ids:       enrollmentIDs,
		RowOffset: int32(offset),
		RowLimit:  int32(limit),
	})
	if err != nil {
		return nil, err
	}
	resp := make(map[string][]storage.StatusError)
	for _, row := range rows {
		sErr := storage.StatusError{
			Path:      row.Path,
			StatusID:  row.StatusID.String,
			Timestamp: time.UnixMilli(row.CreatedAt),
		}
		_ = json.Unmarshal([]byte(row.Error), &sErr.Error)
		resp[row.EnrollmentID] = append(resp[row.EnrollmentID], sErr)
	}
	return resp, nil
}

// RetrieveStatusValues retrieves the status values for enrollmentIDs.
// The search can be filtered with pathPrefix by using SQL LIKE syntax.
// See also the storage package for documentation on the storage interfaces.
func (s *PgSQLStorage) RetrieveStatusValues(ctx context.Context, enrollmentIDs []string, pathPrefix string) (map[string][]storage.StatusValue, error) {
	rows, err := s.q.GetStatusValues(ctx, sqlc.GetStatusValuesParams{
		Ids:        enrollmentIDs,
		PathPrefix: nullEmptyString(pathPrefix),
	})
	if err != nil {
		return nil, err
	}
	resp := make(map[string][]storage.StatusValue)
	for _, row := range rows {
		resp[row.EnrollmentID] = append(resp[row.EnrollmentID], storage.StatusValue{
			Path:      row.Path,
			Value:     row.Value,
			StatusID:  row.StatusID.String,
			Timestamp: time.UnixMilli(row.UpdatedAt),
		})
	}
	return resp, nil
}

// RetrieveStatusReport retrieves the status report for an enrollment ID.
// The search can be filtered with properties on q. Index 0 is the most
// recent status report. If both Index and StatusID are specified then the
// report at Index must also have StatusID. A nil report is returned if none
// is found. The returned report's Index is likewise reverse-chronological
// (0 is the most recent), including when searching by StatusID.
// See also the storage package for documentation on the storage interfaces.
func (s *PgSQLStorage) RetrieveStatusReport(ctx context.Context, q storage.StatusReportQuery) (*storage.StoredStatusReport, error) {
	if err := q.Valid(); err != nil {
		return nil, err
	}
	report := new(storage.StoredStatusReport)
	if q.Index != nil {
		if *q.Index < 0 {
			return nil, fmt.Errorf("index out of range: too low (%d)", *q.Index)
		}
		row, err := s.q.SelectStatusReportByIndex(ctx, sqlc.SelectStatusReportByIndexParams{
			EnrollmentID: q.EnrollmentID,
			RowOffset:    int32(*q.Index),
		})
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		} else if err != nil {
			return nil, err
		}
		if q.StatusID != nil && *q.StatusID != "" && *q.StatusID != row.StatusID.String {
			return nil, nil
		}
		report.StatusID = row.StatusID.String
		report.Index = *q.Index
		report.Raw = []byte(row.StatusReport)
		report.Timestamp = time.UnixMilli(row.CreatedAt)
	} else {
		row, err := s.q.SelectStatusReportByStatusID(ctx, sqlc.SelectStatusReportByStatusIDParams{
			EnrollmentID: q.EnrollmentID,
			StatusID:     nullEmptyString(*q.StatusID),
		})
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		} else if err != nil {
			return nil, err
		}
		report.StatusID = row.StatusID.String
		report.Index = int(row.Idx)
		report.Raw = []byte(row.StatusReport)
		report.Timestamp = time.UnixMilli(row.CreatedAt)
	}
	return report, nil
}
