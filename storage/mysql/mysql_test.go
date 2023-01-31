//go:build integration
// +build integration

package mysql

import (
	"context"
	"flag"
	"testing"

	"github.com/jessepeterson/kmfddm/storage/internal/test"

	_ "github.com/go-sql-driver/mysql"
)

var flDSN = flag.String("dsn", "", "DSN of test MySQL instance")

func TestMySQL(t *testing.T) {
	if *flDSN == "" {
		t.Fatal("MySQL DSN flag not provided to test")
	}

	storage, err := New(WithDSN(*flDSN))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	test.TestDeclarations(t, storage, ctx)
}
