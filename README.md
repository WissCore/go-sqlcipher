# go-sqlcipher

> Self-contained Go driver for [SQLCipher](https://www.zetetic.net/sqlcipher/) — encrypted SQLite, audited, easy.

<p align="left">
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-BSD--3--Clause-blue" alt="License"></a>
  <a href="https://pkg.go.dev/github.com/WissCore/go-sqlcipher/v4"><img src="https://pkg.go.dev/badge/github.com/WissCore/go-sqlcipher/v4.svg" alt="Go Reference"></a>
  <a href="https://scorecard.dev/viewer/?uri=github.com/WissCore/go-sqlcipher"><img src="https://api.scorecard.dev/projects/github.com/WissCore/go-sqlcipher/badge" alt="OpenSSF Scorecard"></a>
  <a href="SECURITY.md"><img src="https://img.shields.io/badge/security-policy-orange" alt="Security policy"></a>
</p>

Maintained by [@alanwiss](https://github.com/alanwiss), who picked up
maintenance of this driver in 2026 after the upstream had been dormant
since 2020. See [`CONTRIBUTORS.md`](CONTRIBUTORS.md) for the full chain
of credits going back to the original authors.

A Go driver for SQLite that keeps every database file encrypted at rest with
SQLCipher's audited AES-256 + HMAC-SHA-512 page format. Implements the standard
`database/sql` interface, so any code that already speaks `database/sql` works
unchanged — open the connection with a key, every read and write is encrypted
on the way to disk and decrypted on the way back.

## Status

This package was originally written and maintained by Frank Braun and
contributors at [`mutecomm/go-sqlcipher`](https://github.com/mutecomm/go-sqlcipher)
between 2017 and 2020. The upstream repository has not received commits since
December 2020 and was flagged abandoned by automated dependency scanners; we
forked it to resume maintenance, follow the SQLCipher upstream release cadence,
and keep the supply-chain hygiene current.

We are deeply grateful for the original authors' work. Their architecture,
test suite, and tracking scripts (`track_go-sqlite3.sh`, `track_libtomcrypt.sh`)
are the backbone of this fork; we are following the structure they laid down,
just keeping the vendored sources fresh. See [`CONTRIBUTORS.md`](CONTRIBUTORS.md)
for full credits.

## What's different from upstream

- Vendored SQLCipher amalgamation tracks the latest [`sqlcipher/sqlcipher`](https://github.com/sqlcipher/sqlcipher)
  release (currently v4.15.0). Quarterly bumps via a scripted `Makefile` target.
- Vendored SQLite amalgamation follows the SQLCipher upstream snapshot
  (currently 3.53.0).
- Vendored libtomcrypt sources updated in lockstep.
- Reproducible CI: `golangci-lint`, `gosec`, `govulncheck`, `osv-scanner`,
  `gitleaks`, `trufflehog`, `addlicense`, `actionlint`, `zizmor` on every PR.
- Signed releases: cosign keyless signature, SBOM (SPDX), SLSA build provenance.
- Pinned tool versions via [`mise`](https://mise.jdx.dev) so local pre-commit
  hooks and CI run the same scanners with the same arguments.

We did not change the public Go API. Code that imports
`github.com/mutecomm/go-sqlcipher/v4` switches with a single line in `go.mod`:

```go
require github.com/WissCore/go-sqlcipher/v4 v4.15.0
```

```sh
go mod edit -replace github.com/mutecomm/go-sqlcipher/v4=github.com/WissCore/go-sqlcipher/v4@v4.15.0
go mod tidy
```

## Why a Go SQLCipher driver matters

SQLCipher is the most widely audited open-source approach to file-level
SQLite encryption — used in Mozilla, Microsoft, Adobe, and countless
security-sensitive applications. It encrypts every database page with AES-256
in CBC mode and authenticates each page with HMAC-SHA-512, so a stolen disk
image, a leaked backup, or a mistyped `aws s3 cp` reveals only ciphertext.

Go applications that need that property had to depend on a fork that fell
behind the SQLCipher release cadence years ago — bringing along old SQLite
versions and unpatched CVEs. This fork brings the Go ecosystem back in step
with what SQLCipher upstream actually ships today.

## Install

```sh
go get github.com/WissCore/go-sqlcipher/v4
```

## Use

```go
import (
    "database/sql"
    "fmt"

    _ "github.com/WissCore/go-sqlcipher/v4"
)

func main() {
    key := "2DD29CA851E7B56E4697B0E1F08507293D761A05CE4D1B628663F411A8086D99"
    dsn := fmt.Sprintf("db?_pragma_key=x'%s'&_pragma_cipher_page_size=4096", key)
    db, err := sql.Open("sqlite3", dsn)
    if err != nil {
        panic(err)
    }
    defer db.Close()
    // ...regular database/sql calls from here on; everything is encrypted.
}
```

`_pragma_key` accepts a hex blob (64 characters for a 32-byte key) or a URL-
escaped passphrase that goes through SQLCipher's PBKDF2 key derivation. See
the [SQLCipher API documentation](https://www.zetetic.net/sqlcipher/sqlcipher-api/)
for the full set of PRAGMAs.

`sqlite3.IsEncrypted(path)` reports whether a given file already has a
SQLCipher header.

Working examples live under [`_example/`](_example/).

## Building

The package is self-contained — the only external requirement is a C
compiler (typically `gcc` or `clang`) and OpenSSL development headers for
linking `libcrypto`. The vendored sources compile via the standard `cgo`
build pipeline:

```sh
go build ./...
go test -race ./...
```

For exotic platforms (musl, static linking, Apple Silicon cross-compile)
see [`docs/building.md`](docs/building.md).

## Updating SQLCipher

A `Makefile` target wraps the amalgamation refresh:

```sh
make update-sqlcipher VERSION=4.15.0
make test
```

This downloads the SQLCipher source archive from
[`sqlcipher/sqlcipher`](https://github.com/sqlcipher/sqlcipher), runs their
build to produce the standalone `sqlite3.c` / `sqlite3.h` files, and replaces
the vendored copies. The full procedure, including how to verify upstream
signatures, is documented in [`MAINTAINERS.md`](MAINTAINERS.md).

## Security

We treat this as security-critical infrastructure for our own messenger
project and run it with the same supply-chain rigour we apply there:

- Every release is signed with cosign and ships an SBOM and SLSA build
  provenance attestation
- Pinned tool versions and SHA-pinned GitHub Actions
- Weekly OSV-scanner and CodeQL re-runs on `main`
- Vulnerability disclosure: see [`SECURITY.md`](SECURITY.md)

If you find a vulnerability, please report it privately per the security
policy, not through a public issue.

## Compatibility

- Go ≥ 1.21 (we follow [Go's official release support window](https://go.dev/doc/devel/release#policy))
- SQLCipher 4.x format (databases created by SQLCipher 3.x require
  [`PRAGMA cipher_migrate`](https://www.zetetic.net/sqlcipher/sqlcipher-api/#cipher_migrate)
  before they will open under this driver)
- Linux, macOS, FreeBSD; Windows builds work but are not part of CI
  matrix yet — contributions welcome

## Contributing

We are a small team primarily focused on our own use case (an end-to-end
encrypted messenger). That said, we benefit directly from the community
that built this driver before us, so we want to keep this fork open and
useful to others:

- Issues are triaged on a best-effort basis; security reports take priority
- Pull requests are welcome — please read [`CONTRIBUTING.md`](CONTRIBUTING.md)
- Conventional Commits 1.0 + DCO sign-off + GPG-signed commits are required
  by the pre-commit hooks (`lefthook` config in repo)

If you would like to step up as a co-maintainer, please open a discussion.

## Lineage

```text
sqlite.org/sqlite           — public-domain SQLite (Hwaci)
  └── sqlcipher/sqlcipher   — SQLite + AES-256 page encryption (Zetetic, BSD-3)
        └── mattn/go-sqlite3 — Go cgo binding to SQLite (Yasuhiro Matsumoto, MIT)
              └── mutecomm/go-sqlcipher — first SQLCipher Go binding (Frank Braun, BSD-3) ← origin of this fork
                    └── WissCore/go-sqlcipher — this fork, resumed maintenance from 2026-04
```

See [`NOTICE`](NOTICE) for the full attribution chain and
[`LICENSE`](LICENSE) for the legal notices.

## License

BSD-3-Clause. The vendored SQLCipher, SQLite, libtomcrypt, and mattn/go-sqlite3
sources keep their original licenses (BSD-3-Clause-with-patent, public
domain, public domain, and MIT respectively); see [`LICENSE`](LICENSE) for
the combined notice.
