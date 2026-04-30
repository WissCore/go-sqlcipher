# Contributors

This file credits everyone whose work appears in this repository, in
chronological order of contribution.

## Original `mutecomm/go-sqlcipher` (2017–2020)

The Go driver itself, and the integration of SQLCipher with `mattn/go-sqlite3`,
was created and maintained at [`mutecomm/go-sqlcipher`](https://github.com/mutecomm/go-sqlcipher)
by:

- **Frank Braun** ([`@frankbraun`](https://github.com/frankbraun)) — primary
  author, maintained the project across 61 commits from 2017 to 2020
- **Andreas Linz** ([`@klingtnet`](https://github.com/klingtnet)) — contributor
- **nkbai** ([`@nkbai`](https://github.com/nkbai)) — contributor
- **Jonathan Logan** ([`@JonathanLogan`](https://github.com/JonathanLogan)) — co-maintainer

We thank them for the foundation. The architecture, the test suite, the
upstream-tracking scripts (`track_go-sqlite3.sh`, `track_libtomcrypt.sh`),
and the pragma-key DSN convention all come from their work and are kept
faithfully here.

## Upstream sources

The vendored C and Go source files are not original to either fork; they
are taken (with attribution and respective licenses) from:

- **SQLite** — public domain, by [Hwaci](https://www.sqlite.org/crew.html);
  shipped as part of the SQLCipher amalgamation.
- **SQLCipher** ([`sqlcipher/sqlcipher`](https://github.com/sqlcipher/sqlcipher))
  — BSD-3-Clause, by Zetetic LLC. The page-encryption codec, key derivation,
  and PRAGMA surface are theirs.
- **libtomcrypt** ([`libtom/libtomcrypt`](https://github.com/libtom/libtomcrypt))
  — public domain. The AES-256, SHA-256/512, HMAC, and PKCS#5 implementations
  used by SQLCipher come from this library.
- **`mattn/go-sqlite3`** ([`mattn/go-sqlite3`](https://github.com/mattn/go-sqlite3))
  — MIT, by Yasuhiro Matsumoto and contributors. The Go-side `database/sql`
  driver shim that this package wraps.

Without any one of these projects, this fork would not exist. Thank you.

## This fork (2026-present)

Maintained by:

- **WissCore** team — [`github.com/WissCore`](https://github.com/WissCore)

If you contribute, your name belongs here. Add it in the same PR as your
first commit.
