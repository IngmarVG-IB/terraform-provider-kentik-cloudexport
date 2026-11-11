# Upgrading from 0.4.x to 0.5.x

Version 0.5 moves the provider from the Kentik `cloud_export/v202101beta1` API
(through `community_sdk_golang`) to the stable `cloud_export/v202506` gRPC API
(generated clients from `github.com/kentik/api-schema-public`).

**Short version:** resource and data source names, attribute names, provider
settings, environment variables and state are unchanged. `terraform plan` after
upgrading should show no changes for AWS, Azure and Google Cloud exports, apart
from deprecation warnings. IBM Cloud exports cannot be managed any more,
because the API no longer supports them.

## What you need to do

1. Bump the version constraint, e.g. `version = "~> 0.5"`, and run `terraform init -upgrade`.
2. Run `terraform plan`. Expect deprecation warnings for `bgp`, `aws.multiple_buckets`
   and `aws.delete_after_read` if you use them. They are harmless.
3. If you manage IBM Cloud exports (`cloud_provider = "ibm"`), remove them from
   state with `terraform state rm <address>` before upgrading, or keep them on 0.4.x.
4. Optionally remove `bgp {}` blocks and `multiple_buckets` from your configuration;
   they have no effect.

## State compatibility

The resource schema version stays at `0`. No attribute was renamed or changed
type, so no state upgrader is needed: every attribute that 0.4.1 wrote to state
still exists with the same type. `TestStateCompatibilityWithV041` in
`internal/provider/cloudexport_internal_test.go` decodes a 0.4.1 state document
against the new schema to keep it that way. If a future change renames an
attribute, it must bump `SchemaVersion` and add a `StateUpgrader`.

## Provider configuration

| setting | env var | 0.4.x | 0.5.x |
|---|---|---|---|
| `email` | `KTAPI_AUTH_EMAIL` | required | unchanged |
| `token` | `KTAPI_AUTH_TOKEN` | required | unchanged |
| `apiurl` | `KTAPI_URL` | base URL; gRPC endpoint derived as `grpc.<host>:443` | unchanged, same derivation rules |
| `retry { max_attempts, min_delay, max_delay }` | `KTAPI_RETRY_*` | `max_attempts` was ignored (bug) | honoured; retries on `Unavailable` and `ResourceExhausted` with jittered exponential backoff |
| `log_payloads` | `KTAPI_LOG_PAYLOADS` | stdout log | `tflog` debug log, credentials never logged |
| `region` | `KTAPI_REGION` | - | **new**: `US` (default, `grpc.api.kentik.com:443`) or `EU` (`grpc.api.kentik.eu:443`) |
| `endpoint` | `KTAPI_ENDPOINT` | - | **new**: explicit gRPC `host:port` (TLS) |

Precedence: `endpoint`, then `apiurl`, then `region`. `retry` is no longer
`ForceNew` (it never had an effect on resources). Authentication is sent as
`X-CH-Auth-Email` / `X-CH-Auth-API-Token` gRPC metadata over TLS 1.3, as before.

## Resource `kentik-cloudexport_item` and data sources

`R` = required, `O` = optional, `C` = computed (set by the API), `D` = deprecated.
The data sources expose the same attributes, all computed.

### Common attributes

| attribute | 0.4.x (v202101beta1) | 0.5.x (v202506) | notes |
|---|---|---|---|
| `id` | C | C | |
| `type` | R | R | same enum names (`CLOUD_EXPORT_TYPE_*`) |
| `enabled` | R | R | |
| `name` | R | R | |
| `description` | O | O | |
| `plan_id` | R | R | |
| `cloud_provider` | R: `aws`, `azure`, `gce`, `ibm` | R: `aws`, `azure`, `gce`, `oci` | lower-case strings kept; mapped to the `CLOUD_PROVIDER_*` enum. `ibm` is rejected at plan time |
| `current_status.*` | C | C | `status`, `error_message`, `flow_found`, `api_access`, `storage_account_access` unchanged |
| `cdate`, `edate` | - | C | **new**, RFC 3339 UTC |
| `sampling { sampling_enabled, sampling_type, sampling_rate }` | - | O+C | **new**. `sampling_type` is `CLOUD_EXPORT_SAMPLING_TYPE_UNSAMPLED` (all flows) or `CLOUD_EXPORT_SAMPLING_TYPE_RATE` (sampled; `sampling_rate` 1-1999) |
| `bgp { apply_bgp, use_bgp_device_id, device_bgp_type }` | O (inner attributes R) | O, D | not in v202506. Accepted and kept in state, never sent to the API. Data sources leave it empty |
| `ibm { bucket }` | O | O, D | not in v202506; using it fails at plan time through `cloud_provider` validation |

### `aws`

| attribute | 0.4.x | 0.5.x | notes |
|---|---|---|---|
| `iam_role_arn` | R | R | |
| `region` | R | R | |
| `bucket` | R | O | optional in v202506 (e.g. metadata-only exports) |
| `delete_after_read` | R | O, D | deprecated by the API; still sent. Default `false` |
| `multiple_buckets` | R | O, D | not in v202506; kept in state, never sent |
| `bucket_prefix` | - | O | new |
| `metadata_only`, `collect_flow_logs`, `collect_firewall_logs`, `collect_metrics` | - | O+C | new; proto3 `optional`, so unset means "API default" (`collect_flow_logs` defaults to `true`, the rest to `false`) |
| `aws_iam_role_arn_is_org` | - | O | new |
| `secondary_aws_accounts`, `secondary_aws_blocked_accounts`, `secondary_aws_regions` | - | O | new, lists of strings |
| `secondary_aws_suffix` | - | O | new |

### `azure`

| attribute | 0.4.x | 0.5.x | notes |
|---|---|---|---|
| `location`, `resource_group`, `storage_account`, `subscription_id` | R | R | |
| `security_principal_enabled` | R | O | default `false` |

### `gce`

| attribute | 0.4.x | 0.5.x | notes |
|---|---|---|---|
| `project` | R | R | |
| `subscription` | R | O | |
| `collect_flow_logs` | - | O | new, default `true` (what 0.4.x exports did) |
| `metadata_only`, `collect_metrics` | - | O | new, default `false` |

### `oci` (new)

`oci_tenancy_id` (R), `oci_default_region` (R), `oci_user_id`, `oci_compartment_id`
(list), `oci_collect_flow_logs` (default `true`), `oci_bucket_name`,
`oci_bucket_namespace_name`, `oci_service_connector_ocid`,
`oci_flow_object_name_prefix`, `metadata_only` (all O).

### Data source `kentik-cloudexport_list`

`items` is unchanged (same attributes as above). New: `invalid_exports_count`,
the number of exports the API skipped because their data is invalid.

## Import

`kentik-cloudexport_item` now supports `terraform import kentik-cloudexport_item.<name> <id>`.
Imported resources have no `bgp` block and `aws.multiple_buckets = false`,
because the API does not return them; if your configuration sets them, the next
plan shows an in-place update that changes nothing in Kentik.

## Deprecation warnings on outputs

Terraform 1.15 and later also warn when an output references a whole
`kentik-cloudexport_item` object, because the object contains deprecated
attributes. Reference specific attributes (for example `.id`) to avoid it.
The deprecated attributes will be removed in the next major version.
