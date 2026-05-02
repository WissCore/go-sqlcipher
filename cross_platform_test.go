// Cross-platform sanity tests. These run on every OS in the CI matrix
// and exercise real platform-native code paths (path separators,
// filename encoding, file-system locking semantics) that the inherited
// tests cover only implicitly through `t.TempDir`.
//
// Each test is intentionally fast (sub-second) so the full matrix
// stays under five minutes on the slowest runner.

package sqlite3_test

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	_ "github.com/WissCore/go-sqlcipher/v4"
	"github.com/stretchr/testify/require"
)

// openEncryptedAt is a small helper that opens an encrypted DB at the
// given absolute path with a random 256-bit key, closes it, then
// reopens it and verifies a read-back. Used by the platform tests
// below to confirm that whatever path the OS hands us survives a full
// open/close/reopen round-trip.
func openEncryptedAt(t *testing.T, dbPath string) {
	t.Helper()
	var key [32]byte
	_, err := io.ReadFull(rand.Reader, key[:])
	require.NoError(t, err)
	dsn := fmt.Sprintf("%s?_pragma_key=x'%s'", dbPath, hex.EncodeToString(key[:]))

	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	require.NoError(t, db.Ping())
	_, err = db.Exec(`CREATE TABLE t(x INTEGER); INSERT INTO t VALUES (42);`)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	db2, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	defer db2.Close()
	var got int
	require.NoError(t, db2.QueryRow("SELECT x FROM t LIMIT 1;").Scan(&got))
	require.Equal(t, 42, got)
}

// TestNativePathFilenameVariants opens an encrypted DB at filenames
// that exercise OS-native path handling: spaces, unicode, multiple
// dots, very long names. Different filesystems (NTFS, APFS, ext4)
// historically differ on what they accept; this test fails loudly if
// our DSN parser, the cgo bridge, or SQLite's file layer mishandle a
// path on the runner's OS.
func TestNativePathFilenameVariants(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		filename string
	}{
		{"simple", "simple.db"},
		{"with_spaces", "name with spaces.db"},
		{"with_dashes", "name-with-dashes.db"},
		{"many_dots", "many.dots.in.the.name.db"},
		{"unicode", "юникод-名前.db"},
		{"very_long", "a_very_long_filename_that_pushes_path_length_to_test_filesystem_limits_on_each_os_runner.db"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			openEncryptedAt(t, filepath.Join(t.TempDir(), tc.filename))
		})
	}
}

// TestPlatformInfo emits the OS/arch/Go combo into the test log so
// CI artifacts and failure reports show exactly which runner exposed
// any subsequent failure. No assertions — pure breadcrumb.
func TestPlatformInfo(t *testing.T) {
	t.Logf("runtime: GOOS=%s GOARCH=%s NumCPU=%d", runtime.GOOS, runtime.GOARCH, runtime.NumCPU())
}

// TestConcurrentOpenSameDB verifies that two database handles opened
// against the same encrypted file from the same process can both read
// without stepping on each other. SQLite's per-platform file-locking
// implementation differs (fcntl on Linux, BSD locks on macOS, LockFile
// on Windows); this test exercises the platform's actual locking path
// rather than mocking it.
func TestConcurrentOpenSameDB(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(t.TempDir(), "shared.sqlite")
	var key [32]byte
	_, err := io.ReadFull(rand.Reader, key[:])
	require.NoError(t, err)
	dsn := fmt.Sprintf("%s?_pragma_key=x'%s'&_journal_mode=WAL", dbPath, hex.EncodeToString(key[:]))

	// Seed.
	seed, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	_, err = seed.Exec(`CREATE TABLE t(id INTEGER PRIMARY KEY); INSERT INTO t (id) VALUES (1);`)
	require.NoError(t, err)
	require.NoError(t, seed.Close())

	// Two independent reader handles, each in its own goroutine, both
	// hitting the same on-disk DB through the platform's lock manager.
	const readers = 4
	var wg sync.WaitGroup
	wg.Add(readers)
	errCh := make(chan error, readers)
	for i := 0; i < readers; i++ {
		go func() {
			defer wg.Done()
			db, oerr := sql.Open("sqlite3", dsn)
			if oerr != nil {
				errCh <- fmt.Errorf("open: %w", oerr)
				return
			}
			defer db.Close()
			var got int
			if serr := db.QueryRow("SELECT id FROM t LIMIT 1;").Scan(&got); serr != nil {
				errCh <- fmt.Errorf("scan: %w", serr)
				return
			}
			if got != 1 {
				errCh <- fmt.Errorf("got %d, want 1", got)
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for e := range errCh {
		t.Error(e)
	}
}

// TestEnvHandlesAbsoluteAndRelativePath proves that both an absolute
// path (the t.TempDir default) and a relative path resolved from the
// process CWD work. Some Windows code paths historically misparsed
// drive-letter prefixes; this is the regression guard.
func TestEnvHandlesAbsoluteAndRelativePath(t *testing.T) {
	t.Parallel()
	tmpdir := t.TempDir()

	// Absolute first.
	openEncryptedAt(t, filepath.Join(tmpdir, "absolute.sqlite"))

	// Relative — chdir into tmpdir, open by basename.
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpdir))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	openEncryptedAt(t, "relative.sqlite")
}
