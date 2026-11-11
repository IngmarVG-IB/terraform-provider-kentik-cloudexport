# ADR 0001: Build on the api-schema-public gRPC clients through an internal adapter

- Status: accepted
- Date: 2026-11-10

## Context

The provider used `github.com/kentik/community_sdk_golang`, which wraps
`cloud_export/v202101beta1` behind hand-written models. That SDK has had no
release since 2021, pins vulnerable dependencies and does not know about the
stable `cloud_export/v202506` API (OCI, sampling, firewall logs, secondary
AWS accounts). Kentik publishes generated Go gRPC clients for every API in
`github.com/kentik/api-schema-public/gen/go`. The synthetics provider fork has
the same problem and is being moved the same way, so the two providers may
later be merged into one `kentik/kentik` provider.

## Decision

- Use the generated `cloud_export/v202506` gRPC client directly. No
  hand-written API models.
- Put connection concerns in `internal/kentikapi`:
  `kentikapi.New(ctx, kentikapi.Config{Email, Token, Endpoint, Region, UserAgent, Timeout, Retry, ...})`
  returns a `*Client` with one `*grpc.ClientConn` and a field per service
  (`CloudExport`). It provides TLS (1.3 minimum), per-RPC credentials
  (`X-CH-Auth-Email`, `X-CH-Auth-API-Token`, refused on plaintext unless
  `Insecure` is set), a per-attempt timeout, retries with jittered exponential
  backoff on `Unavailable` and `ResourceExhausted`, and an optional log hook.
  It has no Terraform dependency.
- Put the fake servers in `internal/kentikapi/kentikapitest` (bufconn, records
  metadata, in-memory CloudExportAdminService) so tests are offline and other
  services can add fakes the same way.
- Keep the legacy `apiurl` derivation rules (`EndpointFromAPIURL`) byte for byte
  so existing configurations reach the same host.
- Use the package shape agreed for both provider forks (`kentikapi.New(ctx, Config)`,
  one field per service) so that merging the providers is a matter of adding
  fields to `Client`. The synthetics fork's copy was not available to compare
  against when this was written; reconcile the two before merging.

## Consequences

- One dependency on Kentik code (`api-schema-public`), pinned by pseudo-version
  and recorded in `kentik-api.json`; Dependabot proposes updates.
- api-schema-public's own go.mod has old grpc/protobuf floors; we override them
  with current versions (MVS), which govulncheck confirms are clean.
- The adapter is duplicated in two repositories until the providers merge.
  The shared shape keeps that merge mechanical; a shared public Go module is
  the alternative if they stay separate (decision pending).
- Retrying `CreateCloudExport` on `Unavailable` could in theory create a
  duplicate if the server applied the request; the API rejects duplicate names
  (`AlreadyExists`), which surfaces the problem instead of hiding it.
