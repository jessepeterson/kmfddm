package mysql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jessepeterson/kmfddm/ddm"
	"github.com/jessepeterson/kmfddm/storage"
)

func (s *MySQLStorage) storeStatusDeclarations(ctx context.Context, enrollmentID string, declarations []ddm.DeclarationStatus) error {
	if len(declarations) < 1 {
		return nil
	}
	const argLen = 7
	argSQL := strings.Repeat(", (?, ?, ?, ?, ?, ?, ?)", len(declarations))[1:]
	args := make([]interface{}, len(declarations)*argLen)
	for i, d := range declarations {
		args[i*argLen] = enrollmentID
		args[i*argLen+1] = d.ManifestType
		args[i*argLen+2] = d.Identifier
		args[i*argLen+3] = d.Active
		args[i*argLen+4] = d.Valid
		args[i*argLen+5] = d.ServerToken
		args[i*argLen+6] = d.ReasonsJSON
	}
	_, err := s.db.ExecContext(
		ctx, `
INSERT INTO status_declarations
    (
        enrollment_id,
        item_type,
        declaration_identifier,
        active,
        valid,
        server_token,
        reasons
    )
VALUES
    `+argSQL+` AS new
ON DUPLICATE KEY
UPDATE
    item_type = new.item_type,
    active = new.active,
    valid = new.valid,
    server_token = new.server_token,
    reasons = new.reasons;`,
		args...,
	)

	return err
}

func (s *MySQLStorage) storeStatusValues(ctx context.Context, enrollmentID string, values []ddm.StatusValue) error {
	if len(values) < 1 {
		return nil
	}
	argSQL := strings.Repeat(", (?, ?, ?, ?, ?)", len(values))[2:]
	const argLen = 5
	args := make([]interface{}, len(values)*argLen)
	for i, v := range values {
		args[i*argLen] = enrollmentID
		args[i*argLen+1] = v.Path
		args[i*argLen+2] = v.ContainerType
		args[i*argLen+3] = v.ValueType
		args[i*argLen+4] = v.Value
	}
	_, err := s.db.ExecContext(
		ctx, `
INSERT INTO status_values
    (
        enrollment_id,
        path,
        container_type,
        value_type,
        value
    )
VALUES
    `+argSQL+` as new
ON DUPLICATE KEY
UPDATE
    updated_at = CURRENT_TIMESTAMP;`,
		args...,
	)
	return err
}

func (s *MySQLStorage) storeStatusErrors(ctx context.Context, enrollmentID string, errors []ddm.StatusError) error {
	if len(errors) < 1 {
		return nil
	}
	argSQL := strings.Repeat(", (?, ?, ?)", len(errors))[2:]
	const argLen = 3
	args := make([]interface{}, len(errors)*argLen)
	for i, e := range errors {
		args[i*argLen] = enrollmentID
		args[i*argLen+1] = e.Path
		args[i*argLen+2] = e.ErrorJSON
	}
	_, err := s.db.ExecContext(
		ctx, `
INSERT INTO status_errors
    (
        enrollment_id,
        path,
        error
    )
VALUES
    `+argSQL+`;`,
		args...,
	)
	return err

}

// StoreDeclarationStatus stores the status report from enrollmentID.
// See also the storage package for documentation on the storage interfaces.
func (s *MySQLStorage) StoreDeclarationStatus(ctx context.Context, enrollmentID string, status *ddm.StatusReport) error {
	err := s.storeStatusDeclarations(ctx, enrollmentID, status.Declarations)
	if err != nil {
		return fmt.Errorf("storing declaration status: %w", err)
	}
	err = s.storeStatusValues(ctx, enrollmentID, status.Values)
	if err != nil {
		return fmt.Errorf("storing status values: %w", err)
	}
	err = s.storeStatusErrors(ctx, enrollmentID, status.Errors)
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
	idSQL := strings.Repeat(", ?", len(enrollmentIDs))[2:]
	valSQL := make([]interface{}, len(enrollmentIDs))
	for i, id := range enrollmentIDs {
		valSQL[i] = id
	}
	// intent is to query only the declaration statuses from the given
	// enrollment ids and only those that are currently actively
	// enabled and managed via an enrollment's configured sets
	rows, err := s.db.QueryContext(
		ctx, `
SELECT
    statusd.enrollment_id,
    statusd.declaration_identifier,
    statusd.active,
    statusd.valid,
    COALESCE(statusd.reasons, 'null'),
	statusd.server_token,
	statusd.updated_at,
    statusd.server_token = d.server_token AS current
FROM
    status_declarations statusd
    INNER JOIN declarations d
        ON statusd.declaration_identifier = d.identifier
    INNER JOIN set_declarations sd
        ON d.identifier = sd.declaration_identifier
    INNER JOIN enrollment_sets es
        ON sd.set_name = es.set_name AND statusd.enrollment_id = es.enrollment_id
WHERE
    statusd.enrollment_id IN (`+idSQL+`)
ORDER BY
    statusd.enrollment_id;`,
		valSQL...,
	)
	if err != nil {
		return nil, err
	}
	resp := make(map[string][]ddm.DeclarationQueryStatus)
	defer rows.Close()
	for rows.Next() {
		var id, updatedAt string
		var reasonJSON []byte
		var status ddm.DeclarationQueryStatus
		err = rows.Scan(
			&id,
			&status.Identifier,
			&status.Active,
			&status.Valid,
			&reasonJSON,
			&status.ServerToken,
			&updatedAt,
			&status.Current,
		)
		if err != nil {
			break
		}
		status.StatusReceived, err = time.Parse(mysqlTimeFormat, updatedAt)
		if err != nil {
			break
		}
		err = json.Unmarshal(reasonJSON, &status.Reasons)
		if err != nil {
			err = fmt.Errorf("parsing reason JSON: %w", err)
			break
		}
		el := resp[id]
		el = append(el, status)
		resp[id] = el
	}
	if err == nil {
		err = rows.Err()
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
	for rows.Next() {
		sErr := storage.StatusError{}
		err = rows.Scan(&id, &sErr.Path, &dbErrorJSON, &dbTimestamp)
		if err != nil {
			break
		}
		_ = json.Unmarshal(dbErrorJSON, &sErr.Error)
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
    value
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
		err = rows.Scan(
			&id,
			&sVal.Path,
			&sVal.Value,
		)
		if err != nil {
			break
		}
		resp[id] = append(resp[id], sVal)
	}
	if err == nil {
		err = rows.Err()
	}
	return resp, err
}
