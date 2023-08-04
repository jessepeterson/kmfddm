package main

import (
	"fmt"
	"hash"
	"strconv"
	"strings"

	"github.com/cespare/xxhash"
	"github.com/jessepeterson/kmfddm/log"
	"github.com/jessepeterson/kmfddm/log/logkeys"
	"github.com/jessepeterson/kmfddm/storage"
	"github.com/jessepeterson/kmfddm/storage/file"
	"github.com/jessepeterson/kmfddm/storage/mysql"

	_ "github.com/go-sql-driver/mysql"
)

type allStorage interface {
	storage.DeclarationAPIStorage
	storage.EnrollmentIDRetriever
	storage.EnrollmentDeclarationStorage
	storage.StatusStorer
	storage.SetDeclarationStorage
	storage.SetRetreiver
	storage.EnrollmentSetStorage
	storage.StatusAPIStorage
}

var hasher func() hash.Hash = func() hash.Hash { return xxhash.New() }

func setupStorage(name, dsn, options string, logger log.Logger) (allStorage, error) {
	logger = logger.With("storage", name)
	var mapOptions map[string]string
	if options != "" {
		mapOptions = splitOptions(options)
	}
	switch name {
	case "mysql":
		return setupMySQLStorage(dsn, mapOptions, logger)
	case "file":
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
