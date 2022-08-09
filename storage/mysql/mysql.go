package mysql

import (
	"context"
	"database/sql"
	"errors"
	"hash"
)

const mysqlTimeFormat = "2006-01-02 15:04:05"

type MySQLStorage struct {
	db      *sql.DB
	newHash func() hash.Hash
}

type config struct {
	driver  string
	dsn     string
	db      *sql.DB
	newHash func() hash.Hash
}

type Option func(*config)

func WithDSN(dsn string) Option {
	return func(c *config) {
		c.dsn = dsn
	}
}

func WithDriver(driver string) Option {
	return func(c *config) {
		c.driver = driver
	}
}

func WithDB(db *sql.DB) Option {
	return func(c *config) {
		c.db = db
	}
}

func WithNewHash(newHash func() hash.Hash) Option {
	return func(c *config) {
		c.newHash = newHash
	}
}

func New(opts ...Option) (*MySQLStorage, error) {
	cfg := &config{driver: "mysql"}
	for _, opt := range opts {
		opt(cfg)
	}
	var err error
	if cfg.db == nil {
		cfg.db, err = sql.Open(cfg.driver, cfg.dsn)
		if err != nil {
			return nil, err
		}
	}
	if err = cfg.db.Ping(); err != nil {
		return nil, err
	}
	return &MySQLStorage{db: cfg.db, newHash: cfg.newHash}, nil
}

var ErrNotImplemented = errors.New("not implemented")

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
