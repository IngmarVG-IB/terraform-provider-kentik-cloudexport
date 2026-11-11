package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	cloudexport "github.com/kentik/api-schema-public/gen/go/kentik/cloud_export/v202506"
)

func dataSourceCloudExportItem() *schema.Resource {
	return &schema.Resource{
		Description: "Reads a single Kentik cloud export by ID (Kentik API `cloud_export/v202506`).",
		ReadContext: dataSourceCloudExportItemRead,
		Schema:      makeCloudExportSchema(readSingle),
	}
}

func dataSourceCloudExportItemRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	id := d.Get("id").(string)
	resp, err := clientFrom(m).CloudExport.GetCloudExport(ctx, &cloudexport.GetCloudExportRequest{Id: id})
	if err != nil {
		return detailedDiagError("Failed to read cloud export item", err)
	}
	if diags := setAttributes(d, cloudExportToMap(resp.GetExport(), nil)); diags.HasError() {
		return diags
	}
	d.SetId(resp.GetExport().GetId())
	return nil
}
