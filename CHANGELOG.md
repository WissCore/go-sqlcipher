# Changelog

## [Unreleased]

### Changed

- Bump vendored SQLCipher to 4.15.0 (was 4.4.2). Generated automatically by .github/workflows/upstream-bump.yml on 2026-05-01.


All notable changes to this fork are documented here. The format follows
[Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/) and the
versioning is [SemVer 2.0](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Forked from `mutecomm/go-sqlcipher@25f68ad` (last upstream commit,
  2020-12-07). Full git history preserved.
- `CONTRIBUTORS.md` — credits to the original authors and upstream
  projects.
- `NOTICE` — formal attribution chain.
- This `CHANGELOG.md`.

### Planned for v4.15.0 (first WissCore release)

- Bump vendored SQLCipher amalgamation from 4.4.2 to 4.15.0.
- Bump vendored SQLite amalgamation from 3.34.x to 3.53.0
  (carried by SQLCipher 4.15.0).
- Bump vendored libtomcrypt to current `develop` snapshot.
- Replace upstream CI workflow with the WissCore CI orchestrator
  (golangci-lint, gosec, govulncheck, gitleaks, codeql, osv-scanner,
  zizmor, smoke matrix, signed releases).
- Add `Makefile` target `update-sqlcipher VERSION=...` to script the
  quarterly amalgamation refresh.
- Add `docs/building.md` for the cgo + OpenSSL build matrix.

## Pre-fork history

For commits prior to 2026-04, see `git log` and the original repository
at <https://github.com/mutecomm/go-sqlcipher>.
