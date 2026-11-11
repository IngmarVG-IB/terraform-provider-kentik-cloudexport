package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const itemDataSourcesConfig = providerConfig + `
data "kentik-cloudexport_item" "aws" {
  id = "1"
}
data "kentik-cloudexport_item" "gce" {
  id = "2"
}
data "kentik-cloudexport_item" "oci" {
  id = "3"
}
data "kentik-cloudexport_item" "azure" {
  id = "4"
}
`

func TestDataSourceCloudExportItem(t *testing.T) {
	t.Parallel()
	srv, _ := newFake(t, seedExports()...)

	const aws, gce, oci, azure = "data.kentik-cloudexport_item.aws", "data.kentik-cloudexport_item.gce",
		"data.kentik-cloudexport_item.oci", "data.kentik-cloudexport_item.azure"

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: factories(srv),
		Steps: []resource.TestStep{{
			Config: itemDataSourcesConfig,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(aws, "id", "1"),
				resource.TestCheckResourceAttr(aws, "type", "CLOUD_EXPORT_TYPE_KENTIK_MANAGED"),
				resource.TestCheckResourceAttr(aws, "enabled", "true"),
				resource.TestCheckResourceAttr(aws, "name", "test_terraform_aws_export"),
				resource.TestCheckResourceAttr(aws, "description", "terraform aws cloud export"),
				resource.TestCheckResourceAttr(aws, "plan_id", "11467"),
				resource.TestCheckResourceAttr(aws, "cloud_provider", "aws"),
				resource.TestCheckResourceAttr(aws, "cdate", "2025-07-01T10:00:00Z"),
				resource.TestCheckResourceAttr(aws, "edate", "2025-07-01T10:00:00Z"),
				resource.TestCheckResourceAttr(aws, "aws.0.bucket", "terraform-aws-bucket"),
				resource.TestCheckResourceAttr(aws, "aws.0.iam_role_arn", "arn:aws:iam::003740049406:role/trafficTerraformIngestRole"),
				resource.TestCheckResourceAttr(aws, "aws.0.region", "us-east-2"),
				resource.TestCheckResourceAttr(aws, "aws.0.collect_flow_logs", "true"),
				resource.TestCheckResourceAttr(aws, "aws.0.collect_metrics", "true"),
				resource.TestCheckResourceAttr(aws, "aws.0.metadata_only", "false"),
				resource.TestCheckResourceAttr(aws, "aws.0.delete_after_read", "false"),
				resource.TestCheckResourceAttr(aws, "aws.0.multiple_buckets", "false"),
				resource.TestCheckResourceAttr(aws, "aws.0.secondary_aws_accounts.#", "2"),
				resource.TestCheckResourceAttr(aws, "aws.0.secondary_aws_accounts.1", "222222222222"),
				resource.TestCheckResourceAttr(aws, "sampling.0.sampling_enabled", "true"),
				resource.TestCheckResourceAttr(aws, "sampling.0.sampling_type", "CLOUD_EXPORT_SAMPLING_TYPE_RATE"),
				resource.TestCheckResourceAttr(aws, "sampling.0.sampling_rate", "100"),
				resource.TestCheckResourceAttr(aws, "current_status.0.status", "OK"),
				resource.TestCheckResourceAttr(aws, "current_status.0.error_message", "No errors"),
				resource.TestCheckResourceAttr(aws, "current_status.0.flow_found", "true"),
				resource.TestCheckResourceAttr(aws, "current_status.0.api_access", "true"),
				resource.TestCheckResourceAttr(aws, "current_status.0.storage_account_access", "true"),
				resource.TestCheckResourceAttr(aws, "bgp.#", "0"),
				resource.TestCheckResourceAttr(aws, "gce.#", "0"),

				resource.TestCheckResourceAttr(gce, "type", "CLOUD_EXPORT_TYPE_CUSTOMER_MANAGED"),
				resource.TestCheckResourceAttr(gce, "cloud_provider", "gce"),
				resource.TestCheckResourceAttr(gce, "plan_id", "21600"),
				resource.TestCheckResourceAttr(gce, "gce.0.project", "project gce"),
				resource.TestCheckResourceAttr(gce, "gce.0.subscription", "subscription gce"),
				resource.TestCheckResourceAttr(gce, "gce.0.collect_flow_logs", "true"),
				resource.TestCheckResourceAttr(gce, "current_status.0.status", "NOK"),
				resource.TestCheckResourceAttr(gce, "current_status.0.error_message", "Timeout"),
				resource.TestCheckResourceAttr(gce, "current_status.0.flow_found", "false"),
				resource.TestCheckResourceAttr(gce, "cdate", ""),

				resource.TestCheckResourceAttr(oci, "enabled", "false"),
				resource.TestCheckResourceAttr(oci, "cloud_provider", "oci"),
				resource.TestCheckResourceAttr(oci, "oci.0.oci_tenancy_id", "ocid1.tenancy.oc1..aaaa"),
				resource.TestCheckResourceAttr(oci, "oci.0.oci_default_region", "eu-frankfurt-1"),
				resource.TestCheckResourceAttr(oci, "oci.0.oci_compartment_id.0", "ocid1.compartment.oc1..bbbb"),
				resource.TestCheckResourceAttr(oci, "oci.0.oci_collect_flow_logs", "true"),
				resource.TestCheckResourceAttr(oci, "oci.0.oci_bucket_name", "flow-logs"),

				resource.TestCheckResourceAttr(azure, "cloud_provider", "azure"),
				resource.TestCheckResourceAttr(azure, "azure.0.location", "centralus"),
				resource.TestCheckResourceAttr(azure, "azure.0.resource_group", "traffic-generator"),
				resource.TestCheckResourceAttr(azure, "azure.0.storage_account", "kentikstorage"),
				resource.TestCheckResourceAttr(azure, "azure.0.subscription_id", "784bd5ec-122b-41b7-9719-22f23d5b49c8"),
				resource.TestCheckResourceAttr(azure, "azure.0.security_principal_enabled", "true"),
			),
		}},
	})
}

func TestDataSourceCloudExportItem_NotFound(t *testing.T) {
	t.Parallel()
	srv, _ := newFake(t, seedExports()...)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: factories(srv),
		Steps: []resource.TestStep{{
			Config: providerConfig + `
data "kentik-cloudexport_item" "missing" {
  id = "999"
}
`,
			ExpectError: regexp.MustCompile(`(?s)Failed to read cloud export item.*NotFound`),
		}},
	})
}
