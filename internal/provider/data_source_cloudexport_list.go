package provider

import (
	"context"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	cloudexport "github.com/kentik/api-schema-public/gen/go/kentik/cloud_export/v202506"
)

func dataSourceCloudExportList() *schema.Resource {
	return &schema.Resource{
		Description: "Lists all Kentik cloud exports in the account (Kentik API `cloud_export/v202506`).",
		ReadContext: dataSourceCloudExportListRead,
		Schema: map[string]*schema.Schema{
			"items": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "All cloud exports with valid data.",
				Elem:        &schema.Resource{Schema: makeCloudExportSchema(readList)},
			},
			"invalid_exports_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of cloud exports with invalid data, which the API does not return.",
			},
		},
	}
}

func dataSourceCloudExportListRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	resp, err := clientFrom(m).CloudExport.ListCloudExports(ctx, &cloudexport.ListCloudExportsRequest{})
	if err != nil {
		return detailedDiagError("Failed to read cloud export list", err)
	}
	items := make([]any, 0, len(resp.GetExports()))
	for _, e := range resp.GetExports() {
		items = append(items, cloudExportToMap(e, nil))
	}
	if err = d.Set("items", items); err != nil {
		return diag.FromErr(err)
	}
	if err = d.Set("invalid_exports_count", int(resp.GetInvalidExportsCount())); err != nil {
		return diag.FromErr(err)
	}
	if n := resp.GetInvalidExportsCount(); n > 0 {
		tflog.Warn(ctx, "Kentik API skipped cloud exports with invalid data", map[string]any{"count": n})
	}

	// Use the current time as ID so Terraform always treats the list as fresh.
	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))
	return nil
}
