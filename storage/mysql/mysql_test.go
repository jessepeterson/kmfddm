//go:build integration
// +build integration

package mysql

import (
	"context"
	"flag"
	"hash"
	"testing"

	"github.com/cespare/xxhash"
	"github.com/jessepeterson/kmfddm/storage/test"

	_ "github.com/go-sql-driver/mysql"
)

var flDSN = flag.String("dsn", "", "DSN of test MySQL instance")

func TestMySQL(t *testing.T) {
	if *flDSN == "" {
		t.Fatal("MySQL DSN flag not provided to test")
	}

	storage, err := New(func() hash.Hash { return xxhash.New() }, WithDSN(*flDSN))
	if err != nil {
		t.Fatal(err)
	}

	test.TestBasic(t, storage, context.Background())
}
