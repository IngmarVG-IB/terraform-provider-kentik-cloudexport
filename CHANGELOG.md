# Changelog

All notable changes to this project are documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Security
- Bumped every dependency to its current release; `govulncheck ./...` went
  from 13 reachable vulnerabilities (golang.org/x/net, google.golang.org/grpc,
  golang.org/x/text) to none.

### Changed
- Minimum Go version is 1.26 (was 1.17). CI tests Go 1.26 and 1.27.
- terraform-plugin-sdk/v2 v2.20.0 -> v2.40.1, grpc v1.48.0 -> v1.85.0,
  protobuf v1.28.1 -> v1.36.12.
- tfplugindocs and gofumpt are tracked as Go tool dependencies in `go.mod`.
- golangci-lint configuration migrated to v2.
- Examples use the registry source address `kentik/kentik-cloudexport`.
- `.goreleaser.yml` migrated to GoReleaser v2; releases are created as drafts
  and include `terraform-registry-manifest.json`.

### Added
- `AGENTS.md`, `CLAUDE.md`, `SECURITY.md`, `CHANGELOG.md`, `kentik-api.json`.
- Makefile targets `build`, `test`, `testacc`, `lint`, `vuln`, `generate`,
  `fmt`, `docs`, `check-generate`, `check-go-mod`, `check-api-manifest`,
  `validate-examples`.
- `.github/workflows/ci.yml`: read-only token, SHA-pinned actions, Go 1.26/1.27
  x Terraform/OpenTofu test matrix, lint, govulncheck, generated-docs drift
  check, `kentik-api.json` validation and `terraform validate` on examples.
- Release workflow publishes GitHub build provenance attestations.

### Removed
- Super-Linter workflow and the `tools/check_*.sh` / `tools/fmt.sh` scripts
  (replaced by Makefile targets and `ci.yml`).
- Acceptance tests no longer run on `release` events; they run on manual
  dispatch only.

## [0.4.1] - 2021-10-21

See the GitHub release notes for 0.4.1 and earlier.

[Unreleased]: https://github.com/kentik/terraform-provider-kentik-cloudexport/compare/v0.4.1...HEAD
[0.4.1]: https://github.com/kentik/terraform-provider-kentik-cloudexport/releases/tag/v0.4.1
