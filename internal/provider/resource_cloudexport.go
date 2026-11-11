package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	cloudexport "github.com/kentik/api-schema-public/gen/go/kentik/cloud_export/v202506"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func resourceCloudExport() *schema.Resource {
	return &schema.Resource{
		Description: "Manages a Kentik cloud export (Kentik API `cloud_export/v202506`): the configuration Kentik " +
			"uses to collect flow logs and metadata from AWS, Azure, Google Cloud or OCI.",
		CreateContext: resourceCloudExportCreate,
		ReadContext:   resourceCloudExportRead,
		UpdateContext: resourceCloudExportUpdate,
		DeleteContext: resourceCloudExportDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		// Version 0 is shared with v0.4.x: no attribute was renamed or changed
		// type, so no state upgrader is needed (see docs/UPGRADING.md).
		SchemaVersion: 0,
		Schema:        makeCloudExportSchema(create),
	}
}

func resourceCloudExportCreate(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	export, err := resourceDataToCloudExport(d)
	if err != nil {
		return diag.FromErr(err)
	}
	export.Id = ""

	resp, err := clientFrom(m).CloudExport.CreateCloudExport(ctx, &cloudexport.CreateCloudExportRequest{Export: export})
	if err != nil {
		return detailedDiagError("Failed to create cloud export", err)
	}
	d.SetId(resp.GetExport().GetId())
	tflog.Debug(ctx, "Created cloud export", map[string]any{"id": d.Id()})

	// Read back to pick up server-side defaults and computed attributes.
	return resourceCloudExportRead(ctx, d, m)
}

func resourceCloudExportRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	resp, err := clientFrom(m).CloudExport.GetCloudExport(ctx, &cloudexport.GetCloudExportRequest{Id: d.Id()})
	if err != nil {
		if status.Code(err) == codes.NotFound && !d.IsNewResource() {
			tflog.Warn(ctx, "Cloud export not found, removing from state", map[string]any{"id": d.Id()})
			d.SetId("")
			return nil
		}
		return detailedDiagError("Failed to read cloud export", err)
	}
	return setAttributes(d, cloudExportToMap(resp.GetExport(), d.Get))
}

func resourceCloudExportUpdate(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	export, err := resourceDataToCloudExport(d)
	if err != nil {
		return diag.FromErr(err)
	}
	if _, err = clientFrom(m).CloudExport.UpdateCloudExport(ctx, &cloudexport.UpdateCloudExportRequest{Export: export}); err != nil {
		return detailedDiagError("Failed to update cloud export", err)
	}
	return resourceCloudExportRead(ctx, d, m)
}

func resourceCloudExportDelete(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	_, err := clientFrom(m).CloudExport.DeleteCloudExport(ctx, &cloudexport.DeleteCloudExportRequest{Id: d.Id()})
	if err != nil && status.Code(err) != codes.NotFound {
		return detailedDiagError("Failed to delete cloud export", err)
	}
	return nil
}

func setAttributes(d *schema.ResourceData, attrs map[string]any) diag.Diagnostics {
	for k, v := range attrs {
		if err := d.Set(k, v); err != nil {
			return diag.Errorf("set %s: %v", k, err)
		}
	}
	return nil
}
