package provider

import (
	"testing"

	"github.com/hashicorp/go-cty/cty"
	ctyjson "github.com/hashicorp/go-cty/cty/json"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	cloudexport "github.com/kentik/api-schema-public/gen/go/kentik/cloud_export/v202506"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// stateV041 is a resource instance as written by provider v0.4.1
// (cloud_export v202101beta1). The v202506 schema must still decode it.
const stateV041 = `{
  "id": "1234",
  "type": "CLOUD_EXPORT_TYPE_KENTIK_MANAGED",
  "enabled": true,
  "name": "aws-export",
  "description": "d",
  "plan_id": "11467",
  "cloud_provider": "aws",
  "aws": [{"bucket": "b", "iam_role_arn": "arn", "region": "us-east-2", "delete_after_read": false, "multiple_buckets": true}],
  "azure": [],
  "gce": [],
  "ibm": [],
  "bgp": [{"apply_bgp": true, "use_bgp_device_id": "1", "device_bgp_type": "router"}],
  "current_status": [{"status": "OK", "error_message": "", "flow_found": true, "api_access": true, "storage_account_access": true}]
}`

// TestStateCompatibilityWithV041 guards the "no state upgrader needed" claim in
// docs/UPGRADING.md: every attribute v0.4.1 stored still exists with the same type.
func TestStateCompatibilityWithV041(t *testing.T) {
	t.Parallel()
	r := resourceCloudExport()
	assert.Equal(t, 0, r.SchemaVersion)

	v, err := ctyjson.Unmarshal([]byte(stateV041), r.CoreConfigSchema().ImpliedType())
	require.NoError(t, err)
	assert.Equal(t, "aws", v.GetAttr("cloud_provider").AsString())
	assert.True(t, v.GetAttr("aws").Index(ctyIndex0()).GetAttr("multiple_buckets").True())

	// The SDK's own state handling must also accept it.
	is, err := r.ShimInstanceStateFromValue(v)
	require.NoError(t, err)
	assert.Equal(t, "1234", is.ID)
}

func TestCloudExportRoundTrip(t *testing.T) {
	t.Parallel()
	in := &cloudexport.CloudExport{
		Id:            "7",
		Type:          cloudexport.CloudExportType_CLOUD_EXPORT_TYPE_CUSTOMER_MANAGED,
		Enabled:       true,
		Name:          "n",
		Description:   "d",
		PlanId:        "p",
		CloudProvider: cloudexport.CloudProvider_CLOUD_PROVIDER_AWS,
		CloudSpecificProperties: &cloudexport.CloudExport_Aws{Aws: &cloudexport.AwsProperties{
			IamRoleArn:                  "arn",
			Region:                      "r",
			Bucket:                      "b",
			BucketPrefix:                "pre",
			MetadataOnly:                proto.Bool(false),
			CollectFlowLogs:             proto.Bool(true),
			CollectFirewallLogs:         proto.Bool(true),
			CollectMetrics:              proto.Bool(false),
			AwsIamRoleArnIsOrg:          true,
			SecondaryAwsAccounts:        []string{"1"},
			SecondaryAwsBlockedAccounts: []string{"2"},
			SecondaryAwsRegions:         []string{"3"},
			SecondaryAwsSuffix:          "s",
		}},
		Sampling: &cloudexport.CloudExportSamplingProperties{
			SamplingEnabled: true,
			SamplingType:    cloudexport.CloudExportSamplingType_CLOUD_EXPORT_SAMPLING_TYPE_RATE,
			SamplingRate:    10,
		},
	}
	r := resourceCloudExport()
	d := r.Data(nil)
	require.False(t, setAttributes(d, cloudExportToMap(in, nil)).HasError())
	d.SetId(in.GetId())

	out, err := resourceDataToCloudExport(d)
	require.NoError(t, err)
	assert.True(t, proto.Equal(in, out), "round trip mismatch:\n in: %v\nout: %v", in, out)
}

func TestCloudExportToMap_CarriesOverDeprecatedAttributes(t *testing.T) {
	t.Parallel()
	e := &cloudexport.CloudExport{
		CloudProvider:           cloudexport.CloudProvider_CLOUD_PROVIDER_AWS,
		CloudSpecificProperties: &cloudexport.CloudExport_Aws{Aws: &cloudexport.AwsProperties{}},
	}
	prior := func(key string) any {
		if key == "aws.0.multiple_buckets" {
			return true
		}
		return nil
	}
	m := cloudExportToMap(e, prior)
	aws := m[awsKey].([]any)[0].(map[string]any)
	assert.Equal(t, true, aws[multipleBucketsKey])
	assert.NotContains(t, m, bgpKey, "bgp must not be overwritten on read")
	assert.NotContains(t, m, ibmKey, "ibm must not be overwritten on read")
}

func TestCloudExportToMap_Enums(t *testing.T) {
	t.Parallel()
	for provider, key := range map[cloudexport.CloudProvider]string{
		cloudexport.CloudProvider_CLOUD_PROVIDER_UNSPECIFIED: "",
		cloudexport.CloudProvider_CLOUD_PROVIDER_AWS:         "aws",
		cloudexport.CloudProvider_CLOUD_PROVIDER_AZURE:       "azure",
		cloudexport.CloudProvider_CLOUD_PROVIDER_GCE:         "gce",
		cloudexport.CloudProvider_CLOUD_PROVIDER_OCI:         "oci",
		cloudexport.CloudProvider(42):                        "42",
	} {
		assert.Equal(t, key, cloudProviderKey(provider), provider.String())
	}
}

func TestResourceDataToCloudExport_OptionalBoolsUnsetOnCreate(t *testing.T) {
	t.Parallel()
	d := schema.TestResourceDataRaw(t, resourceCloudExport().Schema, map[string]any{
		"type": "CLOUD_EXPORT_TYPE_KENTIK_MANAGED", "enabled": true, "name": "n", "plan_id": "p",
		"cloud_provider": "aws",
		"aws":            []any{map[string]any{"iam_role_arn": "arn", "region": "r"}},
	})
	d.MarkNewResource()
	e, err := resourceDataToCloudExport(d)
	require.NoError(t, err)
	aws := e.GetAws()
	assert.Nil(t, aws.CollectFlowLogs, "unset optional bool must be left to the API default")
	assert.Nil(t, aws.MetadataOnly)
	assert.Nil(t, e.GetSampling())
	// Explicitly configured optional bools on create need the raw config, which
	// TestResourceDataRaw does not provide; TestResourceCloudExportAWS_ExplicitOptionalBoolsOnCreate covers them.
}

func TestResourceDataToCloudExport_Errors(t *testing.T) {
	t.Parallel()
	base := func(provider string, block map[string]any) map[string]any {
		raw := map[string]any{
			"type": "CLOUD_EXPORT_TYPE_KENTIK_MANAGED", "enabled": true, "name": "n", "plan_id": "p",
			"cloud_provider": provider,
		}
		for k, v := range block {
			raw[k] = v
		}
		return raw
	}
	tests := map[string]struct {
		raw     map[string]any
		wantErr string
	}{
		"ibm":           {raw: base("ibm", map[string]any{"ibm": []any{map[string]any{"bucket": "b"}}}), wantErr: `cloud_provider "ibm" is not supported`},
		"unknown":       {raw: base("alibaba", nil), wantErr: "cloud_provider must be one of"},
		"missing block": {raw: base("gce", nil), wantErr: "a gce {...} block is required"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			d := schema.TestResourceDataRaw(t, resourceCloudExport().Schema, tc.raw)
			_, err := resourceDataToCloudExport(d)
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func ctyIndex0() cty.Value { return cty.NumberIntVal(0) }
