# Release process

This project ships a tagged release whenever there is enough accumulated
work to be worth a version bump. The default cadence is roughly
**quarterly** (matching SQLCipher upstream), with patch releases in
between for security or correctness fixes.

## Release flow

```sh
./scripts/release.sh v4.15.1
```

The script enforces every safety check that a manual `git tag` could
forget:

1. The version argument matches `vX.Y.Z[-pre]`.
2. You are on `main` with a clean working tree.
3. The tag does not already exist locally or on `origin`.
4. `CHANGELOG.md` has an entry for the new version (or a non-empty
   `[Unreleased]` section to promote).
5. The latest CI run on `main` is green.

After the dry-run summary, you confirm `y` and the script:

- Creates a signed annotated tag (`git tag -s`) on the current commit.
- Pushes the tag to `origin`.

The `sigstore-sign.yml` workflow takes over and produces:

- Source tarball (`go-sqlcipher-vX.Y.Z.tar.gz`)
- SBOM (`go-sqlcipher.spdx.json`)
- cosign signature (`go-sqlcipher-vX.Y.Z.sigstore.json`)
- SLSA provenance (`*.intoto.jsonl` via slsa-github-generator)

All four are uploaded to the GitHub Release page.

## Choosing the version bump

SemVer 2.0:

| Change kind | Bump |
|---|---|
| Bug fix, no public API change (Tier 1/2 lint cleanup, security patch) | PATCH (`v4.15.0` → `v4.15.1`) |
| New public API, backwards-compatible (e.g. new helper func) | MINOR (`v4.15.0` → `v4.16.0`) |
| Breaking public API change (rare for a fork) | MAJOR (`v4.x.y` → `v5.0.0`) |
| SQLCipher upstream bump (e.g. 4.15 → 4.16) | MINOR by default |

The `Conventional Commits` types in commit subjects map roughly to:

- `fix:` → PATCH
- `feat:` → MINOR
- `feat!:` or `BREAKING CHANGE:` footer → MAJOR
- `chore:` / `docs:` / `test:` / `refactor:` → no bump on their own

## CHANGELOG editing

`CHANGELOG.md` follows [Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/).
The `[Unreleased]` section accumulates entries between releases. When
tagging, manually rename `## [Unreleased]` to `## [X.Y.Z] — YYYY-MM-DD`
and start a new empty `## [Unreleased]` block above it.

The release script does **not** modify `CHANGELOG.md` — keeping that
edit explicit makes the audit trail clear.

## After tagging

1. The Release page appears within ~5 minutes at
   `https://github.com/WissCore/go-sqlcipher/releases/tag/vX.Y.Z`.
2. pkg.go.dev usually picks up the new version within ~15-30 minutes.
   Force the fetch with:

   ```sh
   GOPROXY=https://proxy.golang.org go list -m -versions github.com/WissCore/go-sqlcipher/v4
   ```

3. Update downstream `go.mod` files that pin to a specific version.

## Branch protection

`main` is protected:

- Requires the single composite check `ci-success` (which depends on the
  full matrix of build-test jobs across Linux x86_64, Linux ARM64, macOS,
  and Windows; plus security scans, DCO, etc.).
- Requires PRs (no direct pushes).
- Requires conventional-commits PR titles.
- Requires DCO sign-off on every commit.

Tags are only pushed by maintainers via the release script; we do not
auto-tag on merge.
