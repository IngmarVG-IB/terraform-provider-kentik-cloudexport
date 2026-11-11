package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestDataSourceCloudExportList(t *testing.T) {
	t.Parallel()
	srv, _ := newFake(t, seedExports()...)

	const list = "data.kentik-cloudexport_list.exports"
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: factories(srv),
		Steps: []resource.TestStep{{
			Config: providerConfig + `
data "kentik-cloudexport_list" "exports" {}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet(list, "id"),
				resource.TestCheckResourceAttr(list, "items.#", "4"),
				resource.TestCheckResourceAttr(list, "invalid_exports_count", "0"),
				resource.TestCheckResourceAttr(list, "items.0.id", "1"),
				resource.TestCheckResourceAttr(list, "items.0.name", "test_terraform_aws_export"),
				resource.TestCheckResourceAttr(list, "items.0.cloud_provider", "aws"),
				resource.TestCheckResourceAttr(list, "items.0.aws.0.bucket", "terraform-aws-bucket"),
				resource.TestCheckResourceAttr(list, "items.1.id", "2"),
				resource.TestCheckResourceAttr(list, "items.1.cloud_provider", "gce"),
				resource.TestCheckResourceAttr(list, "items.1.gce.0.project", "project gce"),
				resource.TestCheckResourceAttr(list, "items.2.cloud_provider", "oci"),
				resource.TestCheckResourceAttr(list, "items.2.oci.0.oci_tenancy_id", "ocid1.tenancy.oc1..aaaa"),
				resource.TestCheckResourceAttr(list, "items.3.cloud_provider", "azure"),
				resource.TestCheckResourceAttr(list, "items.3.azure.0.location", "centralus"),
			),
		}},
	})
}

func TestDataSourceCloudExportList_Empty(t *testing.T) {
	t.Parallel()
	srv, _ := newFake(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: factories(srv),
		Steps: []resource.TestStep{{
			Config: providerConfig + `
data "kentik-cloudexport_list" "exports" {}
`,
			Check: resource.TestCheckResourceAttr("data.kentik-cloudexport_list.exports", "items.#", "0"),
		}},
	})
}
