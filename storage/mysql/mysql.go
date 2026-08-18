// Package mysql is a MySQL storage backend for KMFDDM.
package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"hash"
	"time"

	"github.com/jessepeterson/kmfddm/storage/mysql/sqlc"
)

const mysqlTimeFormat = "2006-01-02 15:04:05"

// Connection pool recycling is left at database/sql's defaults unless
// configured. Pooled connections should be recycled well before the shortest
// idle timeout in the network path (the istio-proxy sidecar reaps idle TCP
// connections at ~1h, Aurora's wait_timeout is longer). Without recycling,
// database/sql hands out long-idle connections that the far end has already
// closed, producing "broken pipe" writes and "closing bad idle connection:
// EOF" errors. Set WithConnMaxLifetime / WithConnMaxIdleTime per-deployment to
// enable recycling for the path's idle timeout.

// MySQLStorage implements a MySQL storage backend.
type MySQLStorage struct {
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

// WithErrorDeletion sets the maximum number of error event rows to keep
// per enrollment ID.
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

// New creates and initializes a new MySQL storage backend.
// New attempts to Ping the database after opening to verify connectivity.
func New(newHash func() hash.Hash, opts ...Option) (*MySQLStorage, error) {
	if newHash == nil {
		panic("nil hasher")
	}
	cfg := config{
		driver: "mysql",
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
	return &MySQLStorage{
		db:      cfg.db,
		q:       sqlc.New(cfg.db),
		newHash: newHash,
		errDel:  cfg.errDel,
		stsDel:  cfg.stsDel,
		noSts:   cfg.noSts,
	}, nil
}

// resultChangedRows tries to tell us if if the record changed. Note that
// MySQL has an odd special case for result rows when INSERT INTO ... ON
// DUPLICATE KEY is used. The manual states 0 is returned for no change,
// 1 for an INSERT and 2 for UPDATE (per row).
func resultChangedRows(r sql.Result) (bool, error) {
	rowCt, err := r.RowsAffected()
	if err != nil {
		// assume the row changed because (presumably) the query succeeded
		return true, err
	}
	return rowCt > 0, nil
}

// singleStringColumn executes sql with args using ctx and expects a single
// column string to return all the rows in a string slice.
func (s *MySQLStorage) singleStringColumn(ctx context.Context, sql string, args ...interface{}) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var str string
	var strs []string
	for rows.Next() {
		err = rows.Scan(&str)
		if err != nil {
			break
		}
		strs = append(strs, str)
	}
	if err == nil {
		err = rows.Err()
	}
	return strs, err
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
