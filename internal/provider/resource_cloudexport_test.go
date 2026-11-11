package provider_test

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	cloudexport "github.com/kentik/api-schema-public/gen/go/kentik/cloud_export/v202506"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kentik/terraform-provider-kentik-cloudexport/internal/kentikapi/kentikapitest"
)

// checkStored runs fn against the object the fake API holds for resource name,
// so tests assert what was actually sent over gRPC, not only Terraform state.
func checkStored(fake *kentikapitest.CloudExportServer, name string, fn func(*cloudexport.CloudExport) error) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("resource %s not in state", name)
		}
		e := fake.Get(rs.Primary.ID)
		if e == nil {
			return fmt.Errorf("cloud export %s not stored by the fake API", rs.Primary.ID)
		}
		return fn(e)
	}
}

func checkDestroyed(fake *kentikapitest.CloudExportServer) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if n := fake.Len(); n != 0 {
			return fmt.Errorf("%d cloud exports left after destroy", n)
		}
		return nil
	}
}

const awsName = "kentik-cloudexport_item.test_aws"

// The create step uses a v0.4-style configuration, including attributes that
// v202506 dropped (multiple_buckets, bgp), to prove existing configs still apply.
const awsCreateConfig = providerConfig + `
resource "kentik-cloudexport_item" "test_aws" {
  name           = "resource_test_terraform_aws_export"
  type           = "CLOUD_EXPORT_TYPE_KENTIK_MANAGED"
  enabled        = true
  description    = "resource test aws export"
  plan_id        = "9948"
  cloud_provider = "aws"
  bgp {
    apply_bgp         = true
    use_bgp_device_id = "1234"
    device_bgp_type   = "router"
  }
  aws {
    bucket            = "resource-terraform-aws-bucket"
    iam_role_arn      = "arn:aws:iam::003740049406:role/trafficTerraformIngestRole"
    region            = "eu-central-1"
    delete_after_read = true
    multiple_buckets  = true
  }
}
`

const awsUpdateConfig = providerConfig + `
resource "kentik-cloudexport_item" "test_aws" {
  name           = "resource_test_terraform_aws_export_updated"
  type           = "CLOUD_EXPORT_TYPE_CUSTOMER_MANAGED"
  enabled        = false
  description    = "resource test aws export updated"
  plan_id        = "3333"
  cloud_provider = "aws"
  aws {
    bucket                         = "resource-terraform-aws-bucket-updated"
    bucket_prefix                  = "flows/"
    iam_role_arn                   = "arn:aws:iam::003740049406:role/trafficTerraformIngestRole_updated"
    region                         = "us-west-2"
    collect_flow_logs              = false
    collect_firewall_logs          = true
    aws_iam_role_arn_is_org        = true
    secondary_aws_accounts         = ["111111111111"]
    secondary_aws_blocked_accounts = ["999999999999"]
    secondary_aws_regions          = ["us-east-1", "eu-west-1"]
    secondary_aws_suffix           = "KentikRole"
  }
  sampling {
    sampling_enabled = true
    sampling_type    = "CLOUD_EXPORT_SAMPLING_TYPE_RATE"
    sampling_rate    = 50
  }
}
`

func TestResourceCloudExportAWS(t *testing.T) {
	t.Parallel()
	srv, fake := newFake(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: factories(srv),
		CheckDestroy:             checkDestroyed(fake),
		Steps: []resource.TestStep{
			{
				Config: awsCreateConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(awsName, "id"),
					resource.TestCheckResourceAttr(awsName, "type", "CLOUD_EXPORT_TYPE_KENTIK_MANAGED"),
					resource.TestCheckResourceAttr(awsName, "enabled", "true"),
					resource.TestCheckResourceAttr(awsName, "name", "resource_test_terraform_aws_export"),
					resource.TestCheckResourceAttr(awsName, "description", "resource test aws export"),
					resource.TestCheckResourceAttr(awsName, "plan_id", "9948"),
					resource.TestCheckResourceAttr(awsName, "cloud_provider", "aws"),
					resource.TestCheckResourceAttr(awsName, "aws.0.bucket", "resource-terraform-aws-bucket"),
					resource.TestCheckResourceAttr(awsName, "aws.0.iam_role_arn", "arn:aws:iam::003740049406:role/trafficTerraformIngestRole"),
					resource.TestCheckResourceAttr(awsName, "aws.0.region", "eu-central-1"),
					resource.TestCheckResourceAttr(awsName, "aws.0.delete_after_read", "true"),
					// Deprecated, kept from configuration:
					resource.TestCheckResourceAttr(awsName, "aws.0.multiple_buckets", "true"),
					resource.TestCheckResourceAttr(awsName, "bgp.0.apply_bgp", "true"),
					resource.TestCheckResourceAttr(awsName, "bgp.0.use_bgp_device_id", "1234"),
					// Server defaults for unset proto3 optional fields:
					resource.TestCheckResourceAttr(awsName, "aws.0.collect_flow_logs", "true"),
					resource.TestCheckResourceAttr(awsName, "aws.0.collect_firewall_logs", "false"),
					resource.TestCheckResourceAttr(awsName, "aws.0.metadata_only", "false"),
					resource.TestCheckResourceAttr(awsName, "sampling.0.sampling_type", "CLOUD_EXPORT_SAMPLING_TYPE_UNSAMPLED"),
					resource.TestCheckResourceAttr(awsName, "current_status.0.status", "OK"),
					resource.TestCheckResourceAttrSet(awsName, "cdate"),
					resource.TestCheckResourceAttrSet(awsName, "edate"),
					checkStored(fake, awsName, func(e *cloudexport.CloudExport) error {
						aws := e.GetAws()
						if aws == nil {
							return errors.New("aws properties not sent")
						}
						if e.GetCloudProvider() != cloudexport.CloudProvider_CLOUD_PROVIDER_AWS {
							return fmt.Errorf("cloud_provider = %v", e.GetCloudProvider())
						}
						if !aws.GetDeleteAfterRead() { //nolint:staticcheck // deprecated field still supported
							return errors.New("delete_after_read not sent")
						}
						return nil
					}),
				),
			},
			{
				Config: awsUpdateConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(awsName, "type", "CLOUD_EXPORT_TYPE_CUSTOMER_MANAGED"),
					resource.TestCheckResourceAttr(awsName, "enabled", "false"),
					resource.TestCheckResourceAttr(awsName, "name", "resource_test_terraform_aws_export_updated"),
					resource.TestCheckResourceAttr(awsName, "description", "resource test aws export updated"),
					resource.TestCheckResourceAttr(awsName, "plan_id", "3333"),
					resource.TestCheckResourceAttr(awsName, "aws.0.bucket", "resource-terraform-aws-bucket-updated"),
					resource.TestCheckResourceAttr(awsName, "aws.0.bucket_prefix", "flows/"),
					resource.TestCheckResourceAttr(awsName, "aws.0.region", "us-west-2"),
					resource.TestCheckResourceAttr(awsName, "aws.0.delete_after_read", "false"),
					resource.TestCheckResourceAttr(awsName, "aws.0.collect_flow_logs", "false"),
					resource.TestCheckResourceAttr(awsName, "aws.0.collect_firewall_logs", "true"),
					resource.TestCheckResourceAttr(awsName, "aws.0.aws_iam_role_arn_is_org", "true"),
					resource.TestCheckResourceAttr(awsName, "aws.0.secondary_aws_regions.#", "2"),
					resource.TestCheckResourceAttr(awsName, "aws.0.secondary_aws_suffix", "KentikRole"),
					resource.TestCheckResourceAttr(awsName, "sampling.0.sampling_enabled", "true"),
					resource.TestCheckResourceAttr(awsName, "sampling.0.sampling_type", "CLOUD_EXPORT_SAMPLING_TYPE_RATE"),
					resource.TestCheckResourceAttr(awsName, "sampling.0.sampling_rate", "50"),
					checkStored(fake, awsName, func(e *cloudexport.CloudExport) error {
						aws := e.GetAws()
						switch {
						case aws.CollectFlowLogs == nil || aws.GetCollectFlowLogs():
							return errors.New("collect_flow_logs=false not sent")
						case aws.CollectMetrics == nil:
							return errors.New("server-computed collect_metrics not sent back on update")
						case e.GetSampling().GetSamplingRate() != 50:
							return fmt.Errorf("sampling_rate = %d", e.GetSampling().GetSamplingRate())
						case len(aws.GetSecondaryAwsBlockedAccounts()) != 1:
							return fmt.Errorf("secondary_aws_blocked_accounts = %v", aws.GetSecondaryAwsBlockedAccounts())
						}
						return nil
					}),
				),
			},
			{
				ResourceName:      awsName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestResourceCloudExportAWS_ExplicitOptionalBoolsOnCreate(t *testing.T) {
	t.Parallel()
	srv, fake := newFake(t)
	const name = "kentik-cloudexport_item.explicit"

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: factories(srv),
		CheckDestroy:             checkDestroyed(fake),
		Steps: []resource.TestStep{{
			Config: providerConfig + `
resource "kentik-cloudexport_item" "explicit" {
  name           = "explicit"
  type           = "CLOUD_EXPORT_TYPE_KENTIK_MANAGED"
  enabled        = true
  plan_id        = "9948"
  cloud_provider = "aws"
  aws {
    iam_role_arn      = "arn"
    region            = "us-east-1"
    metadata_only     = true
    collect_flow_logs = false
  }
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(name, "aws.0.metadata_only", "true"),
				resource.TestCheckResourceAttr(name, "aws.0.collect_flow_logs", "false"),
				resource.TestCheckResourceAttr(name, "aws.0.collect_metrics", "false"),
				checkStored(fake, name, func(e *cloudexport.CloudExport) error {
					aws := e.GetAws()
					if aws.MetadataOnly == nil || !aws.GetMetadataOnly() || aws.CollectFlowLogs == nil || aws.GetCollectFlowLogs() {
						return fmt.Errorf("explicit optional bools not sent on create: %v", aws)
					}
					return nil
				}),
			),
		}},
	})
}

func TestResourceCloudExportGCE(t *testing.T) {
	t.Parallel()
	srv, fake := newFake(t)
	const name = "kentik-cloudexport_item.test_gce"
	config := func(suffix string, collectMetrics bool) string {
		return providerConfig + fmt.Sprintf(`
resource "kentik-cloudexport_item" "test_gce" {
  name           = "resource_test_terraform_gce_export%[1]s"
  type           = "CLOUD_EXPORT_TYPE_KENTIK_MANAGED"
  enabled        = true
  description    = "resource test gce export%[1]s"
  plan_id        = "9948"
  cloud_provider = "gce"
  gce {
    project         = "gce project%[1]s"
    subscription    = "gce subscription%[1]s"
    collect_metrics = %[2]t
  }
}
`, suffix, collectMetrics)
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: factories(srv),
		CheckDestroy:             checkDestroyed(fake),
		Steps: []resource.TestStep{
			{
				Config: config("", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "cloud_provider", "gce"),
					resource.TestCheckResourceAttr(name, "gce.0.project", "gce project"),
					resource.TestCheckResourceAttr(name, "gce.0.subscription", "gce subscription"),
					resource.TestCheckResourceAttr(name, "gce.0.collect_flow_logs", "true"),
					resource.TestCheckResourceAttr(name, "gce.0.collect_metrics", "false"),
					checkStored(fake, name, func(e *cloudexport.CloudExport) error {
						if !e.GetGce().GetCollectFlowLogs() {
							return errors.New("gce collect_flow_logs must default to true, as in v0.4")
						}
						return nil
					}),
				),
			},
			{
				Config: config(" updated", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "resource_test_terraform_gce_export updated"),
					resource.TestCheckResourceAttr(name, "gce.0.project", "gce project updated"),
					resource.TestCheckResourceAttr(name, "gce.0.collect_metrics", "true"),
				),
			},
			{ResourceName: name, ImportState: true, ImportStateVerify: true},
		},
	})
}

func TestResourceCloudExportAzure(t *testing.T) {
	t.Parallel()
	srv, fake := newFake(t)
	const name = "kentik-cloudexport_item.test_azure"
	config := func(suffix string, principal bool) string {
		return providerConfig + fmt.Sprintf(`
resource "kentik-cloudexport_item" "test_azure" {
  name           = "resource_test_terraform_azure_export%[1]s"
  type           = "CLOUD_EXPORT_TYPE_KENTIK_MANAGED"
  enabled        = true
  plan_id        = "9948"
  cloud_provider = "azure"
  azure {
    location                   = "centralus%[1]s"
    resource_group             = "traffic-generator%[1]s"
    storage_account            = "kentikstorage%[1]s"
    subscription_id            = "7777%[1]s"
    security_principal_enabled = %[2]t
  }
}
`, suffix, principal)
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: factories(srv),
		CheckDestroy:             checkDestroyed(fake),
		Steps: []resource.TestStep{
			{
				Config: config("", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "cloud_provider", "azure"),
					resource.TestCheckResourceAttr(name, "description", ""),
					resource.TestCheckResourceAttr(name, "azure.0.location", "centralus"),
					resource.TestCheckResourceAttr(name, "azure.0.resource_group", "traffic-generator"),
					resource.TestCheckResourceAttr(name, "azure.0.storage_account", "kentikstorage"),
					resource.TestCheckResourceAttr(name, "azure.0.subscription_id", "7777"),
					resource.TestCheckResourceAttr(name, "azure.0.security_principal_enabled", "true"),
				),
			},
			{
				Config: config("-updated", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "azure.0.location", "centralus-updated"),
					resource.TestCheckResourceAttr(name, "azure.0.security_principal_enabled", "false"),
				),
			},
			{ResourceName: name, ImportState: true, ImportStateVerify: true},
		},
	})
}

func TestResourceCloudExportOCI(t *testing.T) {
	t.Parallel()
	srv, fake := newFake(t)
	const name = "kentik-cloudexport_item.test_oci"
	config := func(region string, compartments string) string {
		return providerConfig + fmt.Sprintf(`
resource "kentik-cloudexport_item" "test_oci" {
  name           = "resource_test_terraform_oci_export"
  type           = "CLOUD_EXPORT_TYPE_KENTIK_MANAGED"
  enabled        = true
  plan_id        = "9948"
  cloud_provider = "oci"
  oci {
    oci_tenancy_id              = "ocid1.tenancy.oc1..aaaa"
    oci_default_region          = %q
    oci_user_id                 = "ocid1.user.oc1..cccc"
    oci_compartment_id          = %s
    oci_bucket_name             = "flow-logs"
    oci_bucket_namespace_name   = "ns"
    oci_service_connector_ocid  = "ocid1.serviceconnector.oc1..dddd"
    oci_flow_object_name_prefix = "prefix"
  }
}
`, region, compartments)
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: factories(srv),
		CheckDestroy:             checkDestroyed(fake),
		Steps: []resource.TestStep{
			{
				Config: config("eu-frankfurt-1", `["ocid1.compartment.oc1..bbbb"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "cloud_provider", "oci"),
					resource.TestCheckResourceAttr(name, "oci.0.oci_default_region", "eu-frankfurt-1"),
					resource.TestCheckResourceAttr(name, "oci.0.oci_collect_flow_logs", "true"),
					resource.TestCheckResourceAttr(name, "oci.0.metadata_only", "false"),
					resource.TestCheckResourceAttr(name, "oci.0.oci_compartment_id.#", "1"),
					checkStored(fake, name, func(e *cloudexport.CloudExport) error {
						if e.GetOci().GetOciServiceConnectorOcid() != "ocid1.serviceconnector.oc1..dddd" {
							return errors.New("oci_service_connector_ocid not sent")
						}
						return nil
					}),
				),
			},
			{
				Config: config("us-ashburn-1", `["a", "b"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "oci.0.oci_default_region", "us-ashburn-1"),
					resource.TestCheckResourceAttr(name, "oci.0.oci_compartment_id.#", "2"),
				),
			},
			{ResourceName: name, ImportState: true, ImportStateVerify: true},
		},
	})
}

func TestResourceCloudExport_IBMIsRejectedAtPlan(t *testing.T) {
	t.Parallel()
	srv, fake := newFake(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: factories(srv),
		Steps: []resource.TestStep{{
			Config: providerConfig + `
resource "kentik-cloudexport_item" "test_ibm" {
  name           = "resource_test_terraform_ibm_export"
  type           = "CLOUD_EXPORT_TYPE_KENTIK_MANAGED"
  enabled        = true
  plan_id        = "9948"
  cloud_provider = "ibm"
  ibm {
    bucket = "ibm-bucket"
  }
}
`,
			ExpectError: regexp.MustCompile(`expected cloud_provider to be one of`),
		}},
	})
	assert.Equal(t, 0, fake.Len())
}

func TestResourceCloudExport_ProviderBlockMustMatch(t *testing.T) {
	t.Parallel()
	srv, _ := newFake(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: factories(srv),
		Steps: []resource.TestStep{{
			Config: providerConfig + `
resource "kentik-cloudexport_item" "mismatch" {
  name           = "mismatch"
  type           = "CLOUD_EXPORT_TYPE_KENTIK_MANAGED"
  enabled        = true
  plan_id        = "9948"
  cloud_provider = "aws"
  gce {
    project = "p"
  }
}
`,
			ExpectError: regexp.MustCompile(`a aws \{\.\.\.\} block is required`),
		}},
	})
}

func TestResourceCloudExport_DuplicateNameFails(t *testing.T) {
	t.Parallel()
	srv, _ := newFake(t, seedExports()...)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: factories(srv),
		Steps: []resource.TestStep{{
			Config: providerConfig + `
resource "kentik-cloudexport_item" "dup" {
  name           = "test_terraform_aws_export"
  type           = "CLOUD_EXPORT_TYPE_KENTIK_MANAGED"
  enabled        = true
  plan_id        = "9948"
  cloud_provider = "aws"
  aws {
    iam_role_arn = "arn"
    region       = "us-east-1"
  }
}
`,
			ExpectError: regexp.MustCompile(`(?s)Failed to create cloud export.*AlreadyExists`),
		}},
	})
}

// Deleting the export outside Terraform must lead to a plan that recreates it.
func TestResourceCloudExport_DeletedOutsideTerraform(t *testing.T) {
	t.Parallel()
	srv, fake := newFake(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: factories(srv),
		Steps: []resource.TestStep{
			{Config: awsCreateConfig},
			{
				PreConfig: func() {
					for _, e := range listAll(t, fake) {
						_, err := fake.DeleteCloudExport(context.Background(), &cloudexport.DeleteCloudExportRequest{Id: e.GetId()})
						require.NoError(t, err)
					}
				},
				Config:             awsCreateConfig,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func listAll(t *testing.T, fake *kentikapitest.CloudExportServer) []*cloudexport.CloudExport {
	t.Helper()
	resp, err := fake.ListCloudExports(context.Background(), &cloudexport.ListCloudExportsRequest{})
	require.NoError(t, err)
	return resp.GetExports()
}
