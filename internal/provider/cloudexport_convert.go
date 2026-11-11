package provider

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	cloudexport "github.com/kentik/api-schema-public/gen/go/kentik/cloud_export/v202506"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Terraform keeps the lower-case cloud_provider values of the v202101beta1 API
// ("aws", "azure", "gce"); v202506 uses the CloudProvider enum.
var cloudProviderByKey = map[string]cloudexport.CloudProvider{ //nolint:gochecknoglobals
	awsKey:   cloudexport.CloudProvider_CLOUD_PROVIDER_AWS,
	azureKey: cloudexport.CloudProvider_CLOUD_PROVIDER_AZURE,
	gceKey:   cloudexport.CloudProvider_CLOUD_PROVIDER_GCE,
	ociKey:   cloudexport.CloudProvider_CLOUD_PROVIDER_OCI,
}

func supportedCloudProviders() []string {
	keys := make([]string, 0, len(cloudProviderByKey))
	for k := range cloudProviderByKey {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func cloudProviderKey(p cloudexport.CloudProvider) string {
	for k, v := range cloudProviderByKey {
		if v == p {
			return k
		}
	}
	if p == cloudexport.CloudProvider_CLOUD_PROVIDER_UNSPECIFIED {
		return ""
	}
	return strings.ToLower(strings.TrimPrefix(p.String(), "CLOUD_PROVIDER_"))
}

func enumNames(names map[int32]string) []string {
	out := make([]string, 0, len(names))
	for _, n := range names {
		out = append(out, n)
	}
	slices.Sort(out)
	return out
}

func formatTimestamp(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return ""
	}
	return ts.AsTime().UTC().Format(time.RFC3339)
}

// priorFunc returns the current Terraform value at a path such as
// "aws.0.multiple_buckets", or nil when there is none (data sources).
type priorFunc func(key string) any

// cloudExportToMap converts an API object into Terraform attribute values.
// Attributes that no longer exist in the API (bgp, ibm, aws.multiple_buckets)
// are left alone, or carried over from prior, so existing configurations do not
// show a perpetual diff.
func cloudExportToMap(e *cloudexport.CloudExport, prior priorFunc) map[string]any {
	o := map[string]any{
		"id":             e.GetId(),
		"type":           e.GetType().String(),
		"enabled":        e.GetEnabled(),
		"name":           e.GetName(),
		"description":    e.GetDescription(),
		"plan_id":        e.GetPlanId(),
		"cloud_provider": cloudProviderKey(e.GetCloudProvider()),
		"cdate":          formatTimestamp(e.GetCdate()),
		"edate":          formatTimestamp(e.GetEdate()),
		awsKey:           []any{},
		azureKey:         []any{},
		gceKey:           []any{},
		ociKey:           []any{},
		samplingKey:      []any{},
		statusKey:        []any{},
	}

	if p := e.GetAws(); p != nil {
		multipleBuckets := false
		if prior != nil {
			if v, ok := prior(awsKey + ".0." + multipleBucketsKey).(bool); ok {
				multipleBuckets = v
			}
		}
		o[awsKey] = []any{map[string]any{
			"iam_role_arn":                   p.GetIamRoleArn(),
			"region":                         p.GetRegion(),
			"bucket":                         p.GetBucket(),
			"bucket_prefix":                  p.GetBucketPrefix(),
			"metadata_only":                  p.GetMetadataOnly(),
			"collect_flow_logs":              p.GetCollectFlowLogs(),
			"collect_firewall_logs":          p.GetCollectFirewallLogs(),
			"collect_metrics":                p.GetCollectMetrics(),
			"delete_after_read":              p.GetDeleteAfterRead(), //nolint:staticcheck // deprecated in the API, still exposed
			"aws_iam_role_arn_is_org":        p.GetAwsIamRoleArnIsOrg(),
			"secondary_aws_accounts":         p.GetSecondaryAwsAccounts(),
			"secondary_aws_blocked_accounts": p.GetSecondaryAwsBlockedAccounts(),
			"secondary_aws_regions":          p.GetSecondaryAwsRegions(),
			"secondary_aws_suffix":           p.GetSecondaryAwsSuffix(),
			multipleBucketsKey:               multipleBuckets,
		}}
	}
	if p := e.GetAzure(); p != nil {
		o[azureKey] = []any{map[string]any{
			"location":                   p.GetLocation(),
			"resource_group":             p.GetResourceGroup(),
			"storage_account":            p.GetStorageAccount(),
			"subscription_id":            p.GetSubscriptionId(),
			"security_principal_enabled": p.GetSecurityPrincipalEnabled(),
		}}
	}
	if p := e.GetGce(); p != nil {
		o[gceKey] = []any{map[string]any{
			"project":           p.GetProject(),
			"subscription":      p.GetSubscription(),
			"metadata_only":     p.GetMetadataOnly(),
			"collect_flow_logs": p.GetCollectFlowLogs(),
			"collect_metrics":   p.GetCollectMetrics(),
		}}
	}
	if p := e.GetOci(); p != nil {
		o[ociKey] = []any{map[string]any{
			"oci_user_id":                 p.GetOciUserId(),
			"oci_tenancy_id":              p.GetOciTenancyId(),
			"oci_compartment_id":          p.GetOciCompartmentId(),
			"oci_default_region":          p.GetOciDefaultRegion(),
			"oci_collect_flow_logs":       p.GetOciCollectFlowLogs(),
			"oci_bucket_name":             p.GetOciBucketName(),
			"oci_bucket_namespace_name":   p.GetOciBucketNamespaceName(),
			"oci_service_connector_ocid":  p.GetOciServiceConnectorOcid(),
			"oci_flow_object_name_prefix": p.GetOciFlowObjectNamePrefix(),
			"metadata_only":               p.GetMetadataOnly(),
		}}
	}
	if s := e.GetSampling(); s != nil {
		o[samplingKey] = []any{map[string]any{
			"sampling_enabled": s.GetSamplingEnabled(),
			"sampling_type":    s.GetSamplingType().String(),
			"sampling_rate":    int(s.GetSamplingRate()),
		}}
	}
	if s := e.GetCurrentStatus(); s != nil {
		o[statusKey] = []any{map[string]any{
			"status":                 s.GetStatus(),
			"error_message":          s.GetErrorMessage(),
			"flow_found":             s.GetFlowFound(),
			"api_access":             s.GetApiAccess(),
			"storage_account_access": s.GetStorageAccountAccess(),
		}}
	}
	return o
}

// resourceDataToCloudExport builds the API object for create and update.
// Output-only fields (id on create, current_status, cdate, edate) are not sent.
func resourceDataToCloudExport(d *schema.ResourceData) (*cloudexport.CloudExport, error) {
	exportType, ok := cloudexport.CloudExportType_value[d.Get("type").(string)]
	if !ok {
		return nil, fmt.Errorf("invalid type %q", d.Get("type"))
	}
	e := &cloudexport.CloudExport{
		Id:          d.Id(),
		Type:        cloudexport.CloudExportType(exportType),
		Enabled:     d.Get("enabled").(bool),
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		PlanId:      d.Get("plan_id").(string),
	}

	cloudProvider := d.Get("cloud_provider").(string)
	if cloudProvider == ibmKey {
		return nil, errors.New("cloud_provider \"ibm\" is not supported by the Kentik cloud_export v202506 API")
	}
	p, ok := cloudProviderByKey[cloudProvider]
	if !ok {
		return nil, fmt.Errorf("cloud_provider must be one of %v, got %q", supportedCloudProviders(), cloudProvider)
	}
	e.CloudProvider = p

	r := blockReader{d: d}
	if !r.present(cloudProvider) {
		return nil, fmt.Errorf("for cloud_provider=%[1]q, a %[1]s {...} block is required", cloudProvider)
	}
	switch p {
	case cloudexport.CloudProvider_CLOUD_PROVIDER_AWS:
		e.CloudSpecificProperties = &cloudexport.CloudExport_Aws{Aws: r.aws()}
	case cloudexport.CloudProvider_CLOUD_PROVIDER_AZURE:
		e.CloudSpecificProperties = &cloudexport.CloudExport_Azure{Azure: r.azure()}
	case cloudexport.CloudProvider_CLOUD_PROVIDER_GCE:
		e.CloudSpecificProperties = &cloudexport.CloudExport_Gce{Gce: r.gce()}
	case cloudexport.CloudProvider_CLOUD_PROVIDER_OCI:
		e.CloudSpecificProperties = &cloudexport.CloudExport_Oci{Oci: r.oci()}
	default:
		return nil, fmt.Errorf("unsupported cloud_provider %q", cloudProvider)
	}

	if r.present(samplingKey) {
		sampling, err := r.sampling()
		if err != nil {
			return nil, err
		}
		e.Sampling = sampling
	}
	return e, nil
}

// blockReader reads attributes of single-item nested blocks.
type blockReader struct {
	d *schema.ResourceData
}

func (r blockReader) present(blockKey string) bool {
	v, ok := r.d.GetOk(blockKey)
	if !ok {
		return false
	}
	l, ok := v.([]any)
	return ok && len(l) > 0 && l[0] != nil
}

func (r blockReader) str(blockKey, attr string) string {
	v, _ := r.d.Get(blockKey + ".0." + attr).(string)
	return v
}

func (r blockReader) boolean(blockKey, attr string) bool {
	v, _ := r.d.Get(blockKey + ".0." + attr).(bool)
	return v
}

func (r blockReader) strings(blockKey, attr string) []string {
	raw, _ := r.d.Get(blockKey + ".0." + attr).([]any)
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// optionalBool handles proto3 `optional bool` fields backed by Optional+Computed
// attributes: send the configured value; on update send the value already in
// state (server-computed); on create leave it unset so the API default applies.
//
// Only aws has proto3 optional fields today; the block key stays a parameter
// so other blocks can use it when the API adds them.
func (r blockReader) optionalBool(blockKey, attr string) *bool { //nolint:unparam // see above
	if configured(r.d, blockKey, attr) || !r.d.IsNewResource() {
		return proto.Bool(r.boolean(blockKey, attr))
	}
	return nil
}

// configured reports whether blockKey.0.attr is set in the configuration.
func configured(d *schema.ResourceData, blockKey, attr string) bool {
	raw := d.GetRawConfig()
	if raw.IsNull() || !raw.IsKnown() || !raw.Type().IsObjectType() || !raw.Type().HasAttribute(blockKey) {
		return false
	}
	b := raw.GetAttr(blockKey)
	if b.IsNull() || !b.IsKnown() || b.LengthInt() == 0 {
		return false
	}
	obj := b.Index(cty.NumberIntVal(0))
	if obj.IsNull() || !obj.Type().HasAttribute(attr) {
		return false
	}
	return !obj.GetAttr(attr).IsNull()
}

func (r blockReader) aws() *cloudexport.AwsProperties {
	const k = awsKey
	return &cloudexport.AwsProperties{
		IamRoleArn:                  r.str(k, "iam_role_arn"),
		Region:                      r.str(k, "region"),
		Bucket:                      r.str(k, "bucket"),
		BucketPrefix:                r.str(k, "bucket_prefix"),
		MetadataOnly:                r.optionalBool(k, "metadata_only"),
		CollectFlowLogs:             r.optionalBool(k, "collect_flow_logs"),
		CollectFirewallLogs:         r.optionalBool(k, "collect_firewall_logs"),
		CollectMetrics:              r.optionalBool(k, "collect_metrics"),
		DeleteAfterRead:             r.boolean(k, "delete_after_read"), //nolint:staticcheck // deprecated in the API, still supported
		AwsIamRoleArnIsOrg:          r.boolean(k, "aws_iam_role_arn_is_org"),
		SecondaryAwsAccounts:        r.strings(k, "secondary_aws_accounts"),
		SecondaryAwsBlockedAccounts: r.strings(k, "secondary_aws_blocked_accounts"),
		SecondaryAwsRegions:         r.strings(k, "secondary_aws_regions"),
		SecondaryAwsSuffix:          r.str(k, "secondary_aws_suffix"),
	}
}

func (r blockReader) azure() *cloudexport.AzureProperties {
	const k = azureKey
	return &cloudexport.AzureProperties{
		Location:                 r.str(k, "location"),
		ResourceGroup:            r.str(k, "resource_group"),
		StorageAccount:           r.str(k, "storage_account"),
		SubscriptionId:           r.str(k, "subscription_id"),
		SecurityPrincipalEnabled: r.boolean(k, "security_principal_enabled"),
	}
}

func (r blockReader) gce() *cloudexport.GceProperties {
	const k = gceKey
	return &cloudexport.GceProperties{
		Project:         r.str(k, "project"),
		Subscription:    r.str(k, "subscription"),
		MetadataOnly:    r.boolean(k, "metadata_only"),
		CollectFlowLogs: r.boolean(k, "collect_flow_logs"),
		CollectMetrics:  r.boolean(k, "collect_metrics"),
	}
}

func (r blockReader) oci() *cloudexport.OciProperties {
	const k = ociKey
	return &cloudexport.OciProperties{
		OciUserId:               r.str(k, "oci_user_id"),
		OciTenancyId:            r.str(k, "oci_tenancy_id"),
		OciCompartmentId:        r.strings(k, "oci_compartment_id"),
		OciDefaultRegion:        r.str(k, "oci_default_region"),
		OciCollectFlowLogs:      r.boolean(k, "oci_collect_flow_logs"),
		OciBucketName:           r.str(k, "oci_bucket_name"),
		OciBucketNamespaceName:  r.str(k, "oci_bucket_namespace_name"),
		OciServiceConnectorOcid: r.str(k, "oci_service_connector_ocid"),
		OciFlowObjectNamePrefix: r.str(k, "oci_flow_object_name_prefix"),
		MetadataOnly:            r.boolean(k, "metadata_only"),
	}
}

func (r blockReader) sampling() (*cloudexport.CloudExportSamplingProperties, error) {
	const k = samplingKey
	s := &cloudexport.CloudExportSamplingProperties{
		SamplingEnabled: r.boolean(k, "sampling_enabled"),
	}
	if t := r.str(k, "sampling_type"); t != "" {
		v, ok := cloudexport.CloudExportSamplingType_value[t]
		if !ok {
			return nil, fmt.Errorf("invalid sampling.sampling_type %q", t)
		}
		s.SamplingType = cloudexport.CloudExportSamplingType(v)
	}
	rate, _ := r.d.Get(k + ".0.sampling_rate").(int)
	if rate < 0 || rate > 1999 {
		return nil, fmt.Errorf("sampling.sampling_rate must be between 0 and 1999, got %d", rate)
	}
	s.SamplingRate = uint32(rate)
	return s, nil
}
