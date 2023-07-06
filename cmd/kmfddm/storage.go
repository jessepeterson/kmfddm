package main

import (
	"fmt"
	"hash"

	"github.com/cespare/xxhash"
	"github.com/jessepeterson/kmfddm/http/api"
	"github.com/jessepeterson/kmfddm/http/ddm"
	"github.com/jessepeterson/kmfddm/storage"
	"github.com/jessepeterson/kmfddm/storage/file"
	"github.com/jessepeterson/kmfddm/storage/mysql"

	_ "github.com/go-sql-driver/mysql"
)

type allStorage interface {
	api.SetAPIStorage
	api.DeclarationAPIStorage
	ddm.StatusStorage
	api.EnrollmentAPIStorage
	api.StatusAPIStorage
	storage.Toucher
	storage.EnrollmentIDRetriever
	storage.DeclarationRetriever
	storage.TokensDeclarationItemsRetriever
}

var hasher func() hash.Hash = func() hash.Hash { return xxhash.New() }

func setupStorage(name, dsn string) (allStorage, error) {
	switch name {
	case "mysql":
		return mysql.New(
			hasher,
			mysql.WithDSN(dsn),
		)
	case "file":
		if dsn == "" {
			dsn = "db"
		}
		return file.New(dsn, hasher)
	default:
		return nil, fmt.Errorf("unknown storage name: %s", name)
	}
}
