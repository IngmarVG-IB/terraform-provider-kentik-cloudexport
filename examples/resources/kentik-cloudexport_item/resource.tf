# AWS: VPC flow logs from an S3 bucket, plus metadata from secondary accounts.
resource "kentik-cloudexport_item" "aws" {
  name           = "example_aws_export"
  type           = "CLOUD_EXPORT_TYPE_KENTIK_MANAGED"
  enabled        = true
  description    = "AWS VPC flow logs"
  plan_id        = "11467"
  cloud_provider = "aws"

  aws {
    bucket                 = "example-vpc-flow-logs"
    bucket_prefix          = "AWSLogs/"
    iam_role_arn           = "arn:aws:iam::003740049406:role/KentikIngestRole"
    region                 = "us-east-2"
    collect_firewall_logs  = true
    secondary_aws_accounts = ["111111111111", "222222222222"]
    secondary_aws_regions  = ["us-east-1", "eu-west-1"]
    secondary_aws_suffix   = "KentikMetadataRole"
  }

  sampling {
    sampling_enabled = true
    sampling_type    = "CLOUD_EXPORT_SAMPLING_TYPE_RATE"
    sampling_rate    = 100
  }
}

# Azure: NSG flow logs from a storage account.
resource "kentik-cloudexport_item" "azure" {
  name           = "example_azure_export"
  type           = "CLOUD_EXPORT_TYPE_KENTIK_MANAGED"
  enabled        = true
  plan_id        = "11467"
  cloud_provider = "azure"

  azure {
    location                   = "centralus"
    resource_group             = "traffic-generator"
    storage_account            = "kentikstorage"
    subscription_id            = "784bd5ec-122b-41b7-9719-22f23d5b49c8"
    security_principal_enabled = true
  }
}

# Google Cloud: VPC flow logs from a Pub/Sub subscription.
resource "kentik-cloudexport_item" "gce" {
  name           = "example_gce_export"
  type           = "CLOUD_EXPORT_TYPE_KENTIK_MANAGED"
  enabled        = true
  plan_id        = "11467"
  cloud_provider = "gce"

  gce {
    project         = "example-project"
    subscription    = "kentik-flow-logs"
    collect_metrics = true
  }
}

# Oracle Cloud Infrastructure: flow logs from an object storage bucket.
resource "kentik-cloudexport_item" "oci" {
  name           = "example_oci_export"
  type           = "CLOUD_EXPORT_TYPE_KENTIK_MANAGED"
  enabled        = true
  plan_id        = "11467"
  cloud_provider = "oci"

  oci {
    oci_tenancy_id             = "ocid1.tenancy.oc1..exampleuniqueID"
    oci_default_region         = "eu-frankfurt-1"
    oci_user_id                = "ocid1.user.oc1..exampleuniqueID"
    oci_compartment_id         = ["ocid1.compartment.oc1..exampleuniqueID"]
    oci_bucket_name            = "flow-logs"
    oci_bucket_namespace_name  = "examplenamespace"
    oci_service_connector_ocid = "ocid1.serviceconnector.oc1..exampleuniqueID"
  }
}

output "aws_export_id" {
  value = kentik-cloudexport_item.aws.id
}
