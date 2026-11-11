package provider_test

import (
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	cloudexport "github.com/kentik/api-schema-public/gen/go/kentik/cloud_export/v202506"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kentik/terraform-provider-kentik-cloudexport/internal/kentikapi"
	"github.com/kentik/terraform-provider-kentik-cloudexport/internal/kentikapi/kentikapitest"
	"github.com/kentik/terraform-provider-kentik-cloudexport/internal/provider"
)

func TestMain(m *testing.M) {
	resource.TestMain(m) // enables -sweep for acceptance test leftovers
}

// providerConfig configures the provider with dummy credentials. The fake
// server is wired in through provider.NewWithClientConfig, not through HCL.
const providerConfig = `
provider "kentik-cloudexport" {
  email = "joe.doe@example.com"
  token = "dummy-token"
}
`

// newFake starts an in-process CloudExportAdminService seeded with exports.
func newFake(t *testing.T, exports ...*cloudexport.CloudExport) (*kentikapitest.Server, *kentikapitest.CloudExportServer) {
	t.Helper()
	srv := kentikapitest.NewServer(t)
	fake := kentikapitest.NewCloudExportServer(exports...).Register(srv)
	srv.Start()
	return srv, fake
}

func mustTime(s string) time.Time {
	ts, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return ts
}

// factories returns provider factories whose API client talks to srv.
func factories(srv *kentikapitest.Server) map[string]func() (tfprotov5.ProviderServer, error) {
	return map[string]func() (tfprotov5.ProviderServer, error){
		"kentik-cloudexport": func() (tfprotov5.ProviderServer, error) {
			p := provider.NewWithClientConfig(func(cfg *kentikapi.Config) {
				srv.Configure(cfg)
				cfg.Retry = kentikapi.RetryConfig{MaxAttempts: 1}
			})
			return p.GRPCProvider(), nil
		},
	}
}

func seedExports() []*cloudexport.CloudExport {
	ts := timestamppb.New(mustTime("2025-07-01T10:00:00Z"))
	okStatus := &cloudexport.CloudExportStatus{
		Status: "OK", ErrorMessage: "No errors", FlowFound: true, ApiAccess: true, StorageAccountAccess: true,
	}
	return []*cloudexport.CloudExport{
		{
			Id:            "1",
			Type:          cloudexport.CloudExportType_CLOUD_EXPORT_TYPE_KENTIK_MANAGED,
			Enabled:       true,
			Name:          "test_terraform_aws_export",
			Description:   "terraform aws cloud export",
			PlanId:        "11467",
			CloudProvider: cloudexport.CloudProvider_CLOUD_PROVIDER_AWS,
			CloudSpecificProperties: &cloudexport.CloudExport_Aws{Aws: &cloudexport.AwsProperties{
				Bucket:               "terraform-aws-bucket",
				IamRoleArn:           "arn:aws:iam::003740049406:role/trafficTerraformIngestRole",
				Region:               "us-east-2",
				CollectFlowLogs:      proto.Bool(true),
				CollectMetrics:       proto.Bool(true),
				SecondaryAwsAccounts: []string{"111111111111", "222222222222"},
			}},
			Sampling: &cloudexport.CloudExportSamplingProperties{
				SamplingEnabled: true,
				SamplingType:    cloudexport.CloudExportSamplingType_CLOUD_EXPORT_SAMPLING_TYPE_RATE,
				SamplingRate:    100,
			},
			CurrentStatus: okStatus,
			Cdate:         ts,
			Edate:         ts,
		},
		{
			Id:            "2",
			Type:          cloudexport.CloudExportType_CLOUD_EXPORT_TYPE_CUSTOMER_MANAGED,
			Enabled:       true,
			Name:          "test_terraform_gce_export",
			Description:   "terraform gce cloud export",
			PlanId:        "21600",
			CloudProvider: cloudexport.CloudProvider_CLOUD_PROVIDER_GCE,
			CloudSpecificProperties: &cloudexport.CloudExport_Gce{Gce: &cloudexport.GceProperties{
				Project:         "project gce",
				Subscription:    "subscription gce",
				CollectFlowLogs: true,
			}},
			CurrentStatus: &cloudexport.CloudExportStatus{Status: "NOK", ErrorMessage: "Timeout"},
		},
		{
			Id:            "3",
			Type:          cloudexport.CloudExportType_CLOUD_EXPORT_TYPE_KENTIK_MANAGED,
			Enabled:       false,
			Name:          "test_terraform_oci_export",
			Description:   "terraform oci cloud export",
			PlanId:        "11467",
			CloudProvider: cloudexport.CloudProvider_CLOUD_PROVIDER_OCI,
			CloudSpecificProperties: &cloudexport.CloudExport_Oci{Oci: &cloudexport.OciProperties{
				OciTenancyId:       "ocid1.tenancy.oc1..aaaa",
				OciDefaultRegion:   "eu-frankfurt-1",
				OciCompartmentId:   []string{"ocid1.compartment.oc1..bbbb"},
				OciCollectFlowLogs: true,
				OciBucketName:      "flow-logs",
			}},
			CurrentStatus: okStatus,
		},
		{
			Id:            "4",
			Type:          cloudexport.CloudExportType_CLOUD_EXPORT_TYPE_KENTIK_MANAGED,
			Enabled:       true,
			Name:          "test_terraform_azure_export",
			Description:   "terraform azure cloud export",
			PlanId:        "11467",
			CloudProvider: cloudexport.CloudProvider_CLOUD_PROVIDER_AZURE,
			CloudSpecificProperties: &cloudexport.CloudExport_Azure{Azure: &cloudexport.AzureProperties{
				Location:                 "centralus",
				ResourceGroup:            "traffic-generator",
				StorageAccount:           "kentikstorage",
				SubscriptionId:           "784bd5ec-122b-41b7-9719-22f23d5b49c8",
				SecurityPrincipalEnabled: true,
			}},
			CurrentStatus: okStatus,
		},
	}
}
