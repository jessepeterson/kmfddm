package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jessepeterson/kmfddm/ddm"
	"github.com/jessepeterson/kmfddm/storage"
	"github.com/jessepeterson/kmfddm/storage/mysql/sqlc"
)

// storeStatusDeclarations will completely remove and replace the set of declaration status for an enrollmentID with declarations.
// Exits early if no declarations are present.
func (s *MySQLStorage) storeStatusDeclarations(ctx context.Context, enrollmentID, statusID string, declarations []ddm.DeclarationStatus) error {
	if len(declarations) < 1 {
		// do not delete existing declaration status if no status are included.
		return nil
	}
	return tx(ctx, s.db, s.q, func(ctx context.Context, tx *sql.Tx, qtx *sqlc.Queries) error {
		err := qtx.RemoveDeclarationStatus(ctx, enrollmentID)
		if err != nil {
			return err
		}
		for _, ds := range declarations {
			err = qtx.PutDeclarationStatus(ctx, sqlc.PutDeclarationStatusParams{
				EnrollmentID:          enrollmentID,
				ItemType:              ds.ManifestType,
				DeclarationIdentifier: ds.Identifier,
				Active:                ds.Active,
				Valid:                 ds.Valid,
				ServerToken:           ds.ServerToken,
				Reasons:               ds.ReasonsJSON,
				StatusID: sql.NullString{
					String: statusID,
					Valid:  len(statusID) > 0,
				},
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *MySQLStorage) storeStatusValues(ctx context.Context, enrollmentID, statusID string, values []ddm.StatusValue) error {
	if len(values) < 1 {
		return nil
	}
	argSQL := strings.Repeat(", (?, ?, ?, ?, ?, ?)", len(values))[2:]
	const argLen = 6
	args := make([]interface{}, len(values)*argLen)
	for i, v := range values {
		args[i*argLen] = enrollmentID
		args[i*argLen+1] = v.Path
		args[i*argLen+2] = v.ContainerType
		args[i*argLen+3] = v.ValueType
		args[i*argLen+4] = v.Value
		args[i*argLen+5] = sql.NullString{
			String: statusID,
			Valid:  len(statusID) > 0,
		}
	}
	_, err := s.db.ExecContext(
		ctx, `
INSERT INTO status_values
    (
        enrollment_id,
        path,
        container_type,
        value_type,
        value,
        status_id
    )
VALUES
    `+argSQL+` as new
ON DUPLICATE KEY
UPDATE
    updated_at = CURRENT_TIMESTAMP,
    status_id = new.status_id;`,
		args...,
	)
	return err
}

func (s *MySQLStorage) storeStatusErrors(ctx context.Context, enrollmentID, statusID string, errors []ddm.StatusError) error {
	if len(errors) < 1 {
		return nil
	}
	err := tx(ctx, s.db, s.q, func(ctx context.Context, _ *sql.Tx, qtx *sqlc.Queries) error {
		for _, e := range errors {
			err := qtx.InsertStatusError(ctx, sqlc.InsertStatusErrorParams{
				EnrollmentID: enrollmentID,
				Path:         e.Path,
				Error:        e.ErrorJSON,
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
		EnrollmentID:   enrollmentID,
		EnrollmentID_2: enrollmentID,
		Offset:         int32(s.errDel),
	})
	if err != nil {
		return fmt.Errorf("deleting status errors: %w", err)
	}
	return nil
}

func (s *MySQLStorage) storeStatusReport(ctx context.Context, enrollmentID, statusID string, raw []byte) error {
	if len(raw) < 1 {
		return errors.New("empty raw status report")
	}
	err := tx(ctx, s.db, s.q, func(ctx context.Context, _ *sql.Tx, qtx *sqlc.Queries) error {
		return qtx.InsertStatusReport(ctx, sqlc.InsertStatusReportParams{
			EnrollmentID: enrollmentID,
			StatusID:     nullEmptyString(statusID),
			StatusReport: raw,
		})
	})
	if err != nil || s.stsDel < 1 {
		return err
	}
	// deletion is separate from (and after) storing the report: if it
	// fails the next status report will delete them.
	err = s.q.DeleteStatusReports(ctx, sqlc.DeleteStatusReportsParams{
		EnrollmentID:   enrollmentID,
		EnrollmentID_2: enrollmentID,
		Offset:         int32(s.stsDel),
	})
	if err != nil {
		return fmt.Errorf("deleting status reports: %w", err)
	}
	return nil
}

// StoreDeclarationStatus stores the status report from enrollmentID.
// See also the storage package for documentation on the storage interfaces.
func (s *MySQLStorage) StoreDeclarationStatus(ctx context.Context, enrollmentID string, status *ddm.StatusReport) error {
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
func (s *MySQLStorage) RetrieveDeclarationStatus(ctx context.Context, enrollmentIDs []string) (map[string][]ddm.DeclarationQueryStatus, error) {
	if len(enrollmentIDs) < 1 {
		return nil, errors.New("no enrollment IDs provided")
	}

	// storeStatusDeclarations removes and replaces an enrollment's rows for
	// every status report, so they are already scoped to that enrollment's
	// latest report. This query must not join set_declarations or
	// enrollment_sets to scope them any further: those tables contribute no
	// columns and only multiply each row by the number of sets carrying the
	// declaration.
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
				ReasonsJSON: row.Reasons,
			},
			Current: row.Current,
		}
		dqs.StatusReceived, _ = time.Parse(mysqlTimeFormat, row.UpdatedAt)
		if len(row.Reasons) > 0 {
			_ = json.Unmarshal(row.Reasons, &dqs.Reasons)
		}
		resp[row.EnrollmentID] = append(resp[row.EnrollmentID], dqs)
	}
	return resp, err
}

// RetrieveStatusErrors retrieves the reported status errors for enrollmentIDs.
// See also the storage package for documentation on the storage interfaces.
func (s *MySQLStorage) RetrieveStatusErrors(ctx context.Context, enrollmentIDs []string, offset, limit int) (map[string][]storage.StatusError, error) {
	rows, err := s.q.SelectStatusErrors(ctx, sqlc.SelectStatusErrorsParams{
		Ids:    enrollmentIDs,
		Offset: int32(offset),
		Limit:  int32(limit),
	})
	if err != nil {
		return nil, err
	}
	resp := make(map[string][]storage.StatusError)
	for _, row := range rows {
		sErr := storage.StatusError{
			Path:     row.Path,
			StatusID: row.StatusID.String,
		}
		_ = json.Unmarshal(row.Error, &sErr.Error)
		sErr.Timestamp, _ = time.Parse(mysqlTimeFormat, row.CreatedAt)
		resp[row.EnrollmentID] = append(resp[row.EnrollmentID], sErr)
	}
	return resp, nil
}

// RetrieveStatusValues retrieves the status values for enrollmentIDs.
// The search can be filtered with pathPrefix by using SQL LIKE syntax.
// See also the storage package for documentation on the storage interfaces.
func (s *MySQLStorage) RetrieveStatusValues(ctx context.Context, enrollmentIDs []string, pathPrefix string) (map[string][]storage.StatusValue, error) {
	idSQL := strings.Repeat(", ?", len(enrollmentIDs))[2:]
	args := make([]interface{}, len(enrollmentIDs))
	for i, id := range enrollmentIDs {
		args[i] = id
	}
	prefixCond := ""
	if pathPrefix != "" {
		args = append(args, pathPrefix)
		prefixCond = `AND path LIKE ?`
	}
	rows, err := s.db.QueryContext(
		ctx, `
SELECT
    enrollment_id,
    path,
    value,
    status_id,
    updated_at
FROM
    status_values
WHERE
    enrollment_id IN (`+idSQL+`) `+prefixCond+`
ORDER BY
    enrollment_id, created_at;`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	resp := make(map[string][]storage.StatusValue)
	var id string
	for rows.Next() {
		sVal := storage.StatusValue{}
		var dbTimestamp string
		var statusID sql.NullString
		err = rows.Scan(
			&id,
			&sVal.Path,
			&sVal.Value,
			&statusID,
			&dbTimestamp,
		)
		if err != nil {
			break
		}
		sVal.StatusID = statusID.String
		sVal.Timestamp, _ = time.Parse(mysqlTimeFormat, dbTimestamp)
		resp[id] = append(resp[id], sVal)
	}
	if err == nil {
		err = rows.Err()
	}
	return resp, err
}

// RetrieveStatusReport retrieves the status report for an enrollment ID.
// The search can be filtered with properties on q. Index 0 is the most
// recent status report. If both Index and StatusID are specified then the
// report at Index must also have StatusID. A nil report is returned if none
// is found. The returned report's Index is likewise reverse-chronological
// (0 is the most recent), including when searching by StatusID.
// See also the storage package for documentation on the storage interfaces.
func (s *MySQLStorage) RetrieveStatusReport(ctx context.Context, q storage.StatusReportQuery) (*storage.StoredStatusReport, error) {
	if err := q.Valid(); err != nil {
		return nil, err
	}
	report := new(storage.StoredStatusReport)
	var createdAt string
	if q.Index != nil {
		if *q.Index < 0 {
			return nil, fmt.Errorf("index out of range: too low (%d)", *q.Index)
		}
		row, err := s.q.SelectStatusReportByIndex(ctx, sqlc.SelectStatusReportByIndexParams{
			EnrollmentID: q.EnrollmentID,
			Offset:       int32(*q.Index),
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
		report.Raw = row.StatusReport
		createdAt = row.CreatedAt
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
		report.Raw = row.StatusReport
		createdAt = row.CreatedAt
	}
	report.Timestamp, _ = time.Parse(mysqlTimeFormat, createdAt)
	return report, nil
}
