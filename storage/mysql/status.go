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

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(
		ctx,
		`UPDATE status_errors SET row_count = row_count + 1 WHERE enrollment_id = ?;`,
		enrollmentID,
	)

	if err == nil {
		argSQL := strings.Repeat(", (?, ?, ?, ?)", len(errors))[2:]
		const argLen = 4
		args := make([]interface{}, len(errors)*argLen)
		for i, e := range errors {
			args[i*argLen] = enrollmentID
			args[i*argLen+1] = e.Path
			args[i*argLen+2] = e.ErrorJSON
			args[i*argLen+3] = sql.NullString{
				String: statusID,
				Valid:  len(statusID) > 0,
			}
		}
		_, err = tx.ExecContext(
			ctx, `
INSERT INTO status_errors
    (
        enrollment_id,
        path,
        error,
        status_id
    )
VALUES
    `+argSQL+`;`,
			args...,
		)
	}

	if s.errDel > 0 {
		_, err = tx.ExecContext(
			ctx,
			`DELETE FROM status_errors WHERE enrollment_id = ? AND row_count >= ?`,
			enrollmentID,
			s.errDel,
		)
	}

	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback error: %w; while trying to handle error: %v", rbErr, err)
		}
		return err
	}

	return tx.Commit()
}

func (s *MySQLStorage) storeStatusReport(ctx context.Context, enrollmentID, statusID string, raw []byte) error {
	if len(raw) < 1 {
		return errors.New("empty raw status report")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(
		ctx,
		`UPDATE status_reports SET row_count = row_count + 1 WHERE enrollment_id = ?;`,
		enrollmentID,
	)

	if err == nil {
		_, err = tx.ExecContext(
			ctx, `
INSERT INTO status_reports
    (
        enrollment_id,
        status_id,
        status_report
    )
VALUES
    (?, ?, ?);`,
			enrollmentID,
			statusID,
			raw,
		)
	}

	if s.stsDel > 0 {
		_, err = tx.ExecContext(
			ctx,
			`DELETE FROM status_errors WHERE enrollment_id = ? AND row_count >= ?`,
			enrollmentID,
			s.stsDel,
		)
	}

	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback error: %w; while trying to handle error: %v", rbErr, err)
		}
		return err
	}

	return tx.Commit()
}

// StoreDeclarationStatus stores the status report from enrollmentID.
// See also the storage package for documentation on the storage interfaces.
func (s *MySQLStorage) StoreDeclarationStatus(ctx context.Context, enrollmentID string, status *ddm.StatusReport) error {
	err := s.storeStatusReport(ctx, enrollmentID, status.ID, status.Raw)
	if err != nil {
		return fmt.Errorf("storing status report: %w", err)
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
	idSQL := strings.Repeat(", ?", len(enrollmentIDs))[2:]
	args := make([]interface{}, len(enrollmentIDs), len(enrollmentIDs)+2)
	for i, id := range enrollmentIDs {
		args[i] = id
	}
	args = append(args, offset, limit)
	rows, err := s.db.QueryContext(
		ctx, `
SELECT
    enrollment_id,
    path,
    error,
	status_id,
	created_at
FROM
    status_errors
WHERE
    enrollment_id IN (`+idSQL+`)
ORDER BY
    enrollment_id, created_at
LIMIT ?, ?;`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	resp := make(map[string][]storage.StatusError)
	var id, dbTimestamp string
	var dbErrorJSON []byte
	var statusID sql.NullString
	for rows.Next() {
		sErr := storage.StatusError{}
		err = rows.Scan(&id, &sErr.Path, &dbErrorJSON, &statusID, &dbTimestamp)
		if err != nil {
			break
		}
		_ = json.Unmarshal(dbErrorJSON, &sErr.Error)
		sErr.StatusID = statusID.String
		sErr.Timestamp, _ = time.Parse(mysqlTimeFormat, dbTimestamp)
		resp[id] = append(resp[id], sErr)
	}
	if err == nil {
		err = rows.Err()
	}
	return resp, err
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

// RetrieveStatusValues retrieves the status report for an enrollment ID.
// The search can be filtered with properties on q.
// See also the storage package for documentation on the storage interfaces.
func (s *MySQLStorage) RetrieveStatusReport(ctx context.Context, q storage.StatusReportQuery) (*storage.StoredStatusReport, error) {
	if err := q.Valid(); err != nil {
		return nil, err
	}
	args := []interface{}{q.EnrollmentID}
	where := ""
	if q.Index != nil {
		where = "row_count = ?"
		args = append(args, *q.Index)
	}
	if q.StatusID != nil {
		if where != "" {
			where += " AND"
		}
		where += "status_id = ?"
		args = append(args, *q.StatusID)
	}
	if where == "" {
		return nil, errors.New("invalid query")
	}
	report := new(storage.StoredStatusReport)
	var dbTimestamp string
	err := s.db.QueryRowContext(
		ctx,
		`
SELECT
    status_id,
	created_at,
	row_count,
    status_report
FROM
    status_reports
WHERE
    enrollment_id = ? AND `+where+`
LIMIT 1;`,
		args...,
	).Scan(
		&report.StatusID,
		&dbTimestamp,
		&report.Index,
		&report.Raw,
	)
	if err != nil {
		return report, err
	}
	report.Timestamp, _ = time.Parse(mysqlTimeFormat, dbTimestamp)
	return report, err
}
