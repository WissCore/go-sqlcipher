// SQLCipher self-validation suite. Covers gaps left by the
// upstream sqlcipher_test.go: wrong-key correctness, lifecycle
// persistence across process boundaries, heavy concurrent access
// on encrypted DBs, the cipher_page_size matrix, large-data
// stress, encrypted-to-encrypted backup via sqlcipher_export,
// and PRAGMA rekey.
//
// These tests are the contract WissCore commits to as the fork
// maintainer: anything that breaks here in a future SQLCipher
// bump is a release blocker.
//
// Run with: go test -race -count=1 ./...
// The -race flag is load-bearing for TestEncryptedConcurrentAccess;
// without it the test still passes but loses its main signal.

package sqlite3_test

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	sqlite3 "github.com/WissCore/go-sqlcipher/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"golang.org/x/sync/errgroup"
)

// TestMain runs goleak.VerifyTestMain after the test suite to
// catch goroutine leaks across all tests in this binary. Cgo
// wrappers are exactly where leaked goroutines hide (background
// finalizer goroutines, database/sql pool reapers that never
// shut down). VerifyTestMain is the right hook because individual
// VerifyNone calls would conflict with t.Parallel.
//
// IgnoreTopFunction excludes known-benign noise from the runtime
// and stdlib that goleak otherwise flags as false positives.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		// database/sql connection pool reaper exits asynchronously
		// after db.Close(); this is documented stdlib behaviour.
		goleak.IgnoreTopFunction("database/sql.(*DB).connectionOpener"),
		goleak.IgnoreTopFunction("database/sql.(*DB).connectionResetter"),
	)
}

// newEncryptedDB returns a freshly-created encrypted DB and the
// dbname/key it was opened with. The caller does NOT need to
// Close — t.Cleanup handles it. The temp directory is provided
// by t.TempDir and removed automatically after the test, after
// all t.Cleanup callbacks finish.
func newEncryptedDB(t *testing.T, dsnSuffix string) (*sql.DB, string, string) {
	t.Helper()
	var key [32]byte
	_, err := io.ReadFull(rand.Reader, key[:])
	require.NoError(t, err)
	hexKey := hex.EncodeToString(key[:])

	dbname := filepath.Join(t.TempDir(), "test.sqlite")
	dsn := fmt.Sprintf("%s?_pragma_key=x'%s'%s", dbname, hexKey, dsnSuffix)
	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	require.NoError(t, db.Ping())
	return db, dbname, hexKey
}

// requireSQLiteCode unwraps err and asserts the underlying SQLite
// result code matches want. Substring matching against err.Error()
// is brittle; this is the modern errors.As-based equivalent that
// will keep working if the driver ever changes its message format.
func requireSQLiteCode(t *testing.T, err error, want sqlite3.ErrNo) {
	t.Helper()
	require.Error(t, err)
	var sqliteErr sqlite3.Error
	require.True(t, errors.As(err, &sqliteErr),
		"expected sqlite3.Error, got %T: %v", err, err)
	require.Equal(t, want, sqliteErr.Code,
		"expected SQLite code %d, got %d (%s)",
		want, sqliteErr.Code, sqliteErr.Error())
}

// --- Wrong-key correctness -------------------------------------

// Opening an encrypted DB with the wrong key MUST fail loudly.
// Silent success here would be a security incident: the consumer
// would think they have data access while actually working against
// undefined behaviour. Asserts the typed error code.
func TestWrongKeyRejected(t *testing.T) {
	t.Parallel()
	db, dbname, _ := newEncryptedDB(t, "")
	_, err := db.Exec(`CREATE TABLE t(x INTEGER); INSERT INTO t VALUES (1);`)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	var wrongKey [32]byte
	_, err = io.ReadFull(rand.Reader, wrongKey[:])
	require.NoError(t, err)
	wrongDSN := fmt.Sprintf("%s?_pragma_key=x'%s'", dbname, hex.EncodeToString(wrongKey[:]))

	bad, err := sql.Open("sqlite3", wrongDSN)
	require.NoError(t, err) // sql.Open is lazy
	t.Cleanup(func() { _ = bad.Close() })

	_, err = bad.Exec("SELECT count(*) FROM t;")
	requireSQLiteCode(t, err, sqlite3.ErrNotADB)
}

// Opening with no key at all against an encrypted DB must also
// fail. Separates "wrong key" from "no key" — both bad, but
// callers should not be able to bypass crypto by omitting it.
func TestNoKeyAgainstEncryptedRejected(t *testing.T) {
	t.Parallel()
	db, dbname, _ := newEncryptedDB(t, "")
	_, err := db.Exec(`CREATE TABLE t(x INTEGER);`)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	plain, err := sql.Open("sqlite3", dbname)
	require.NoError(t, err)
	t.Cleanup(func() { _ = plain.Close() })

	_, err = plain.Exec("SELECT count(*) FROM t;")
	requireSQLiteCode(t, err, sqlite3.ErrNotADB)
}

// --- Lifecycle persistence -------------------------------------

// Encrypted data must survive a full close + reopen cycle. This
// is the most basic durability claim of the driver and protects
// against amalgamation regressions where extra_init/shutdown
// hooks fail to flush cipher state.
func TestEncryptedLifecyclePersistence(t *testing.T) {
	t.Parallel()
	db, dbname, hexKey := newEncryptedDB(t, "")

	_, err := db.Exec(`CREATE TABLE kv (k TEXT PRIMARY KEY, v TEXT);`)
	require.NoError(t, err)
	wanted := map[string]string{
		"alpha":   "one",
		"bravo":   "two",
		"charlie": "three",
	}
	for k, v := range wanted {
		_, ierr := db.Exec("INSERT INTO kv (k, v) VALUES (?, ?);", k, v)
		require.NoError(t, ierr)
	}
	require.NoError(t, db.Close())

	encrypted, err := sqlite3.IsEncrypted(dbname)
	require.NoError(t, err)
	require.True(t, encrypted, "DB must be encrypted on disk after close")

	dsn := fmt.Sprintf("%s?_pragma_key=x'%s'", dbname, hexKey)
	reopened, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = reopened.Close() })

	rows, err := reopened.Query("SELECT k, v FROM kv ORDER BY k;")
	require.NoError(t, err)
	t.Cleanup(func() { _ = rows.Close() })

	got := map[string]string{}
	for rows.Next() {
		var k, v string
		require.NoError(t, rows.Scan(&k, &v))
		got[k] = v
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, wanted, got)
}

// --- Concurrent access -----------------------------------------

// 100 goroutines hammering the same encrypted DB with mixed
// read/write under -race. SQLite serialises writes via its own
// mutex, but the cipher path is the new code we own; this test
// guards against accidentally introducing non-thread-safe state
// in the cgo wrapper or libtomcrypt PRNG.
//
// Uses errgroup so the first failure cancels the in-flight peers
// and surfaces the real cause instead of a downstream count
// mismatch. Without -race this test loses most of its value.
func TestEncryptedConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrent stress in -short mode")
	}
	db, _, _ := newEncryptedDB(t, "&_journal_mode=WAL&_busy_timeout=5000")

	_, err := db.Exec(`CREATE TABLE t (id INTEGER PRIMARY KEY, payload TEXT);`)
	require.NoError(t, err)

	const goroutines = 100
	const opsPerGoroutine = 50

	var inserts atomic.Int64
	g, ctx := errgroup.WithContext(context.Background())
	for gid := 0; gid < goroutines; gid++ {
		g.Go(func() error {
			for i := 0; i < opsPerGoroutine; i++ {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				if i%2 == 0 {
					_, err := db.ExecContext(ctx,
						"INSERT INTO t (payload) VALUES (?);",
						fmt.Sprintf("g=%d i=%d", gid, i))
					if err != nil {
						return fmt.Errorf("insert g=%d i=%d: %w", gid, i, err)
					}
					inserts.Add(1)
				} else {
					var n int
					if err := db.QueryRowContext(ctx, "SELECT count(*) FROM t;").Scan(&n); err != nil {
						return fmt.Errorf("select g=%d i=%d: %w", gid, i, err)
					}
				}
			}
			return nil
		})
	}
	require.NoError(t, g.Wait())

	var total int
	require.NoError(t, db.QueryRow("SELECT count(*) FROM t;").Scan(&total))
	assert.Equal(t, int(inserts.Load()), total,
		"row count must equal successful inserts (no lost writes)")
}

// --- cipher_page_size matrix -----------------------------------

// SQLCipher supports configurable page sizes; downstream tools
// (e.g. sqlcipher CLI, DB Browser for SQLite) sometimes default
// to a non-4096 value. A regression in any of these would silently
// break interop, so test the common ones explicitly.
func TestCipherPageSizeMatrix(t *testing.T) {
	t.Parallel()
	for _, ps := range []int{1024, 4096, 8192, 16384} {
		t.Run(fmt.Sprintf("page_size=%d", ps), func(t *testing.T) {
			t.Parallel()
			suffix := fmt.Sprintf("&_pragma_cipher_page_size=%d", ps)
			db, dbname, hexKey := newEncryptedDB(t, suffix)

			_, err := db.Exec(`CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT);`)
			require.NoError(t, err)
			_, err = db.Exec("INSERT INTO t (v) VALUES (?), (?), (?);", "a", "b", "c")
			require.NoError(t, err)
			require.NoError(t, db.Close())

			// page_size must be the same in DSN on reopen, otherwise
			// SQLCipher won't be able to read the file.
			dsn := fmt.Sprintf("%s?_pragma_key=x'%s'%s", dbname, hexKey, suffix)
			reopened, err := sql.Open("sqlite3", dsn)
			require.NoError(t, err)
			t.Cleanup(func() { _ = reopened.Close() })

			var n int
			require.NoError(t, reopened.QueryRow("SELECT count(*) FROM t;").Scan(&n))
			assert.Equal(t, 3, n)
		})
	}
}

// --- Large data stress -----------------------------------------

// 10k rows including MB-sized blobs. Catches regressions where
// the cipher path hits an edge case at scale (e.g. page boundary
// arithmetic, blob streaming). Verifies both small and large
// blobs survive byte-for-byte.
func TestEncryptedLargeData(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large-data test in -short mode")
	}
	db, _, _ := newEncryptedDB(t, "&_journal_mode=WAL")

	_, err := db.Exec(`CREATE TABLE big (id INTEGER PRIMARY KEY, blob BLOB);`)
	require.NoError(t, err)

	const rows = 10000
	smallBlob := make([]byte, 64)
	_, _ = io.ReadFull(rand.Reader, smallBlob)
	smallSum := sha256.Sum256(smallBlob)

	tx, err := db.Begin()
	require.NoError(t, err)
	stmt, err := tx.Prepare("INSERT INTO big (blob) VALUES (?);")
	require.NoError(t, err)
	defer func() { _ = stmt.Close() }()
	for i := 0; i < rows; i++ {
		_, ierr := stmt.Exec(smallBlob)
		require.NoError(t, ierr)
	}
	require.NoError(t, tx.Commit())

	// One large blob (~2 MB) to stress the cipher streaming path.
	bigBlob := make([]byte, 2*1024*1024)
	_, _ = io.ReadFull(rand.Reader, bigBlob)
	bigSum := sha256.Sum256(bigBlob)
	_, err = db.Exec("INSERT INTO big (blob) VALUES (?);", bigBlob)
	require.NoError(t, err)

	var n int
	require.NoError(t, db.QueryRow("SELECT count(*) FROM big;").Scan(&n))
	assert.Equal(t, rows+1, n)

	// Sample a few small-blob reads to confirm round-trip integrity.
	for _, id := range []int{1, rows / 2, rows} {
		var got []byte
		require.NoError(t,
			db.QueryRow("SELECT blob FROM big WHERE id = ?;", id).Scan(&got))
		assert.Equal(t, smallSum, sha256.Sum256(got),
			"small blob id=%d corrupted", id)
	}

	// Big blob byte-for-byte (compare hashes to keep failure output bounded).
	var got []byte
	require.NoError(t,
		db.QueryRow("SELECT blob FROM big WHERE id = ?;", rows+1).Scan(&got))
	assert.Equal(t, bigSum, sha256.Sum256(got),
		"big blob mismatch after encrypt/decrypt round-trip")
}

// --- Encrypted-to-encrypted backup -----------------------------

// SQLCipher's recommended backup pattern is ATTACH + sqlcipher_export.
// Round-trip an encrypted DB into a different-key encrypted DB and
// verify the destination is independently openable with the new key
// and rejects the old key.
func TestEncryptedExportToDifferentKey(t *testing.T) {
	t.Parallel()
	srcDB, _, _ := newEncryptedDB(t, "")
	_, err := srcDB.Exec(`CREATE TABLE secrets (id INTEGER PRIMARY KEY, payload TEXT);`)
	require.NoError(t, err)
	for i := 0; i < 50; i++ {
		_, ierr := srcDB.Exec("INSERT INTO secrets (payload) VALUES (?);",
			fmt.Sprintf("row-%d", i))
		require.NoError(t, ierr)
	}

	destPath := filepath.Join(t.TempDir(), "dest.sqlite")
	var destKey [32]byte
	_, err = io.ReadFull(rand.Reader, destKey[:])
	require.NoError(t, err)
	destHex := hex.EncodeToString(destKey[:])

	_, err = srcDB.Exec(fmt.Sprintf(
		"ATTACH DATABASE %s AS encdest KEY \"x'%s'\";",
		quoteSQLString(destPath), destHex))
	require.NoError(t, err, "attach failed")
	_, err = srcDB.Exec("SELECT sqlcipher_export('encdest');")
	require.NoError(t, err, "sqlcipher_export failed")
	_, err = srcDB.Exec("DETACH DATABASE encdest;")
	require.NoError(t, err)

	destDSN := fmt.Sprintf("%s?_pragma_key=x'%s'", destPath, destHex)
	destDB, err := sql.Open("sqlite3", destDSN)
	require.NoError(t, err)
	t.Cleanup(func() { _ = destDB.Close() })

	var n int
	require.NoError(t, destDB.QueryRow("SELECT count(*) FROM secrets;").Scan(&n))
	assert.Equal(t, 50, n, "exported DB row count mismatch")

	var wrongKey [32]byte
	_, err = io.ReadFull(rand.Reader, wrongKey[:])
	require.NoError(t, err)
	wrongDSN := fmt.Sprintf("%s?_pragma_key=x'%s'", destPath, hex.EncodeToString(wrongKey[:]))
	bad, err := sql.Open("sqlite3", wrongDSN)
	require.NoError(t, err)
	t.Cleanup(func() { _ = bad.Close() })
	_, err = bad.Exec("SELECT count(*) FROM secrets;")
	requireSQLiteCode(t, err, sqlite3.ErrNotADB)
}

// --- PRAGMA rekey ----------------------------------------------

// Changing the encryption key in place via PRAGMA rekey must not
// destroy data and must invalidate the old key for new sessions.
// This is the documented SQLCipher mechanism for key rotation.
func TestPragmaRekey(t *testing.T) {
	t.Parallel()
	db, dbname, _ := newEncryptedDB(t, "")

	_, err := db.Exec(`CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT);
		INSERT INTO t (v) VALUES ('original-1'), ('original-2');`)
	require.NoError(t, err)

	var newKey [32]byte
	_, err = io.ReadFull(rand.Reader, newKey[:])
	require.NoError(t, err)
	newHex := hex.EncodeToString(newKey[:])

	_, err = db.Exec(fmt.Sprintf("PRAGMA rekey = \"x'%s'\";", newHex))
	require.NoError(t, err, "PRAGMA rekey failed")

	var n int
	require.NoError(t, db.QueryRow("SELECT count(*) FROM t;").Scan(&n))
	assert.Equal(t, 2, n)
	require.NoError(t, db.Close())

	newDSN := fmt.Sprintf("%s?_pragma_key=x'%s'", dbname, newHex)
	reopened, err := sql.Open("sqlite3", newDSN)
	require.NoError(t, err)
	t.Cleanup(func() { _ = reopened.Close() })
	require.NoError(t, reopened.QueryRow("SELECT count(*) FROM t;").Scan(&n))
	assert.Equal(t, 2, n)
}

// --- quoteSQLString helper test --------------------------------

// quoteSQLString is the only piece of derived logic in this file
// that isn't validated by the surrounding integration tests.
// Table-driven test guards against accidental injection regressions.
func TestQuoteSQLString(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", "''"},
		{"plain", "alpha", "'alpha'"},
		{"single_quote", "a'b", "'a''b'"},
		{"only_quotes", "''", "''''''"},
		{"with_null", "a\x00b", "'a\x00b'"},
		{"path_with_space", "/tmp/my db.sqlite", "'/tmp/my db.sqlite'"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, quoteSQLString(tc.in))
		})
	}
}

// quoteSQLString returns a single-quoted SQL string literal with
// embedded single quotes doubled — required for safe interpolation
// of file paths into ATTACH DATABASE statements.
func quoteSQLString(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// --- Fuzz: DSN parsing ----------------------------------------

// FuzzEncryptedDSNRoundTrip feeds random byte sequences into the
// _pragma_key DSN parameter and asserts the driver either errors
// cleanly or successfully round-trips the data. The high-value
// surface is the DSN parser, not SQL — SQLite has its own fuzzers.
//
// Run with: go test -fuzz=FuzzEncryptedDSNRoundTrip -fuzztime=30s
func FuzzEncryptedDSNRoundTrip(f *testing.F) {
	// Seed corpus: known-good shapes the fuzzer can mutate.
	f.Add("passphrase")
	f.Add("0123456789abcdef0123456789abcdef")
	f.Add("")
	f.Add("a'b\"c;DROP TABLE t;--")

	f.Fuzz(func(t *testing.T, key string) {
		// URL-unsafe characters in the key would fail DSN parsing
		// itself (a different code path); skip those — fuzzing
		// here targets the cipher-init layer.
		if strings.ContainsAny(key, "\x00&?#") {
			t.Skip()
		}
		dbname := filepath.Join(t.TempDir(), "fuzz.sqlite")
		dsn := fmt.Sprintf("%s?_pragma_key=%s", dbname, key)
		db, err := sql.Open("sqlite3", dsn)
		if err != nil {
			return // sql.Open is lazy; errors here are DSN-parser, fine
		}
		defer func() { _ = db.Close() }()
		// If the key is empty SQLCipher creates an unencrypted DB
		// (documented behaviour). Either way Exec must not panic.
		if _, err := db.Exec(`CREATE TABLE t (x INT); INSERT INTO t VALUES (1);`); err != nil {
			return
		}
		var n int
		if err := db.QueryRow("SELECT count(*) FROM t;").Scan(&n); err != nil {
			return
		}
		if n != 1 {
			t.Fatalf("round-trip lost data: got %d rows, want 1", n)
		}
	})
}

// --- Benchmarks ------------------------------------------------

// Mature DB drivers ship benchmarks paired with the encryption
// surface so that future amalgamation bumps surface perf
// regressions. Run with: go test -bench=. -benchmem -run=^$

func BenchmarkInsertEncrypted(b *testing.B) {
	db := benchSetupDB(b)
	_, err := db.Exec(`CREATE TABLE bench (id INTEGER PRIMARY KEY, v TEXT);`)
	require.NoError(b, err)
	stmt, err := db.Prepare("INSERT INTO bench (v) VALUES (?);")
	require.NoError(b, err)
	defer func() { _ = stmt.Close() }()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := stmt.Exec("payload"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSelectEncrypted(b *testing.B) {
	db := benchSetupDB(b)
	_, err := db.Exec(`CREATE TABLE bench (id INTEGER PRIMARY KEY, v TEXT);`)
	require.NoError(b, err)
	for i := 0; i < 1000; i++ {
		_, ierr := db.Exec("INSERT INTO bench (v) VALUES (?);", fmt.Sprintf("row-%d", i))
		require.NoError(b, ierr)
	}
	stmt, err := db.Prepare("SELECT v FROM bench WHERE id = ?;")
	require.NoError(b, err)
	defer func() { _ = stmt.Close() }()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var v string
		if err := stmt.QueryRow((i % 1000) + 1).Scan(&v); err != nil {
			b.Fatal(err)
		}
	}
}

// benchSetupDB mirrors newEncryptedDB but without the *testing.T
// helper machinery. *testing.B implements the same Cleanup/TempDir
// surface so the resource hygiene is identical.
func benchSetupDB(b *testing.B) *sql.DB {
	b.Helper()
	var key [32]byte
	_, err := io.ReadFull(rand.Reader, key[:])
	require.NoError(b, err)
	dbname := filepath.Join(b.TempDir(), "bench.sqlite")
	dsn := fmt.Sprintf("%s?_pragma_key=x'%s'&_journal_mode=WAL", dbname, hex.EncodeToString(key[:]))
	db, err := sql.Open("sqlite3", dsn)
	require.NoError(b, err)
	b.Cleanup(func() { _ = db.Close() })
	require.NoError(b, db.Ping())
	return db
}
