package pgsql

import (
	"context"
	"database/sql"
	"hash"
	"hash/fnv"
	"os"
	"testing"

	"github.com/jessepeterson/kmfddm/ddm"
	"github.com/jessepeterson/kmfddm/storage"
	"github.com/jessepeterson/kmfddm/test/e2e"

	_ "github.com/lib/pq"
)

func TestPgSQL(t *testing.T) {
	testDSN := os.Getenv("KMFDDM_PGSQL_STORAGE_TEST_DSN")
	if testDSN == "" {
		t.Skip("KMFDDM_PGSQL_STORAGE_TEST_DSN not set")
	}

	storage, err := New(func() hash.Hash { return fnv.New128() }, WithDSN(testDSN))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	t.Run("TestE2E", func(t *testing.T) {
		e2e.TestE2E(t, ctx, storage)
	})

	// keepErrors splits the errors of a status report
	const keepReports, keepErrors = 3, 2
	retentionStorage, err := New(
		func() hash.Hash { return fnv.New128() },
		WithDSN(testDSN),
		WithStatusReportDeletion(keepReports),
		WithErrorDeletion(keepErrors),
	)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("TestStatusRetention", func(t *testing.T) {
		e2e.TestStatusRetention(t, ctx, retentionStorage, keepReports, keepErrors)
	})
}

// newTestStorage connects one session in timeZone, so session-dependent SQL
// (e.g. timestamp to text casts) is exercised deterministically.
func newTestStorage(t *testing.T, timeZone string) *PgSQLStorage {
	t.Helper()
	testDSN := os.Getenv("KMFDDM_PGSQL_STORAGE_TEST_DSN")
	if testDSN == "" {
		t.Skip("KMFDDM_PGSQL_STORAGE_TEST_DSN not set")
	}
	db, err := sql.Open("postgres", testDSN)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	db.SetMaxOpenConns(1)
	if _, err = db.Exec("SET TIME ZONE '" + timeZone + "'"); err != nil {
		t.Fatal(err)
	}
	s, err := New(func() hash.Hash { return fnv.New128() }, WithDB(db))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestPgSQLServerTokenIgnoresSessionTimeZone(t *testing.T) {
	ctx := context.Background()
	utc := newTestStorage(t, "UTC")
	tokyo := newTestStorage(t, "Asia/Tokyo")

	d := &ddm.Declaration{
		Identifier: "pgsql_test_token_tz",
		Type:       "com.apple.configuration.management.test",
		Payload:    []byte(`{"Echo":"a"}`),
	}
	token := func(s *PgSQLStorage, payload string) string {
		t.Helper()
		d.Payload = []byte(payload)
		if _, err := s.StoreDeclaration(ctx, d); err != nil {
			t.Fatal(err)
		}
		got, err := s.RetrieveDeclaration(ctx, d.Identifier)
		if err != nil {
			t.Fatal(err)
		}
		return got.ServerToken
	}

	want := token(utc, `{"Echo":"a"}`)
	token(tokyo, `{"Echo":"b"}`)
	if have := token(tokyo, `{"Echo":"a"}`); have != want {
		t.Errorf("token for the same declaration differs by session time zone: have %s, want %s", have, want)
	}
}

func TestPgSQLStoresStatusReportExactly(t *testing.T) {
	ctx := context.Background()
	s := newTestStorage(t, "UTC")

	// \u0000 sits outside the parsed status items, so only the raw report holds it.
	raw := []byte(`{"StatusItems":{"x-vendor":{"note":"a\u0000b"}},  "Errors":[]}`)
	_, report, err := ddm.ParseStatus(raw)
	if err != nil {
		t.Fatal(err)
	}
	report.ID = "pgsql_test_exact"
	const enrollmentID = "pgsql_test_enr_exact"
	if err := s.StoreDeclarationStatus(ctx, enrollmentID, report); err != nil {
		t.Fatal(err)
	}

	index := 0
	got, err := s.RetrieveStatusReport(ctx, storage.StatusReportQuery{EnrollmentID: enrollmentID, Index: &index})
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || string(got.Raw) != string(raw) {
		t.Errorf("status report not stored as sent: have %q, want %q", got.Raw, raw)
	}
}
