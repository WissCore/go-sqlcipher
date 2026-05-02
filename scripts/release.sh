#!/usr/bin/env bash
# Release helper for WissCore/go-sqlcipher.
#
# Manual release flow with safety guards. Run from repo root with the
# new version tag as the only argument:
#
#   ./scripts/release.sh v4.15.1
#
# What it does:
#   1. Verifies argument is a valid semver-shaped tag (vX.Y.Z[-pre]).
#   2. Verifies branch is `main` and clean.
#   3. Verifies tag does not already exist (locally or on origin).
#   4. Verifies CHANGELOG.md has an entry for the new version (or that
#      [Unreleased] has content to promote).
#   5. Verifies CI on main is green for the current HEAD.
#   6. Prints a dry-run summary and asks for explicit yes/no.
#   7. On confirmation: creates a signed annotated tag and pushes it.
#
# After push, the sigstore-sign workflow takes over: it produces a
# source tarball, SBOM, cosign signature, and SLSA provenance, and
# uploads them to the GitHub Release page.

set -euo pipefail

VERSION="${1:-}"
REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "${REPO_ROOT}"

# 1. Argument shape
if [[ -z "${VERSION}" ]]; then
  cat <<EOF >&2
Usage: $0 vX.Y.Z[-pre]

Example: $0 v4.15.1
EOF
  exit 1
fi
if ! [[ "${VERSION}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]]; then
  echo "ERROR: version must match vX.Y.Z[-pre] (got: ${VERSION})" >&2
  exit 1
fi

# 2. Branch + working tree
BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [[ "${BRANCH}" != "main" ]]; then
  echo "ERROR: must release from main (current: ${BRANCH})" >&2
  exit 1
fi
if ! git diff --quiet || ! git diff --cached --quiet; then
  echo "ERROR: working tree has uncommitted changes" >&2
  git status --short >&2
  exit 1
fi

# 3. Tag uniqueness
if git rev-parse "refs/tags/${VERSION}" >/dev/null 2>&1; then
  echo "ERROR: tag ${VERSION} already exists locally" >&2
  exit 1
fi
if git ls-remote --exit-code --tags origin "refs/tags/${VERSION}" >/dev/null 2>&1; then
  echo "ERROR: tag ${VERSION} already exists on origin" >&2
  exit 1
fi

# 4. CHANGELOG
if ! grep -qE "^## \[?${VERSION#v}\]?" CHANGELOG.md; then
  if ! grep -qE '^## \[Unreleased\]' CHANGELOG.md; then
    echo "ERROR: CHANGELOG.md has neither a ${VERSION} section nor an [Unreleased] section" >&2
    exit 1
  fi
  # Check that [Unreleased] is not empty (has at least one bullet)
  awk '/^## \[Unreleased\]/{found=1; next} found && /^## /{exit} found && /^- /{print; exit}' CHANGELOG.md |
    grep -q . || {
    echo "ERROR: CHANGELOG.md [Unreleased] section is empty — nothing to release" >&2
    exit 1
  }
  echo "INFO: CHANGELOG.md will keep [Unreleased] heading; you should manually rename it to [${VERSION#v}] before tagging."
  echo "      (This script does NOT modify CHANGELOG.md to keep the audit trail explicit.)"
fi

# 5. CI status (best effort: gh required)
if command -v gh >/dev/null 2>&1; then
  HEAD_SHA=$(git rev-parse HEAD)
  STATUS=$(gh run list --branch main --commit "${HEAD_SHA}" --limit 1 --json conclusion --jq '.[0].conclusion // "missing"' 2>/dev/null || echo "missing")
  case "${STATUS}" in
    success) echo "INFO: latest CI on main HEAD is green" ;;
    missing) echo "WARN: no CI run found for main HEAD ${HEAD_SHA:0:7}" ;;
    *)
      echo "ERROR: latest CI on main HEAD is '${STATUS}', not 'success'" >&2
      exit 1
      ;;
  esac
else
  echo "WARN: gh CLI not installed — skipping CI status check"
fi

# 6. Dry-run summary + confirmation
COMMIT=$(git rev-parse --short HEAD)
SUBJECT=$(git log -1 --pretty=%s)
cat <<EOF

=== Release dry-run ===
  Version:  ${VERSION}
  Branch:   ${BRANCH}
  Commit:   ${COMMIT}
  Subject:  ${SUBJECT}

The script will:
  - Create a signed annotated tag '${VERSION}' on this commit
  - Push the tag to origin
  - This triggers .github/workflows/sigstore-sign.yml which builds the
    source tarball, SBOM, cosign signature, and SLSA provenance, and
    uploads them to the GitHub Release page.

EOF

read -rp "Proceed with release ${VERSION}? (y/N) " confirm
if [[ "${confirm}" != "y" && "${confirm}" != "Y" ]]; then
  echo "Aborted."
  exit 1
fi

# 7. Tag + push
git tag -s "${VERSION}" -m "Release ${VERSION}"
git push origin "${VERSION}"

cat <<EOF

=== Release ${VERSION} pushed ===

Watch the sigstore-sign workflow:
  https://github.com/WissCore/go-sqlcipher/actions/workflows/sigstore-sign.yml

After it completes (~3-5 minutes), the Release page will be at:
  https://github.com/WissCore/go-sqlcipher/releases/tag/${VERSION}

Notify pkg.go.dev that the new version exists:
  GOPROXY=https://proxy.golang.org go list -m -versions github.com/WissCore/go-sqlcipher/v4

EOF
