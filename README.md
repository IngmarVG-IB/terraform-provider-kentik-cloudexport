# Terraform Provider for Kentik Cloud Export

## Requirements

- [Go](https://go.dev/doc/install) >= 1.26 (to build)
- [Terraform](https://developer.hashicorp.com/terraform/install) >= 1.0 or [OpenTofu](https://opentofu.org/) >= 1.6

## Usage

Detailed user documentation for the provider can be found [here](https://registry.terraform.io/providers/kentik/kentik-cloudexport/latest/docs).

## Development

Anybody who wants to contribute to development is welcome to provide pull requests. To work on the provider, install tools listed in [requirements section](#requirements).

See [AGENTS.md](AGENTS.md) for the full maintainer guide (layout, commands, compatibility rules, release process).

Development steps:
- Build the provider: `make build`
- Run offline unit tests: `make test`
- Run acceptance tests against a live tenant (opt-in): `make testacc`
- Lint, format, vulnerability scan: `make lint`, `make fmt`, `make vuln`
- Regenerate the documentation: `make generate`
- Validate the examples with a dev_overrides config: `make validate-examples`

### Test

Unit tests run the provider against an in-process fake of the Kentik
`CloudExportAdminService` gRPC API (`internal/kentikapi/kentikapitest`, over
bufconn), so they need no credentials and no network. A `terraform` or `tofu`
binary must be on `PATH`.

### Acceptance Tests

Acceptance tests run the provider against a live Kentik tenant and are skipped unless `TF_ACC=1`.
To run it locally, you need to set your login credentials as environment variables as below.

```bash
export KTAPI_AUTH_EMAIL=<Kentik API authentication email>
export KTAPI_AUTH_TOKEN=<Kentik API authentication token>
export KENTIK_PLAN_ID=<plan ID to create test exports under>
make testacc
```

### Debug

For debugging use [Delve debugger](https://github.com/go-delve/delve)

```bash
make build
dlv exec ./terraform-provider-kentik-cloudexport
r -debug
c
# attach with terraform following the just-printed out instruction in your terminal
```

## Release

Release process for the provider is based on Git repository tags that follow [semantic versioning](https://semver.org/). Every tag with format _v\[0-9].\[0-9].\[0-9]_ will trigger automatic build of package and publish it in [Terraform registry](https://registry.terraform.io/providers/kentik/kentik-cloudexport).

To release the provider:
1. Make sure that all code that you want to release is in _master_ branch.
2. Push a tag in _v\[0-9].\[0-9].\[0-9]_ format. The release workflow builds, signs and attests the artifacts and creates a **draft** GitHub release.
3. Review the draft release and publish it; the Terraform Registry picks it up on publish.
