package mysql

import "context"

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

// RetrieveDeclarationEnrollmentIDs retrieves a list of enrollment IDs that are transitively associated with a declaration.
func (s *MySQLStorage) RetrieveDeclarationEnrollmentIDs(ctx context.Context, declarationID string) ([]string, error) {
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
WHERE
    d.identifier = ?;`,
		declarationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var id string
	var ids []string
	for rows.Next() {
		err = rows.Scan(&id)
		if err != nil {
			break
		}
		ids = append(ids, id)
	}
	if err == nil {
		err = rows.Err()
	}
	return ids, err
}

// RetrieveSetEnrollmentIDs retrieves a list of enrollment IDs that are associated with a set.
func (s *MySQLStorage) RetrieveSetEnrollmentIDs(ctx context.Context, setName string) ([]string, error) {
	return s.singleStringColumn(
		ctx,
		`SELECT enrollment_id FROM enrollment_sets WHERE set_name = ?;`,
		setName,
	)
}
