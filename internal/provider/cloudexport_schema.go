package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	cloudexport "github.com/kentik/api-schema-public/gen/go/kentik/cloud_export/v202506"
)

// Schema of a cloud export, mapped from kentik.cloud_export.v202506.CloudExport.
// See docs/UPGRADING.md for the mapping from the v202101beta1-based provider.

const (
	awsKey      = "aws"
	azureKey    = "azure"
	gceKey      = "gce"
	ociKey      = "oci"
	ibmKey      = "ibm"
	bgpKey      = "bgp"
	samplingKey = "sampling"
	statusKey   = "current_status"

	multipleBucketsKey = "multiple_buckets"

	deprecatedIBM = "IBM Cloud is not supported by the Kentik cloud_export v202506 API. " +
		"This block is ignored and will be removed in the next major version."
	deprecatedBGP = "BGP settings are not part of the Kentik cloud_export v202506 API. " +
		"This block is ignored and will be removed in the next major version."
	deprecatedMultipleBuckets = "multiple_buckets is not part of the Kentik cloud_export v202506 API. " +
		"It is ignored and will be removed in the next major version."
	deprecatedDeleteAfterRead = "delete_after_read is deprecated by the Kentik cloud_export v202506 API."
)

func cloudProviderKeys() []string { return []string{awsKey, azureKey, gceKey, ociKey, ibmKey} }

func makeCloudExportSchema(mode schemaMode) map[string]*schema.Schema {
	m := makeCloudExportAttributes(mode)
	m["id"] = field(mode, computedOnly, &schema.Schema{
		Type:        schema.TypeString,
		Description: "Unique identifier of the instance. Assigned by Kentik.",
	})
	if mode == readSingle {
		// The single-item data source looks the export up by ID.
		m["id"] = &schema.Schema{
			Type:        schema.TypeString,
			Required:    true,
			Description: "Unique identifier of the cloud export to read.",
		}
	}
	return m
}

func makeCloudExportAttributes(mode schemaMode) map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"type": field(mode, required, &schema.Schema{
			Type: schema.TypeString,
			Description: "Ownership of the cloud export process. One of `CLOUD_EXPORT_TYPE_KENTIK_MANAGED` " +
				"(the export process is managed by Kentik), `CLOUD_EXPORT_TYPE_CUSTOMER_MANAGED` (managed by the customer, " +
				"e.g. by running an agent) or `CLOUD_EXPORT_TYPE_UNSPECIFIED`.",
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(enumNames(cloudexport.CloudExportType_name), false)),
		}),
		"enabled": field(mode, required, &schema.Schema{
			Type:        schema.TypeBool,
			Description: "Whether the instance is active.",
		}),
		"name": field(mode, required, &schema.Schema{
			Type:        schema.TypeString,
			Description: "User selected name of the instance.",
		}),
		"description": field(mode, optional, &schema.Schema{
			Type:        schema.TypeString,
			Description: "Description of the instance.",
		}),
		"plan_id": field(mode, required, &schema.Schema{
			Type:        schema.TypeString,
			Description: "Identifier of the billing plan for the instance.",
		}),
		"cloud_provider": field(mode, required, &schema.Schema{
			Type: schema.TypeString,
			Description: "Cloud provider: `aws`, `azure`, `gce` or `oci`. The block of the same name holds the " +
				"provider-specific properties. (`ibm` is no longer supported by the API.)",
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(supportedCloudProviders(), false)),
		}),
		awsKey:      makeAWSSchema(mode),
		azureKey:    makeAzureSchema(mode),
		gceKey:      makeGCESchema(mode),
		ociKey:      makeOCISchema(mode),
		ibmKey:      makeIBMSchema(mode),
		bgpKey:      makeBGPSchema(mode),
		samplingKey: makeSamplingSchema(mode),
		statusKey:   makeCurrentStatusSchema(mode),
		"cdate": field(mode, computedOnly, &schema.Schema{
			Type:        schema.TypeString,
			Description: "Creation timestamp (UTC, RFC 3339).",
		}),
		"edate": field(mode, computedOnly, &schema.Schema{
			Type:        schema.TypeString,
			Description: "Last modification timestamp (UTC, RFC 3339).",
		}),
	}
}

func providerBlock(mode schemaMode, description string, attrs map[string]*schema.Schema) *schema.Schema {
	s := block(mode, optional, description, attrs)
	if mode == create {
		s.ExactlyOneOf = cloudProviderKeys()
	}
	return s
}

func makeAWSSchema(mode schemaMode) *schema.Schema {
	return providerBlock(mode, "Properties specific to Amazon Web Services (VPC flow logs) exports.", map[string]*schema.Schema{
		"iam_role_arn": field(mode, required, &schema.Schema{
			Type:        schema.TypeString,
			Description: "ARN of the IAM role granted access to the S3 bucket and describe API end-points.",
		}),
		"region": field(mode, required, &schema.Schema{
			Type:        schema.TypeString,
			Description: "Name of AWS region from which to export flow logs.",
		}),
		"bucket": field(mode, optional, &schema.Schema{
			Type:        schema.TypeString,
			Description: "Name of S3 bucket from which flow and/or firewall logs are to be exported.",
		}),
		"bucket_prefix": field(mode, optional, &schema.Schema{
			Type:        schema.TypeString,
			Description: "Prefix of S3 bucket from which flow and/or firewall logs are to be exported.",
		}),
		"metadata_only": field(mode, optionalComputed, &schema.Schema{
			Type:        schema.TypeBool,
			Description: "Import only metadata without any flows. The API defaults to `false`.",
		}),
		"collect_flow_logs": field(mode, optionalComputed, &schema.Schema{
			Type:        schema.TypeBool,
			Description: "Import metadata and collect flow logs. The API defaults to `true`.",
		}),
		"collect_firewall_logs": field(mode, optionalComputed, &schema.Schema{
			Type: schema.TypeBool,
			Description: "Import metadata and collect firewall logs in addition to flow logs. The API defaults to `false`. " +
				"To collect firewall logs without flow logs, set `collect_flow_logs = false`.",
		}),
		"collect_metrics": field(mode, optionalComputed, &schema.Schema{
			Type: schema.TypeBool,
			Description: "Import metadata and collect metrics data. Flow logs are still collected unless " +
				"`metadata_only` is also `true`. The API defaults to `false`.",
		}),
		"delete_after_read": field(mode, optional, &schema.Schema{
			Type:        schema.TypeBool,
			Default:     false,
			Deprecated:  deprecatedDeleteAfterRead,
			Description: "Delete logs from the S3 bucket after export. Deprecated by the API.",
		}),
		"aws_iam_role_arn_is_org": field(mode, optional, &schema.Schema{
			Type:        schema.TypeBool,
			Default:     false,
			Description: "Whether the IAM role is an organization role.",
		}),
		"secondary_aws_accounts": field(mode, optional, &schema.Schema{
			Type:        schema.TypeList,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "Account IDs granted access to the describe API end-points.",
		}),
		"secondary_aws_blocked_accounts": field(mode, optional, &schema.Schema{
			Type:        schema.TypeList,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "Account IDs to filter out of the organization when calling the describe API end-points.",
		}),
		"secondary_aws_regions": field(mode, optional, &schema.Schema{
			Type:        schema.TypeList,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "Regions of secondary accounts from which to collect metadata.",
		}),
		"secondary_aws_suffix": field(mode, optional, &schema.Schema{
			Type: schema.TypeString,
			Description: "Role suffix used to build secondary account role ARNs from the template " +
				"`arn:aws:iam::<aws_account_id>:role/<role_suffix>`.",
		}),
		multipleBucketsKey: field(mode, optional, &schema.Schema{
			Type:        schema.TypeBool,
			Deprecated:  deprecatedMultipleBuckets,
			Description: "Ignored: not part of the cloud_export v202506 API. Kept so existing configurations keep working.",
		}),
	})
}

func makeAzureSchema(mode schemaMode) *schema.Schema {
	return providerBlock(mode, "Properties specific to Microsoft Azure (NSG flow logs) exports.", map[string]*schema.Schema{
		"location": field(mode, required, &schema.Schema{
			Type:        schema.TypeString,
			Description: "Azure region/location from which to export flow logs.",
		}),
		"resource_group": field(mode, required, &schema.Schema{
			Type:        schema.TypeString,
			Description: "Resource group containing the NSG generating flow logs.",
		}),
		"storage_account": field(mode, required, &schema.Schema{
			Type:        schema.TypeString,
			Description: "Storage account from which flow logs are to be extracted.",
		}),
		"subscription_id": field(mode, required, &schema.Schema{
			Type:        schema.TypeString,
			Description: "ID of Azure account from which flows logs are to be exported.",
		}),
		"security_principal_enabled": field(mode, optional, &schema.Schema{
			Type:        schema.TypeBool,
			Default:     false,
			Description: "Whether the security principal for the Kentik flow export application has been authorized.",
		}),
	})
}

func makeGCESchema(mode schemaMode) *schema.Schema {
	return providerBlock(mode, "Properties specific to Google Cloud exports.", map[string]*schema.Schema{
		"project": field(mode, required, &schema.Schema{
			Type:        schema.TypeString,
			Description: "Name of the project from which to export flow logs.",
		}),
		"subscription": field(mode, optional, &schema.Schema{
			Type:        schema.TypeString,
			Description: "GCP Pub/Sub subscription providing flow logs.",
		}),
		"metadata_only": field(mode, optional, &schema.Schema{
			Type:        schema.TypeBool,
			Default:     false,
			Description: "Import only metadata without any flows.",
		}),
		"collect_flow_logs": field(mode, optional, &schema.Schema{
			Type:        schema.TypeBool,
			Default:     true,
			Description: "Import metadata and collect flow logs.",
		}),
		"collect_metrics": field(mode, optional, &schema.Schema{
			Type: schema.TypeBool,
			Description: "Import metadata and collect metrics data. Flow logs are still collected unless " +
				"`metadata_only` is also `true`.",
			Default: false,
		}),
	})
}

func makeOCISchema(mode schemaMode) *schema.Schema {
	return providerBlock(mode, "Properties specific to Oracle Cloud Infrastructure exports.", map[string]*schema.Schema{
		"oci_tenancy_id": field(mode, required, &schema.Schema{
			Type:        schema.TypeString,
			Description: "OCI tenancy OCID.",
		}),
		"oci_default_region": field(mode, required, &schema.Schema{
			Type:        schema.TypeString,
			Description: "Default enabled region from which to collect metadata.",
		}),
		"oci_user_id": field(mode, optional, &schema.Schema{
			Type:        schema.TypeString,
			Description: "ID of the user created to represent the Kentik cloud export tool.",
		}),
		"oci_compartment_id": field(mode, optional, &schema.Schema{
			Type:        schema.TypeList,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "Compartment IDs from which to collect metadata.",
		}),
		"oci_collect_flow_logs": field(mode, optional, &schema.Schema{
			Type:        schema.TypeBool,
			Default:     true,
			Description: "Import metadata and collect flow logs.",
		}),
		"oci_bucket_name": field(mode, optional, &schema.Schema{
			Type:        schema.TypeString,
			Description: "Name of the bucket from which flow logs are extracted. Required when collecting flow logs.",
		}),
		"oci_bucket_namespace_name": field(mode, optional, &schema.Schema{
			Type:        schema.TypeString,
			Description: "Namespace of the bucket from which flow logs are extracted. Required when collecting flow logs.",
		}),
		"oci_service_connector_ocid": field(mode, optional, &schema.Schema{
			Type:        schema.TypeString,
			Description: "OCID of the OCI service connector that delivers flow logs to the bucket.",
		}),
		"oci_flow_object_name_prefix": field(mode, optional, &schema.Schema{
			Type: schema.TypeString,
			Description: "Object name prefix: Kentik looks for files in the bucket whose names start with " +
				"`<prefix>/<service_connector_ocid>`.",
		}),
		"metadata_only": field(mode, optional, &schema.Schema{
			Type:        schema.TypeBool,
			Default:     false,
			Description: "Import only metadata without any flows.",
		}),
	})
}

func makeIBMSchema(mode schemaMode) *schema.Schema {
	s := providerBlock(mode, "Deprecated: IBM Cloud is not supported by the cloud_export v202506 API.", map[string]*schema.Schema{
		"bucket": field(mode, required, &schema.Schema{Type: schema.TypeString}),
	})
	if mode == create {
		s.Deprecated = deprecatedIBM
	}
	return s
}

func makeBGPSchema(mode schemaMode) *schema.Schema {
	s := block(mode, optional, "Deprecated: BGP settings are not part of the cloud_export v202506 API and are ignored.",
		map[string]*schema.Schema{
			"apply_bgp":         field(mode, optional, &schema.Schema{Type: schema.TypeBool}),
			"use_bgp_device_id": field(mode, optional, &schema.Schema{Type: schema.TypeString}),
			"device_bgp_type":   field(mode, optional, &schema.Schema{Type: schema.TypeString}),
		})
	if mode == create {
		s.Deprecated = deprecatedBGP
	}
	return s
}

func makeSamplingSchema(mode schemaMode) *schema.Schema {
	return block(mode, optionalComputed, "Sampling settings for the export.", map[string]*schema.Schema{
		"sampling_enabled": field(mode, optional, &schema.Schema{
			Type:        schema.TypeBool,
			Default:     false,
			Description: "Whether sampling is used for this export.",
		}),
		"sampling_type": field(mode, optionalComputed, &schema.Schema{
			Type: schema.TypeString,
			Description: "`CLOUD_EXPORT_SAMPLING_TYPE_UNSAMPLED` exports all flows; `CLOUD_EXPORT_SAMPLING_TYPE_RATE` " +
				"exports sampled flows and requires `sampling_rate` greater than 0 and less than 2000.",
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(
				enumNames(cloudexport.CloudExportSamplingType_name), false)),
		}),
		"sampling_rate": field(mode, optional, &schema.Schema{
			Type:             schema.TypeInt,
			Description:      "Sampling rate for the export.",
			ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(0, 1999)),
		}),
	})
}

func makeCurrentStatusSchema(mode schemaMode) *schema.Schema {
	return block(mode, computedOnly, "Runtime status of the export process.", map[string]*schema.Schema{
		"status": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Status of the export task.",
		},
		"error_message": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Text of the last error message (empty if status is OK).",
		},
		"flow_found": {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "Whether any flow data were exported.",
		},
		"api_access": {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "Whether the export process is able to access the cloud API.",
		},
		"storage_account_access": {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "Whether the export process is able to access the storage account containing flow logs.",
		},
	})
}
