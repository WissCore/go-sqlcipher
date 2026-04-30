# Maintainers and maintenance procedure

This file documents the procedure for keeping the vendored upstream
sources fresh. It is the operator-facing reference for the routine
quarterly bumps and the occasional out-of-band security refresh.

## Current maintainers

| Maintainer | GitHub | Scope |
|---|---|---|
| WissCore | [`@WissCore`](https://github.com/WissCore) | Everything |

If you would like to step up as a co-maintainer, please open a
discussion. We particularly welcome reviewers with cgo / sqlite-internals
experience.

## Upstream sources to track

The repository pulls C source from three upstream projects. Each has its
own release cadence; the script `scripts/update-vendored.sh` (TODO in
v4.15.0) automates the safe parts of the refresh.

### 1. SQLCipher (page-encryption codec + SQLite amalgamation)

- Source: <https://github.com/sqlcipher/sqlcipher>
- Release cadence: ≈ monthly
- License: BSD-3-Clause
- Vendored files: `sqlite3.c`, `sqlite3.h`

The SQLCipher build produces a single-file amalgamation that bundles
both the SQLite source and the SQLCipher patch on top. Building is the
canonical way to refresh both at once:

```sh
git clone https://github.com/sqlcipher/sqlcipher.git
cd sqlcipher
git checkout v4.15.0     # or whichever release we are pinning
./configure --enable-tempstore=yes \
            CFLAGS="-DSQLITE_HAS_CODEC -DSQLCIPHER_CRYPTO_LIBTOMCRYPT"
make sqlite3.c
cp sqlite3.c sqlite3.h /path/to/this/repo/
```

Verify the upstream tag is signed (or that the tarball checksum matches
the GitHub release artefact) before vendoring.

### 2. mattn/go-sqlite3 (Go-side database/sql binding)

- Source: <https://github.com/mattn/go-sqlite3>
- Release cadence: ≈ quarterly
- License: MIT
- Vendored Go files: `sqlite3.go`, `sqlite3_*.go`, `callback.go`,
  `convert.go`, `error.go`, `backup.go`, et al.

The original `mutecomm/go-sqlcipher` includes a script
`track_go-sqlite3.sh` that diffs our vendored Go files against the
specified upstream tag. Refresh procedure:

```sh
./track_go-sqlite3.sh v1.14.30        # or current latest
# review the diff carefully — some deltas are SQLCipher-specific patches
# we deliberately keep, not bugs upstream fixed
```

Manual review is required: not every upstream change applies cleanly to
our SQLCipher integration, and some changes touch the SQLITE_HAS_CODEC
guards which we own.

### 3. libtomcrypt (AES + SHA + HMAC primitives)

- Source: <https://github.com/libtom/libtomcrypt>
- Release cadence: irregular (the project ships from `develop`)
- License: public domain
- Vendored files: `aes.c`, `aes_tab.h`, `cbc_*.c`, `crypt_*.c`,
  `fortuna.c`, `hash_memory.c`, `hmac_*.c`, `pkcs_5_2.c`, `sha*.c`,
  `tomcrypt*.h`, `zeromem.c`

The original `mutecomm/go-sqlcipher` includes a script
`track_libtomcrypt.sh` that pulls a specific commit and copies the
files we depend on. A new commit can typically be vendored every six
months — bumping more often makes diff review heavier than the gain.

## Refresh checklist (per release)

1. Identify the target SQLCipher version (latest stable on
   <https://github.com/sqlcipher/sqlcipher/releases>).
2. Run `make update-sqlcipher VERSION=...` (script lives at
   `scripts/update-vendored.sh`).
3. Run `go test -race -count=1 ./...` — must be green.
4. Run `golangci-lint run ./...` — must be green.
5. Run `gosec ./...` and `govulncheck ./...` — must be green.
6. Open a single PR with subject `chore(vendor): bump SQLCipher to X.Y.Z`,
   touching only the vendored files plus `CHANGELOG.md`.
7. After merge, tag a release `vX.Y.Z` matching the SQLCipher version.
8. Release workflow signs the artefacts (cosign), generates SBOM and
   SLSA provenance, attaches them to the GitHub Release.

## Emergency security refresh

If SQLCipher publishes a security advisory:

1. Acknowledge the advisory in our repository within 24 hours
   (open a tracking issue, do not include exploit details).
2. Refresh as above, but treat the PR as security: minimal scope, fast
   review, no other changes bundled in.
3. Tag a patch release.
4. Publish a GitHub Security Advisory linking to the upstream advisory
   and our patch release.

## Security contact

See [`SECURITY.md`](SECURITY.md). Vulnerabilities found in the vendored
upstream code should also be reported to the respective upstream
project — we will help route the report.
