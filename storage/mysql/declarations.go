package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jessepeterson/kmfddm/ddm"
	"github.com/jessepeterson/kmfddm/storage"
)

// StoreDeclaration stores a declaration and returns whether it changed or not.
// See also the storage package for documentation on the storage interfaces.
func (s *MySQLStorage) StoreDeclaration(ctx context.Context, d *ddm.Declaration) (bool, error) {
	result, err := s.db.ExecContext(
		ctx,
		`
INSERT INTO declarations
    (identifier, type, payload, server_token)
VALUES
    (?, ?, ?, SHA1(CONCAT(identifier, type, payload, CURRENT_TIMESTAMP, 0))) AS new
ON DUPLICATE KEY
UPDATE
    type    = new.type,
    payload = new.payload,
	server_token = SHA1(CONCAT(new.identifier, new.type, new.payload, created_at, touched_ct));`,
		d.Identifier,
		d.Type,
		d.Payload,
	)
	if err != nil {
		return false, err
	}
	return resultChangedRows(result)
}

// RetrieveDeclaration retrieves a declaration.
// See also the storage package for documentation on the storage interfaces.
func (s *MySQLStorage) RetrieveDeclaration(ctx context.Context, declarationID string) (*ddm.Declaration, error) {
	qd, err := s.q.GetDeclaration(ctx, declarationID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: %v", storage.ErrDeclarationNotFound, err)
	} else if err != nil {
		return nil, err
	}
	return &ddm.Declaration{
		Identifier:  qd.Identifier,
		Type:        qd.Type,
		ServerToken: qd.ServerToken,
		Payload:     qd.Payload,
		Raw:         qd.Declaration,
	}, nil
}

// RetrieveDeclarationModTime retrieves the last modification time of the declaration.
// See also the storage package for documentation on the storage interfaces.
func (s *MySQLStorage) RetrieveDeclarationModTime(ctx context.Context, declarationID string) (time.Time, error) {
	var dbTimestamp string
	if err := s.db.QueryRowContext(ctx, "SELECT updated_at FROM declarations WHERE identifier = ? LIMIT 1;", declarationID).Scan(&dbTimestamp); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = fmt.Errorf("%w: %v", storage.ErrDeclarationNotFound, err)
		}
		return time.Time{}, err
	}
	return time.Parse(mysqlTimeFormat, dbTimestamp)
}

// DeleteDeclaration deletes a declaration and returns whether it was deleted or already existed.
// See also the storage package for documentation on the storage interfaces.
func (s *MySQLStorage) DeleteDeclaration(ctx context.Context, declarationID string) (bool, error) {
	result, err := s.db.ExecContext(
		ctx,
		`DELETE FROM declarations WHERE identifier = ?;`,
		declarationID,
	)
	if err != nil {
		return false, err
	}
	return resultChangedRows(result)
}

// RetrieveDeclarationSets returns the list of sets a declaration is a part of.
// See also the storage package for documentation on the storage interfaces.
func (s *MySQLStorage) RetrieveDeclarationSets(ctx context.Context, declarationID string) ([]string, error) {
	return s.singleStringColumn(
		ctx,
		`SELECT set_name FROM set_declarations WHERE declaration_identifier = ?;`,
		declarationID,
	)
}

// RetrieveDeclarations returns the list of declaration IDs.
// See also the storage package for documentation on the storage interfaces.
func (s *MySQLStorage) RetrieveDeclarations(ctx context.Context) ([]string, error) {
	return s.singleStringColumn(
		ctx,
		`SELECT identifier FROM declarations;`,
	)
}

// TouchDeclaration updates a declaration's "touch count" which makes a new server token.
// See also the storage package for documentation on the storage interfaces.
func (s *MySQLStorage) TouchDeclaration(ctx context.Context, declarationID string) error {
	result, err := s.db.ExecContext(
		ctx,
		`
UPDATE
    declarations
SET
    touched_ct = touched_ct + 1,
    server_token = SHA1(CONCAT(identifier, type, payload, created_at, touched_ct))
WHERE
    identifier = ?;`,
		declarationID,
	)
	if err != nil {
		return err
	}
	changed, err := resultChangedRows(result)
	if err != nil {
		return err
	}
	if !changed {
		return fmt.Errorf("%w: declaration not touched (may not exist)", storage.ErrDeclarationNotFound)
	}
	return nil
}
