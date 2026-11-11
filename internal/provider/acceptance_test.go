package provider_test

// Acceptance tests run against a live Kentik tenant. resource.Test skips them
// unless TF_ACC=1; PreCheck then requires KTAPI_AUTH_EMAIL, KTAPI_AUTH_TOKEN and
// KENTIK_PLAN_ID (KTAPI_URL, KTAPI_ENDPOINT or KTAPI_REGION select the tenant
// region). Leftovers named with the test prefix are removed by `go test -sweep`.

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	cloudexport "github.com/kentik/api-schema-public/gen/go/kentik/cloud_export/v202506"

	"github.com/kentik/terraform-provider-kentik-cloudexport/internal/kentikapi"
	"github.com/kentik/terraform-provider-kentik-cloudexport/internal/provider"
)

const accSweeper = "kentik_cloudexport"

//nolint:gochecknoinits // test sweepers must register in init
func init() {
	resource.AddTestSweepers(accSweeper, &resource.Sweeper{Name: accSweeper, F: sweepCloudExports})
}

func accPrefix() string {
	return "kentik_tf_test_" + os.Getenv("TF_ACC_PREFIX")
}

func accPreCheck(t *testing.T) {
	t.Helper()
	for _, k := range []string{"KTAPI_AUTH_EMAIL", "KTAPI_AUTH_TOKEN", "KENTIK_PLAN_ID"} {
		if os.Getenv(k) == "" {
			t.Fatalf("%s must be set for acceptance tests", k)
		}
	}
}

func accFactories() map[string]func() (tfprotov5.ProviderServer, error) {
	return map[string]func() (tfprotov5.ProviderServer, error){
		"kentik-cloudexport": func() (tfprotov5.ProviderServer, error) { return provider.New().GRPCProvider(), nil },
	}
}

func TestAccResourceCloudExportAWS(t *testing.T) {
	name := accPrefix() + "aws"
	const res = "kentik-cloudexport_item.acc"
	config := func(description string) string {
		return fmt.Sprintf(`
provider "kentik-cloudexport" {}

resource "kentik-cloudexport_item" "acc" {
  name           = %q
  type           = "CLOUD_EXPORT_TYPE_KENTIK_MANAGED"
  enabled        = false
  description    = %q
  plan_id        = %q
  cloud_provider = "aws"
  aws {
    bucket        = "%s-bucket"
    iam_role_arn  = "arn:aws:iam::000000000000:role/kentik-tf-acc"
    region        = "us-east-2"
    metadata_only = true
  }
}

data "kentik-cloudexport_item" "acc" {
  id = kentik-cloudexport_item.acc.id
}
`, name, description, os.Getenv("KENTIK_PLAN_ID"), strings.ReplaceAll(name, "_", "-"))
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV5ProviderFactories: accFactories(),
		Steps: []resource.TestStep{
			{
				Config: config("created by terraform acceptance test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(res, "id"),
					resource.TestCheckResourceAttr(res, "name", name),
					resource.TestCheckResourceAttrPair("data.kentik-cloudexport_item.acc", "name", res, "name"),
				),
			},
			{
				Config: config("updated by terraform acceptance test"),
				Check:  resource.TestCheckResourceAttr(res, "description", "updated by terraform acceptance test"),
			},
			{ResourceName: res, ImportState: true, ImportStateVerify: true},
		},
	})
}

func sweepCloudExports(string) error {
	ctx := context.Background()
	cfg := kentikapi.Config{
		Email:  os.Getenv("KTAPI_AUTH_EMAIL"),
		Token:  os.Getenv("KTAPI_AUTH_TOKEN"),
		Region: kentikapi.Region(strings.ToUpper(os.Getenv("KTAPI_REGION"))),
	}
	switch {
	case os.Getenv("KTAPI_ENDPOINT") != "":
		cfg.Endpoint = os.Getenv("KTAPI_ENDPOINT")
	case os.Getenv("KTAPI_URL") != "":
		var err error
		if cfg.Endpoint, cfg.Insecure, err = kentikapi.EndpointFromAPIURL(os.Getenv("KTAPI_URL")); err != nil {
			return err
		}
	}
	client, err := kentikapi.New(ctx, cfg)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}
	defer func() { _ = client.Close() }()

	resp, err := client.CloudExport.ListCloudExports(ctx, &cloudexport.ListCloudExportsRequest{})
	if err != nil {
		return fmt.Errorf("list cloud exports: %w", err)
	}
	for _, e := range resp.GetExports() {
		if !strings.HasPrefix(e.GetName(), accPrefix()) {
			continue
		}
		if _, err := client.CloudExport.DeleteCloudExport(ctx, &cloudexport.DeleteCloudExportRequest{Id: e.GetId()}); err != nil {
			log.Printf("sweep: delete %s (%s): %v", e.GetName(), e.GetId(), err)
		}
	}
	return nil
}
