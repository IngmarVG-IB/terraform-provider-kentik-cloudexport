# AGENTS.md

Guide for maintainers and coding agents working on
`terraform-provider-kentik-cloudexport` (registry name `kentik/kentik-cloudexport`).

## Purpose

A Terraform/OpenTofu provider that manages Kentik cloud exports: the objects
that tell Kentik how to collect flow logs and metadata from AWS, Azure, Google
Cloud and OCI. It has real users, so backward compatibility of resource names,
attribute names and state is a hard requirement.

## Layout

| path | what |
|---|---|
| `main.go` | Plugin entry point (`plugin.Serve`, SDKv2, protocol 5). |
| `internal/kentikapi/` | Service-agnostic adapter over the generated api-schema-public gRPC clients: `kentikapi.New(ctx, Config)` with TLS, per-RPC auth metadata, timeouts, retries, logging hook. No Terraform imports. |
| `internal/kentikapi/kentikapitest/` | Offline fakes: a bufconn gRPC `Server` that records calls and metadata, and an in-memory `CloudExportServer`. |
| `internal/provider/` | Provider schema and configuration, resource `kentik-cloudexport_item`, data sources `kentik-cloudexport_item` and `kentik-cloudexport_list`, plus their tests. |
| `internal/provider/cloudexport_schema.go` / `cloudexport_convert.go` | Terraform schema and the Terraform <-> `cloud_export/v202506` proto mapping. |
| `internal/provider/acceptance_test.go` | Live-tenant tests (skipped unless `TF_ACC=1`) and the sweeper. |
| `docs/UPGRADING.md` | Upgrade guide and attribute mapping from 0.4.x. |
| `examples/` | One directory per resource/data source (`resource.tf`, `data-source.tf`), embedded in the docs by tfplugindocs. |
| `templates/` | tfplugindocs templates. `docs/` is generated from these: do not edit `docs/` by hand. |
| `tools/` | Helper scripts used by the Makefile (`check_api_manifest.sh`, `validate_examples.sh`). |
| `kentik-api.json` | Machine-readable API coverage manifest (read by the kentikcommunity indexer). |
| `docs/adr/` | Architecture decision records. |

## Commands

All targets run offline. `make test` needs a `terraform` (or `tofu`) binary on
`PATH`, or `TF_ACC_TERRAFORM_PATH` pointing at one.

| task | command |
|---|---|
| build | `make build` |
| unit tests (offline) | `make test` |
| acceptance tests (live tenant, opt-in) | `TF_ACC=1 KTAPI_AUTH_EMAIL=... KTAPI_AUTH_TOKEN=... KENTIK_PLAN_ID=... make testacc` |
| lint | `make lint` (golangci-lint v2, pinned in the Makefile, run via `go run`) |
| format | `make fmt` |
| vulnerability scan | `make vuln` (govulncheck, pinned) |
| regenerate docs | `make generate` (alias `make docs`) |
| drift check | `make check-generate`, `make check-go-mod` |
| validate examples | `make validate-examples` (`TF_CLI=tofu` for OpenTofu) |
| validate manifest | `make check-api-manifest` |

### Running a local build in Terraform

Use a dev_overrides CLI config in a temporary directory instead of
`make install`:

```hcl
# /tmp/kentik-dev/terraformrc
provider_installation {
  dev_overrides {
    "registry.terraform.io/kentik/kentik-cloudexport" = "/path/to/repo"
  }
  direct {}
}
```

```sh
make build
TF_CLI_CONFIG_FILE=/tmp/kentik-dev/terraformrc terraform validate
```

Do not run `apply` against a real tenant from a development build.

## Adding a new API service or version

1. Check `github.com/kentik/api-schema-public` (`proto/kentik/<service>/`) for
   the newest **stable** version (no `alpha`/`beta` suffix). Use alpha/beta only
   when no stable version exists, and mark the surface experimental.
2. `go get github.com/kentik/api-schema-public@<commit>` and use the generated
   client in `gen/go/kentik/<service>/<version>`. Do not hand-write API models.
3. Add a field for the service client to `kentikapi.Client` and build it from
   the shared connection in `kentikapi.New`.
4. Add an in-memory fake to `internal/kentikapi/kentikapitest` (embed the
   generated `Unimplemented...Server`, register it on `kentikapitest.Server`)
   so the new RPCs can be tested offline.
5. Map the proto to the Terraform schema. Keep existing attribute names. When
   an attribute must be renamed or change type, bump the resource
   `SchemaVersion` and add a `StateUpgrader` with a unit test.
6. Write unit tests for create, read, update, delete and import with
   `resource.UnitTest` and `factories(srv)` (see `internal/provider/common_test.go`),
   then an example under `examples/`, and run `make generate`.
7. Update `kentik-api.json` (`schema_source.ref` must equal the go.mod
   version; CI checks this), `CHANGELOG.md` and, for user-visible changes,
   `docs/UPGRADING.md`.

## Release process

1. Make sure `CHANGELOG.md` has a section for the version and `master` is green.
2. Push a `vX.Y.Z` tag on the upstream repository. `.github/workflows/release.yml`
   runs GoReleaser v2, GPG-signs `SHA256SUMS` (required by the Terraform
   Registry) and attests build provenance.
3. The GitHub release is created as a **draft**. Review the artifacts, then
   publish it; the Terraform Registry picks it up on publish.

## Conventions

- Go: minimum version is the `go` directive in `go.mod` (1.26); CI tests 1.26 and 1.27.
- Every commit builds and passes `make test lint`.
- Commit messages: imperative subject, wrapped body explaining why.
- Workflows: `permissions: contents: read` at the top, actions pinned by full
  commit SHA with a `# vX.Y.Z` comment, no `pull_request_target`, no
  `${{ github.event.* }}` inside `run:`.

## Do and don't

- Do keep unit tests offline: use the fake gRPC server. No test may need a
  network connection or Kentik credentials unless it is gated behind `TF_ACC`.
- Do keep `govulncheck ./...` clean; justify any exception in `SECURITY.md`.
- Don't commit secrets, tokens, `.terraformrc` files or state files.
- Don't rename or remove resources, data sources or attributes without a
  deprecation period, a state upgrader where needed, and an entry in
  `docs/UPGRADING.md`.
- Don't edit `docs/` by hand; change the schema descriptions, templates or
  examples and run `make generate`.
