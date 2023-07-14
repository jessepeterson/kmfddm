package mysql

import (
	"context"
	"errors"
	"strings"
)

func (s *MySQLStorage) RetrieveEnrollmentSets(ctx context.Context, enrollmentID string) ([]string, error) {
	return s.singleStringColumn(
		ctx,
		`SELECT set_name FROM enrollment_sets WHERE enrollment_id = ?;`,
		enrollmentID,
	)
}

// StoreEnrollmentSet creates the association between an enrollment and a set.
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
		q := "sd.set_name IN (" + r + ")"
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
    declarations d
    INNER JOIN set_declarations sd
        ON d.identifier = sd.declaration_identifier
    INNER JOIN enrollment_sets es
        ON sd.set_name = es.set_name
    WHERE `+strings.Join(where, " OR "),
		params...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var retIDs []string
	var retID string
	for rows.Next() {
		err = rows.Scan(&retID)
		if err != nil {
			break
		}
		retIDs = append(retIDs, retID)
	}
	if err == nil {
		err = rows.Err()
	}
	return retIDs, err
}
