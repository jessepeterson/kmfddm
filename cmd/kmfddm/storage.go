package main

import (
	"errors"
	"fmt"
	"hash"
	"strconv"
	"strings"

	"github.com/jessepeterson/kmfddm/logkeys"
	"github.com/jessepeterson/kmfddm/storage"
	"github.com/jessepeterson/kmfddm/storage/diskv"
	"github.com/jessepeterson/kmfddm/storage/file"
	"github.com/jessepeterson/kmfddm/storage/inmem"
	"github.com/jessepeterson/kmfddm/storage/mysql"

	"github.com/cespare/xxhash"
	_ "github.com/go-sql-driver/mysql"
	"github.com/micromdm/nanolib/log"
)

var ErrOptionsNotSupported = errors.New("storage options not supported")

type allStorage interface {
	storage.DeclarationAPIStorage
	storage.EnrollmentIDRetriever
	storage.EnrollmentDeclarationStorage
	storage.StatusStorer
	storage.SetDeclarationStorage
	storage.SetRetreiver
	storage.EnrollmentSetStorage
	storage.StatusAPIStorage
	storage.EnrollmentDeclarationDataStorage
}

var hasher func() hash.Hash = func() hash.Hash { return xxhash.New() }

func setupStorage(name, dsn, options string, logger log.Logger) (allStorage, error) {
	logger = logger.With("storage", name)
	var mapOptions map[string]string
	if options != "" {
		mapOptions = splitOptions(options)
	}
	switch name {
	case "filekv":
		if dsn == "" {
			dsn = "dbkv"
		}
		if options != "" {
			return nil, ErrOptionsNotSupported
		}
		return diskv.New(dsn, hasher), nil
	case "mysql":
		return setupMySQLStorage(dsn, mapOptions, logger)
	case "inmem":
		if options != "" {
			return nil, ErrOptionsNotSupported
		}
		return inmem.New(hasher), nil
	case "file":
		if options != "enable_deprecated=1" {
			return nil, errors.New("file backend is deprecated; specify storage options to force enable")
		}
		if dsn == "" {
			dsn = "db"
		}
		return file.New(dsn, hasher)
	default:
		return nil, fmt.Errorf("unknown storage name: %s", name)
	}
}

func setupMySQLStorage(dsn string, options map[string]string, logger log.Logger) (allStorage, error) {
	opts := []mysql.Option{mysql.WithDSN(dsn)}
	for k, v := range options {
		switch k {
		case "delete_errors":
			const errorDeleteOption = "error delete option"
			n, err := strconv.ParseUint(v, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid value for %s: %w", errorDeleteOption, err)
			}
			opts = append(opts, mysql.WithErrorDeletion(uint(n)))
			logger.Debug(logkeys.Message, errorDeleteOption, logkeys.GenericCount, int(n))
		case "delete_status_reports":
			const reportDeleteOption = "status report delete option"
			n, err := strconv.ParseUint(v, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid value for %s: %w", reportDeleteOption, err)
			}
			opts = append(opts, mysql.WithStatusReportDeletion(uint(n)))
			logger.Debug(logkeys.Message, reportDeleteOption, logkeys.GenericCount, int(n))
		default:
			return nil, fmt.Errorf("invalid option: %q", k)
		}
	}
	return mysql.New(hasher, opts...)
}

func splitOptions(s string) map[string]string {
	out := make(map[string]string)
	opts := strings.Split(s, ",")
	for _, opt := range opts {
		optKAndV := strings.SplitN(opt, "=", 2)
		if len(optKAndV) < 2 {
			optKAndV = append(optKAndV, "")
		}
		out[optKAndV[0]] = optKAndV[1]
	}
	return out
}
