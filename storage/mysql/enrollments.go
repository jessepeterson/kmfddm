package mysql

import (
	"context"
	"errors"
	"strings"
)

// RetrieveEnrollmentSets retrieves the list of sets an enrollment is assigned to.
// See also the storage package for documentation on the storage interfaces.
func (s *MySQLStorage) RetrieveEnrollmentSets(ctx context.Context, enrollmentID string) ([]string, error) {
	return s.singleStringColumn(
		ctx,
		`SELECT set_name FROM enrollment_sets WHERE enrollment_id = ?;`,
		enrollmentID,
	)
}

// StoreEnrollmentSet creates the association between an enrollment and a set.
// See also the storage package for documentation on the storage interfaces.
func (s *MySQLStorage) StoreEnrollmentSet(ctx context.Context, enrollmentID, setName string) (bool, error) {
	result, err := s.db.ExecContext(
		ctx, `
INSERT INTO enrollment_sets
    (enrollment_id, set_name)
VALUES
    (?, ?)
ON DUPLICATE KEY
UPDATE
    enrollment_id = enrollment_id;`,
		enrollmentID,
		setName,
	)
	if err != nil {
		return false, err
	}
	return resultChangedRows(result)
}

// RemoveEnrollmentSet removes the association between an enrollment and a set.
// See also the storage package for documentation on the storage interfaces.
func (s *MySQLStorage) RemoveEnrollmentSet(ctx context.Context, enrollmentID, setName string) (bool, error) {
	result, err := s.db.ExecContext(
		ctx, `
DELETE FROM enrollment_sets
WHERE
    enrollment_id = ? AND
    set_name = ?;`,
		enrollmentID,
		setName,
	)
	if err != nil {
		return false, err
	}
	return resultChangedRows(result)
}

func qAndP(params []string) (r string, p []interface{}) {
	if len(params) < 1 {
		return
	}
	r = strings.Repeat(", ?", len(params))[2:]
	for _, j := range params {
		p = append(p, j)
	}
	return
}

// RemoveAllEnrollmentSets dissociates enrollment ID from any sets.
// If any associations are removed true is returned.
// It should not be an error if no associations exist.
func (s *MySQLStorage) RemoveAllEnrollmentSets(ctx context.Context, enrollmentID string) (bool, error) {
	r, err := s.q.RemoveAllEnrollmentSets(ctx, enrollmentID)
	if err != nil {
		return false, err
	}
	return resultChangedRows(r)
}

// RetrieveEnrollmentIDs retrieves enrollment IDs.
// See also the storage package for documentation on the storage interfaces.
func (s *MySQLStorage) RetrieveEnrollmentIDs(ctx context.Context, declarations []string, sets []string, ids []string) ([]string, error) {
	var where []string
	var params []interface{}
	if len(declarations) > 0 {
		r, p := qAndP(declarations)
		q := "d.identifier IN (" + r + ")"
		params = append(params, p...)
		where = append(where, q)
	}
	if len(sets) > 0 {
		r, p := qAndP(sets)
		q := "es.set_name IN (" + r + ")"
		params = append(params, p...)
		where = append(where, q)
	}
	if len(ids) > 0 {
		r, p := qAndP(ids)
		q := "es.enrollment_id IN (" + r + ")"
		params = append(params, p...)
		where = append(where, q)
	}
	if len(params) < 1 {
		return nil, errors.New("no parameters provided")
	}
	rows, err := s.db.QueryContext(
		ctx, `
SELECT DISTINCT
    es.enrollment_id
FROM
    enrollment_sets es
    LEFT JOIN set_declarations sd
        ON sd.set_name = es.set_name
    LEFT JOIN declarations d
        ON d.identifier = sd.declaration_identifier
    WHERE `+strings.Join(where, " OR "),
		params...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	retIDMap := make(map[string]struct{})
	var retID string
	for rows.Next() {
		err = rows.Scan(&retID)
		if err != nil {
			break
		}
		retIDMap[retID] = struct{}{}
	}
	if err == nil {
		err = rows.Err()
	}
	// merge in the enrollment IDs directly supplied in params
	for _, id := range ids {
		retIDMap[id] = struct{}{}
	}
	// convert back to slice for return value
	retIDs := make([]string, 0, len(retIDMap))
	for k := range retIDMap {
		retIDs = append(retIDs, k)
	}
	return retIDs, err
}
