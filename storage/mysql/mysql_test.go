package mysql

import (
	"context"
	"hash"
	"os"
	"testing"

	"github.com/cespare/xxhash"
	"github.com/jessepeterson/kmfddm/storage/test"
	"github.com/jessepeterson/kmfddm/test/e2e"

	_ "github.com/go-sql-driver/mysql"
)

func TestMySQL(t *testing.T) {
	testDSN := os.Getenv("KMFDDM_MYSQL_STORAGE_TEST_DSN")
	if testDSN == "" {
		t.Skip("KMFDDM_MYSQL_STORAGE_TEST_DSN not set")
	}

	storage, err := New(func() hash.Hash { return xxhash.New() }, WithDSN(testDSN))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	test.TestBasic(t, storage, ctx)
	test.TestBasicStatus(t, "../test", storage, ctx)
	t.Run("TestE2E", func(t *testing.T) {
		e2e.TestE2E(t, ctx, storage)
	})

}
