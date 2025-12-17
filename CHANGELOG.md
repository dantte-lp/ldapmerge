# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2025-12-17

### Added

- **CLI Commands**
  - `sync` - Full synchronization pipeline: pull → merge → push
  - `merge` - Local merge of JSON configuration files
  - `nsx pull` - Fetch LDAP identity sources from NSX
  - `nsx push` - Push LDAP identity sources to NSX
  - `nsx get` - Get specific LDAP identity source
  - `nsx delete` - Delete LDAP identity source
  - `nsx probe` - Test LDAP server connection
  - `nsx fetch-cert` - Fetch SSL certificate from LDAP server
  - `nsx search` - Search users/groups in LDAP identity source
  - `server` - Start REST API server
  - `version` - Show version information

- **API Server**
  - REST API with Huma framework
  - Scalar documentation at `/docs`
  - OpenAPI 3.0 schema at `/openapi.json`
  - Endpoints: `/api/merge`, `/api/history`, `/api/configs`, `/api/health`
  - SQLite storage for history and configurations

- **NSX Integration**
  - Full VMware NSX 4.2 LDAP Identity Sources API support
  - TLS certificate verification with `--insecure` option
  - Configurable request timeout

- **Merge Logic**
  - Certificate matching by LDAP server URL
  - Support for multiple certificates per server
  - Preserves all original configuration fields

- **Logging**
  - Structured JSON logging with slog
  - Log rotation with lumberjack (100MB, 5 files, 30 days, gzip)
  - Configurable log levels: debug, info, warn, error
  - Console output option

- **Build & Distribution**
  - Cross-platform binaries: Linux amd64, Windows amd64, macOS ARM64
  - Docker image with multi-arch support (amd64, arm64)
  - Version injection via ldflags
  - Makefile for build automation

- **CLI UX**
  - Colorful ASCII banner
  - Emoji icons for commands and sections
  - Structured help output
  - Configuration via flags, environment variables, and config file

- **Documentation**
  - README.md with quick start guide
  - CLI.md with full command reference
  - API.md with REST API documentation
  - QUICK_START.md with usage scenarios

### Security

- Non-root Docker user
- TLS certificate verification by default
- No secrets in logs

---

## Types of changes

- `Added` for new features.
- `Changed` for changes in existing functionality.
- `Deprecated` for soon-to-be removed features.
- `Removed` for now removed features.
- `Fixed` for any bug fixes.
- `Security` in case of vulnerabilities.

[Unreleased]: https://github.com/dantte-lp/ldapmerge/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/dantte-lp/ldapmerge/releases/tag/v1.0.0
