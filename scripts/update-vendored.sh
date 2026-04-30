#!/usr/bin/env bash
# Refresh the vendored SQLCipher amalgamation against an upstream tag.
#
# Usage:
#   scripts/update-vendored.sh 4.15.0
#
# What it does:
#   1. Clones sqlcipher/sqlcipher at the requested tag into a tempdir.
#   2. Runs the SQLCipher build to produce sqlite3.c + sqlite3.h.
#   3. Replaces the vendored copies in the repository root.
#   4. Leaves CHANGELOG.md and the git index for the maintainer to commit.
#
# What it does NOT do:
#   - Bump libtomcrypt or mattn/go-sqlite3 (separate refresh procedures
#     documented in MAINTAINERS.md).
#   - Run tests or commit. Always run `make test` and review the diff
#     before committing.

set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "Usage: $0 <sqlcipher-version>" >&2
  echo "Example: $0 4.15.0" >&2
  exit 1
fi

VERSION="$1"
TAG="v${VERSION}"
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORKDIR="$(mktemp -d)"
trap 'rm -rf "${WORKDIR}"' EXIT

echo ">> Cloning sqlcipher/sqlcipher at ${TAG} into ${WORKDIR}"
git clone --depth 1 --branch "${TAG}" \
  https://github.com/sqlcipher/sqlcipher.git \
  "${WORKDIR}/sqlcipher"

echo ">> Configuring SQLCipher (OpenSSL crypto, requires libssl-dev + tcl)"
if ! (
  cd "${WORKDIR}/sqlcipher"
  ./configure --enable-tempstore=yes \
    CFLAGS="-DSQLITE_HAS_CODEC -DSQLCIPHER_CRYPTO_OPENSSL" \
    LDFLAGS="-lcrypto" \
    >"${WORKDIR}/configure.log" 2>&1
); then
  echo "ERROR: ./configure failed. configure.log follows:" >&2
  cat "${WORKDIR}/configure.log" >&2
  exit 1
fi

echo ">> Building amalgamation (make sqlite3.c)"
if ! (
  cd "${WORKDIR}/sqlcipher"
  make sqlite3.c >"${WORKDIR}/make.log" 2>&1
); then
  echo "ERROR: make sqlite3.c failed. make.log tail follows:" >&2
  tail -n 80 "${WORKDIR}/make.log" >&2
  exit 1
fi

if [[ ! -f "${WORKDIR}/sqlcipher/sqlite3.c" ]]; then
  echo "ERROR: SQLCipher build did not produce sqlite3.c" >&2
  echo "configure.log tail:" >&2
  tail -n 40 "${WORKDIR}/configure.log" >&2
  echo "make.log tail:" >&2
  tail -n 40 "${WORKDIR}/make.log" >&2
  exit 1
fi

echo ">> Replacing vendored amalgamation in ${REPO_ROOT}"
cp "${WORKDIR}/sqlcipher/sqlite3.c" "${REPO_ROOT}/sqlite3.c"
cp "${WORKDIR}/sqlcipher/sqlite3.h" "${REPO_ROOT}/sqlite3.h"
echo "${VERSION}" >"${REPO_ROOT}/.sqlcipher-version"

echo ">> Done. Next steps:"
echo "    1. cd ${REPO_ROOT}"
echo "    2. make test   # must pass"
echo "    3. Review the diff (git diff --stat sqlite3.c sqlite3.h)"
echo "    4. Update CHANGELOG.md with the new version"
echo "    5. git add sqlite3.c sqlite3.h .sqlcipher-version CHANGELOG.md"
echo "    6. git commit -s -m \"chore(vendor): bump SQLCipher to ${VERSION}\""
