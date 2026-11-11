# Changelog

All notable changes to this project are documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

Upgrade notes: [docs/UPGRADING.md](docs/UPGRADING.md).

### Breaking (API-driven)
- IBM Cloud exports can no longer be managed: `cloud_provider = "ibm"` is
  rejected at plan time because cloud_export v202506 has no IBM support.

### Security
- Bumped every dependency to its current release; `govulncheck ./...` went
  from 13 reachable vulnerabilities (golang.org/x/net, google.golang.org/grpc,
  golang.org/x/text) to none.

### Changed
- The provider now uses the stable Kentik `cloud_export/v202506` gRPC API via
  the generated clients in `github.com/kentik/api-schema-public`, through a new
  internal adapter `internal/kentikapi`. `github.com/kentik/community_sdk_golang`
  (cloud_export v202101beta1) is no longer used.
- Requests are retried on `Unavailable` and `ResourceExhausted` with jittered
  exponential backoff; `log_payloads` logs through `tflog` and never logs credentials.
- `aws.bucket`, `azure.security_principal_enabled` and `gce.subscription` are now
  optional, as in the API.
- Unit tests use terraform-plugin-testing and an in-process bufconn fake of
  CloudExportAdminService instead of a TCP test server.
- Minimum Go version is 1.26 (was 1.17). CI tests Go 1.26 and 1.27.
- terraform-plugin-sdk/v2 v2.20.0 -> v2.40.1, grpc v1.48.0 -> v1.85.0,
  protobuf v1.28.1 -> v1.36.12.
- tfplugindocs and gofumpt are tracked as Go tool dependencies in `go.mod`.
- golangci-lint configuration migrated to v2.
- Examples use the registry source address `kentik/kentik-cloudexport`.
- `.goreleaser.yml` migrated to GoReleaser v2; releases are created as drafts
  and include `terraform-registry-manifest.json`.

### Added
- Provider settings `region` (`KTAPI_REGION`, `US` or `EU`) and `endpoint`
  (`KTAPI_ENDPOINT`, gRPC `host:port`).
- `oci` block for Oracle Cloud Infrastructure exports.
- `sampling` block, `cdate` and `edate` attributes.
- AWS: `bucket_prefix`, `metadata_only`, `collect_flow_logs`,
  `collect_firewall_logs`, `collect_metrics`, `aws_iam_role_arn_is_org`,
  `secondary_aws_accounts`, `secondary_aws_blocked_accounts`,
  `secondary_aws_regions`, `secondary_aws_suffix`.
- GCE: `metadata_only`, `collect_flow_logs` (default `true`), `collect_metrics`.
- `kentik-cloudexport_list.invalid_exports_count`.
- `terraform import` support for `kentik-cloudexport_item`.
- `docs/UPGRADING.md`, ADR 0001 and ADR 0002 (proposal: terraform-plugin-framework migration).
- `make docs` also runs `tfplugindocs validate`; the docs index no longer uses the
  legacy `layout` front matter.
- End-to-end test that provider `email`/`token` arrive as gRPC auth metadata.
- `AGENTS.md`, `CLAUDE.md`, `SECURITY.md`, `CHANGELOG.md`, `kentik-api.json`.
- Makefile targets `build`, `test`, `testacc`, `lint`, `vuln`, `generate`,
  `fmt`, `docs`, `check-generate`, `check-go-mod`, `check-api-manifest`,
  `validate-examples`.
- `.github/workflows/ci.yml`: read-only token, SHA-pinned actions, Go 1.26/1.27
  x Terraform/OpenTofu test matrix, lint, govulncheck, generated-docs drift
  check, `kentik-api.json` validation and `terraform validate` on examples.
- Release workflow publishes GitHub build provenance attestations.

### Deprecated
- `bgp` block and `aws.multiple_buckets`: not part of v202506, accepted and ignored.
- `aws.delete_after_read`: deprecated by the API, still sent.
- `ibm` block.

### Fixed
- `retry.max_attempts` was silently ignored (wrong type assertion); it is honoured now.
- Deleting an export that is already gone no longer fails `terraform destroy`.

### Removed
- Super-Linter workflow and the `tools/check_*.sh` / `tools/fmt.sh` scripts
  (replaced by Makefile targets and `ci.yml`).
- Acceptance tests no longer run on `release` events; they run on manual
  dispatch only.

## [0.4.1] - 2021-10-21

See the GitHub release notes for 0.4.1 and earlier.

[Unreleased]: https://github.com/kentik/terraform-provider-kentik-cloudexport/compare/v0.4.1...HEAD
[0.4.1]: https://github.com/kentik/terraform-provider-kentik-cloudexport/releases/tag/v0.4.1
