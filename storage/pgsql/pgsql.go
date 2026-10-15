// Package pgsql is a PostgreSQL storage backend for KMFDDM.
package pgsql

import (
	"context"
	"database/sql"
	"fmt"
	"hash"
	"time"

	"github.com/jessepeterson/kmfddm/storage/pgsql/sqlc"
)

// PgSQLStorage implements a PostgreSQL storage backend.
type PgSQLStorage struct {
	db      *sql.DB
	q       *sqlc.Queries
	newHash func() hash.Hash
	errDel  uint
	stsDel  uint
	noSts   bool
}

type config struct {
	driver          string
	dsn             string
	db              *sql.DB
	errDel          uint
	stsDel          uint
	noSts           bool
	connMaxLifetime time.Duration
	connMaxIdleTime time.Duration
}

type Option func(*config)

// WithDSN configures the Data Source Name (DSN) when opening the database.
func WithDSN(dsn string) Option {
	return func(c *config) {
		c.dsn = dsn
	}
}

// WithDriver configures the name of driver when opening the database.
func WithDriver(driver string) Option {
	return func(c *config) {
		c.driver = driver
	}
}

// WithDB configures the backend to use db. If configured the backend
// will not attempt to open the database itself.
func WithDB(db *sql.DB) Option {
	return func(c *config) {
		c.db = db
	}
}

// WithErrorDeletion sets the maximum number of errors to keep per
// enrollment ID. Note this counts errors, not status reports: a single
// status report can contain many errors.
func WithErrorDeletion(count uint) Option {
	return func(c *config) {
		c.errDel = count
	}
}

// WithStatusReportDeletion sets the maximum number of status reports
// rows to keep per enrollment ID.
func WithStatusReportDeletion(count uint) Option {
	return func(c *config) {
		c.stsDel = count
	}
}

// WithoutStatusReports disables saving of status reports altogether.
func WithoutStatusReports() Option {
	return func(c *config) {
		c.noSts = true
	}
}

// WithConnMaxLifetime sets the maximum amount of time a connection may be
// reused. It should be shorter than the shortest idle timeout in the network
// path to the database. A non-positive value keeps connections forever.
func WithConnMaxLifetime(d time.Duration) Option {
	return func(c *config) {
		c.connMaxLifetime = d
	}
}

// WithConnMaxIdleTime sets the maximum amount of time a connection may be idle
// before it is closed. A non-positive value never closes connections due to
// idle time.
func WithConnMaxIdleTime(d time.Duration) Option {
	return func(c *config) {
		c.connMaxIdleTime = d
	}
}

// New creates and initializes a new PostgreSQL storage backend.
// New attempts to Ping the database after opening to verify connectivity.
func New(newHash func() hash.Hash, opts ...Option) (*PgSQLStorage, error) {
	if newHash == nil {
		panic("nil hasher")
	}
	cfg := config{
		driver: "postgres",
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	var err error
	if cfg.db == nil {
		cfg.db, err = sql.Open(cfg.driver, cfg.dsn)
		if err != nil {
			return nil, err
		}
	}
	if cfg.connMaxLifetime != 0 {
		cfg.db.SetConnMaxLifetime(cfg.connMaxLifetime)
	}
	if cfg.connMaxIdleTime != 0 {
		cfg.db.SetConnMaxIdleTime(cfg.connMaxIdleTime)
	}
	if err = cfg.db.Ping(); err != nil {
		return nil, err
	}
	return &PgSQLStorage{
		db:      cfg.db,
		q:       sqlc.New(cfg.db),
		newHash: newHash,
		errDel:  cfg.errDel,
		stsDel:  cfg.stsDel,
		noSts:   cfg.noSts,
	}, nil
}

// nullEmptyString returns a NULL string if s is empty.
func nullEmptyString(s string) sql.NullString {
	return sql.NullString{
		String: s,
		Valid:  s != "",
	}
}

// resultChangedRows reports whether any rows were affected. The upserts in
// this package only touch a row when its values change, so this mirrors
// MySQL's "0 rows affected" for an unchanged INSERT ... ON DUPLICATE KEY.
func resultChangedRows(r sql.Result) (bool, error) {
	rowCt, err := r.RowsAffected()
	if err != nil {
		// assume the row changed because (presumably) the query succeeded
		return true, err
	}
	return rowCt > 0, nil
}

// tx wraps g in transactions using db.
// If g returns an err the transaction will be rolled back; otherwise committed.
func tx(ctx context.Context, db *sql.DB, q *sqlc.Queries, g func(ctx context.Context, tx *sql.Tx, qtx *sqlc.Queries) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("tx begin: %w", err)
	}
	if err = g(ctx, tx, q.WithTx(tx)); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx rollback: %w; while trying to handle error: %v", rbErr, err)
		}
		return fmt.Errorf("tx rolled back: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}
	return nil
}
