# ADR 0002: Migrate to terraform-plugin-framework (protocol 6)

- Status: proposed (not implemented)
- Date: 2026-11-10

## Context

The provider is built on terraform-plugin-sdk/v2 (v2.40.1) and serves plugin
protocol 5. HashiCorp develops new features in terraform-plugin-framework
(v1.19): protocol 6, nested attributes, real null/unknown handling, plan
modifiers, write-only and ephemeral values, functions, actions and list
resources. SDKv2 still gets fixes but no new features. The D10 target is
plugin protocol 6.

Things that matter for this provider:

- **Optional + computed values.** The v202506 API has proto3 `optional bool`
  fields (`aws.collect_flow_logs` and others) whose "unset" means "API
  default". SDKv2 can't tell null from `false` without reading the raw
  config (`configured()` in `cloudexport_convert.go`). The framework models
  null directly.
- **Deprecated attributes** (`bgp`, `ibm`, `aws.multiple_buckets`) have to be
  carried over from prior state on read so configurations that still set them
  show no diff. That works in both SDKs.
- **State compatibility.** About 40 attributes in nested single-item blocks
  (`aws { }` and so on), stored as lists. A framework port must declare
  `ListNestedBlock` with `listvalidator.SizeAtMost(1)` for each of them, keep
  every attribute type and keep `SchemaVersion` 0. Then the framework reads
  state written by v0.4.x and v0.5.x without an upgrader. Converting the
  blocks to `SingleNestedAttribute` would be nicer, but it changes the state
  shape and the HCL syntax (`aws = { }`), so it needs a major version.
- **Provider consolidation is still open** (findings F3, F10): either keep
  `kentik-cloudexport` and `kentik-synthetics`, or merge them into one
  `kentik/kentik` provider generated against api-schema-public. The
  `internal/kentikapi` adapter (the synthetics fork is building the same
  shape) has no Terraform imports, so it is unaffected by the choice of
  plugin SDK.

## Options

1. **Stay on SDKv2.** Lowest risk. Security and API work is done (0.5.x).
   Protocol 5 keeps working on current Terraform and OpenTofu.
2. **Migrate this provider now, in place, with terraform-plugin-mux.** Serve
   protocol 6 through `tf5to6server` plus `tf6muxserver`, move the two data
   sources to the framework first (no state), then the resource. Keep block
   syntax and types for state compatibility. Estimate: 2-3 days including
   tests, mostly to replicate SDKv2's empty-list vs null behaviour so that no
   existing configuration shows a diff.
3. **Migrate as part of the consolidated `kentik/kentik` provider.** Build the
   new provider on the framework from the start, port cloud export and
   synthetics into it, and keep the old providers on SDKv2 (security fixes
   only) with a documented `moved`/import path.

## Recommendation

**Option 3 if Ingmar decides to consolidate (F3/F10); otherwise option 2 as a
0.6.0 release.** Don't migrate this repository on its own before that
decision. A framework port of the old type names
(`kentik-cloudexport_item`) would be redone, or thrown away, during
consolidation, and 0.5.x already contains the user-facing fixes.

If option 2 is chosen:

1. Change `main.go` to `tf6server.Serve` with `tf6muxserver` over
   `tf5to6server.UpgradeServer(provider.New().GRPCProvider)` and a new
   framework provider. Change `terraform-registry-manifest.json` to
   `"protocol_versions": ["6.0"]`. This needs Terraform >= 1.0 or OpenTofu >= 1.6.
2. Port `kentik-cloudexport_list` and the `kentik-cloudexport_item` data source.
3. Port the resource with identical block structure, attribute types and
   `SchemaVersion: 0`. Use `UseStateForUnknown` for server-computed
   attributes and null-aware handling for the proto3 optional bools.
4. Gate it with a state-compatibility test that writes state with the SDKv2
   implementation (or a recorded v0.4.1/v0.5.x state) and checks that
   `terraform plan` on the framework implementation is empty. Use
   terraform-plugin-testing `ExternalProviders` where network is allowed and
   recorded state otherwise.
5. Remove SDKv2 and mux once nothing is served through them.

## Consequences

- Until then the provider serves protocol 5. CI covers current Terraform and
  OpenTofu, and the registry manifest says `5.0`.
- `kentikapi`, the fakes in `kentikapitest` and the conversion tests carry over
  unchanged into either migration path.
