package main

import (
	"fmt"
	"hash"

	"github.com/cespare/xxhash"
	"github.com/jessepeterson/kmfddm/http/api"
	"github.com/jessepeterson/kmfddm/http/ddm"
	"github.com/jessepeterson/kmfddm/notifier"
	"github.com/jessepeterson/kmfddm/storage/file"
	"github.com/jessepeterson/kmfddm/storage/mysql"

	_ "github.com/go-sql-driver/mysql"
)

type allStorage interface {
	notifier.EnrollmentIDFinder
	api.SetAPIStorage
	api.DeclarationAPIStorage
	ddm.DeclarationRetriever
	ddm.TokensDeclarationItemsRetriever
	ddm.StatusStorage
	api.EnrollmentAPIStorage
	api.StatusAPIStorage
}

func storage(name, dsn string) (allStorage, error) {
	switch name {
	case "mysql":
		return mysql.New(
			mysql.WithDSN(dsn),
			mysql.WithNewHash(func() hash.Hash { return xxhash.New() }),
		)
	case "file":
		if dsn == "" {
			dsn = "db"
		}
		return file.New(dsn)
	default:
		return nil, fmt.Errorf("unknown storage name: %s", name)
	}
}
