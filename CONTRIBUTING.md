# Contributing to go-sqlcipher

Thank you for considering a contribution. This is a security-sensitive
package — every database opened with this driver is guarded by it. The bar
for changes is intentionally high.

## What we accept

- **Bug fixes** with a clear reproduction.
- **Documentation improvements** that make the project easier to audit,
  build, or use.
- **Test coverage** for existing functionality.
- **Vendoring refresh** that pulls newer SQLCipher / SQLite / libtomcrypt
  releases without changing the Go API.
- **Build improvements** for new platforms (Windows, BSD variants,
  static linking, musl, Apple Silicon).

## What needs prior discussion

Open an issue or discussion **before** starting work on:

- Changes to the encryption layer or PRAGMA handling.
- Changes to the public Go API.
- New external dependencies, especially crypto libraries.
- Changes to vendoring strategy.

For substantial changes, open a draft PR with the design rationale before
writing the code.

## Developer Certificate of Origin (DCO)

Every commit **must** carry a `Signed-off-by:` trailer that certifies the
[Developer Certificate of Origin](https://developercertificate.org/):

```sh
git commit -s -m "fix: handle non-UTF8 pragma values"
```

This appends:

```text
Signed-off-by: Your Name <your.email@example.com>
```

PRs without DCO sign-off on every commit will be blocked by CI.

We do **not** require a CLA.

## Signed commits

Commits to `main` should be GPG-signed. CI will not block unsigned
commits in PRs (we know this raises the contribution bar) but we
strongly encourage signing — see GitHub's
[guide on signing commits](https://docs.github.com/en/authentication/managing-commit-signature-verification/signing-commits).

## Workflow

1. Fork the repository.
2. Create a topic branch from `main`: `feat/short-kebab`, `fix/issue-123`,
   `docs/...`, `chore/...`.
3. Make focused commits using
   [Conventional Commits 1.0](https://www.conventionalcommits.org/en/v1.0.0/)
   (`feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `build:`, `ci:`,
   `chore:`, `revert:`, optional scope `fix(codec): ...`, breaking
   with `feat!:` or footer).
4. Push and open a pull request against `main`.
5. CI must pass: build, tests, secret scan, dependency review, code
   scanning, DCO check.
6. Resolve review comments. We squash-merge by default; the PR title
   becomes the squash commit subject.

## Code style

Tooling is pinned via [`mise`](https://mise.jdx.dev) — run `mise install`
once. Pre-commit hooks are managed by
[`lefthook`](https://github.com/evilmartians/lefthook) — run `lefthook install`
once.

| Tool | Purpose |
|---|---|
| `gofumpt` | Go formatting |
| `golangci-lint` | Go static analysis (24 linters) |
| `gosec` | Go security audit |
| `govulncheck` | Known-CVE check against `go.mod` |
| `markdownlint` | Markdown style |
| `typos` | Typo check across the tree |
| `gitleaks`, `trufflehog` | Secret scan |
| `actionlint`, `zizmor` | GitHub Actions lint |

## Tests

```sh
go test -race -count=1 ./...
```

The vendored SQLCipher test vectors live under `testdata/`; do not modify
them by hand. If you need to refresh the corpus, do so as part of an
amalgamation refresh PR.

## Vendoring refresh

If your contribution refreshes the vendored upstream sources, please
follow [`MAINTAINERS.md`](MAINTAINERS.md). The refresh must be a
**single commit** that touches only the vendored files plus
`CHANGELOG.md`, with the upstream version in the commit subject:

```text
chore(vendor): bump SQLCipher to 4.16.0
```

## Reporting security issues

**Do not** open a public issue or PR for security problems. Follow
[`SECURITY.md`](SECURITY.md) — coordinated disclosure protects every
project that depends on this driver.

## Code of conduct

By participating, you agree to abide by the [Code of Conduct](CODE_OF_CONDUCT.md).

## Licence

By contributing, you agree that your contributions are licensed under
BSD-3-Clause, the same licence as the rest of this fork (and the
upstream `mutecomm/go-sqlcipher`).
