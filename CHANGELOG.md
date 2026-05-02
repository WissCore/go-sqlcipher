# Changelog

All notable changes to this fork are documented here. The format follows
[Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/) and the
versioning is [SemVer 2.0](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [4.15.1] — 2026-05-02

Maintenance release. No public API changes; behaviour of code using
the `database/sql` interface is unchanged.

### Verified on

- Linux x86_64, Linux ARM64, macOS Apple Silicon, Windows x86_64
- Go `oldstable` and `stable`

(Previous releases were CI-tested only on Linux x86_64.)

### Fixed

- Tighter error handling inside the package's own test helpers:
  `rows.Err()` is now checked after every iteration, `*sql.Rows`
  and `*sql.Stmt` resources are deferred-closed, and `DROP TABLE`
  test fixtures use `IF EXISTS` so a real DB fault is no longer
  swallowed as a spurious "no such table" error.

## [4.15.0] — 2026-04-30

First WissCore release.

### Added

- Forked from `mutecomm/go-sqlcipher@25f68ad` (last upstream commit,
  2020-12-07). Full git history preserved.
- Vendored SQLCipher amalgamation upgraded from 4.4.2 to 4.15.0
  (carries SQLite 3.53.0; libtomcrypt refreshed in lockstep).
- `CONTRIBUTORS.md` — credits to the original authors and upstream projects.
- `NOTICE` — formal attribution chain.
- WissCore CI orchestrator (`golangci-lint`, `gosec`, `govulncheck`,
  `gitleaks`, `codeql`, `osv-scanner`, `zizmor`) and signed releases
  (cosign keyless, SBOM, SLSA build provenance).

## Pre-fork history

For commits prior to 2026-04, see `git log` and the original repository
at <https://github.com/mutecomm/go-sqlcipher>.
