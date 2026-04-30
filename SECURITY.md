# Security policy

## Reporting a vulnerability

If you believe you have found a security vulnerability in this driver,
**please do not open a public GitHub issue**. Cryptographic libraries
need responsible disclosure: the moment a flaw is public, every project
using the library is at risk.

**Preferred channel:** [GitHub Private Vulnerability Reporting](https://github.com/WissCore/go-sqlcipher/security/advisories/new).

**Backup channel:** email `alan@wisscore.com` — plain text is fine,
describe the issue as you would in an issue.

We commit to:

- **Acknowledge** your report within 72 hours.
- **Initial triage** within 7 days (severity, reproduction confirmation).
- **Coordinated disclosure** in agreement with you. Default embargo is 90
  days from initial report, extendable when a fix is non-trivial.
- **Credit you** in the advisory (your handle, your real name, or
  anonymous — your choice).

## Where to report which kind of issue

This package is a thin Go wrapper around three vendored upstream codebases.
Reports may need to flow upstream:

| Class of issue | Report to |
|---|---|
| Bug in the page-level encryption codec, key derivation, PRAGMA handling | `sqlcipher/sqlcipher` upstream first, copy us in |
| Bug in SQLite query planner, parser, file format | `sqlite.org` (private contact form) |
| Bug in AES / SHA / HMAC primitives | `libtom/libtomcrypt` |
| Bug in the Go-side `database/sql` driver wrapper | report here |
| Build / supply-chain / vendor refresh integrity | report here |

We will route the report appropriately if you are not sure where it
belongs — just tell us.

## Scope

**In scope:**

- Anything that breaks the encryption-at-rest guarantee for an
  attacker who has only the database file (and the SQLCipher header /
  page format integrity).
- Cgo binding bugs that allow Go code to read decrypted data it should
  not see, leak keys to logs, or corrupt the database under specific
  PRAGMA combinations.
- Build / release supply-chain issues (unsigned artifacts,
  reproducibility breaks, dependency confusion).
- Anything that breaks the documented contract between this driver and
  `database/sql`.

**Out of scope:**

- Attacks requiring access to process memory of the running Go program
  with the encryption key already in heap (this is a fundamental cgo
  property, not a defect of this driver).
- Attacks against SQLite itself that are not related to encryption — go
  to `sqlite.org`.
- Theoretical attacks against AES-256-CBC + HMAC-SHA-512 that require
  capabilities outside SQLCipher's stated threat model.

## Supported versions

| Major | Status      | Security fixes |
|-------|-------------|----------------|
| v4.x  | Active      | Latest patch only |
| v3.x  | End of life | Not supported. Migrate via `PRAGMA cipher_migrate`. |

## Safe harbour

We will not pursue legal action against researchers who:

- Make a good-faith effort to comply with this policy.
- Avoid privacy violations, destruction of data, and disruption to
  other users.
- Provide a reasonable time for us to fix issues before public
  disclosure.

We consider security research a public good and we are grateful for
yours.
