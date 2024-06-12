package xenserver

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"xenapi"
)

// srDataSourceModel describes the data source data model.
type srDataSourceModel struct {
	NameLabel types.String   `tfsdk:"name_label"`
	UUID      types.String   `tfsdk:"uuid"`
	DataItems []srRecordData `tfsdk:"data_items"`
}

type srRecordData struct {
	UUID                types.String `tfsdk:"uuid"`
	NameLabel           types.String `tfsdk:"name_label"`
	NameDescription     types.String `tfsdk:"name_description"`
	AllowedOperations   types.List   `tfsdk:"allowed_operations"`
	CurrentOperations   types.Map    `tfsdk:"current_operations"`
	VDIs                types.List   `tfsdk:"vdis"`
	PBDs                types.List   `tfsdk:"pbds"`
	VirtualAllocation   types.Int64  `tfsdk:"virtual_allocation"`
	PhysicalUtilisation types.Int64  `tfsdk:"physical_utilisation"`
	PhysicalSize        types.Int64  `tfsdk:"physical_size"`
	Type                types.String `tfsdk:"type"`
	ContentType         types.String `tfsdk:"content_type"`
	Shared              types.Bool   `tfsdk:"shared"`
	OtherConfig         types.Map    `tfsdk:"other_config"`
	Tags                types.List   `tfsdk:"tags"`
	SmConfig            types.Map    `tfsdk:"sm_config"`
	Blobs               types.Map    `tfsdk:"blobs"`
	LocalCacheEnabled   types.Bool   `tfsdk:"local_cache_enabled"`
	IntroducedBy        types.String `tfsdk:"introduced_by"`
	Clustered           types.Bool   `tfsdk:"clustered"`
	IsToolsSr           types.Bool   `tfsdk:"is_tools_sr"`
}

func updateSRRecordData(ctx context.Context, record xenapi.SRRecord, data *srRecordData) error {
	data.UUID = types.StringValue(record.UUID)
	data.NameLabel = types.StringValue(record.NameLabel)
	data.NameDescription = types.StringValue(record.NameDescription)
	var diags diag.Diagnostics
	data.AllowedOperations, diags = types.ListValueFrom(ctx, types.StringType, record.AllowedOperations)
	if diags.HasError() {
		return errors.New("unable to read SR allowed operations")
	}
	data.CurrentOperations, diags = types.MapValueFrom(ctx, types.StringType, record.CurrentOperations)
	if diags.HasError() {
		return errors.New("unable to read SR current operation")
	}
	data.VDIs, diags = types.ListValueFrom(ctx, types.StringType, record.VDIs)
	if diags.HasError() {
		return errors.New("unable to read SR VDIs")
	}
	data.PBDs, diags = types.ListValueFrom(ctx, types.StringType, record.PBDs)
	if diags.HasError() {
		return errors.New("unable to read SR PBDs")
	}
	data.VirtualAllocation = types.Int64Value(int64(record.VirtualAllocation))
	data.PhysicalUtilisation = types.Int64Value(int64(record.PhysicalUtilisation))
	data.PhysicalSize = types.Int64Value(int64(record.PhysicalSize))
	data.Type = types.StringValue(record.Type)
	data.ContentType = types.StringValue(record.ContentType)
	data.Shared = types.BoolValue(record.Shared)
	data.OtherConfig, diags = types.MapValueFrom(ctx, types.StringType, record.OtherConfig)
	if diags.HasError() {
		return errors.New("unable to read SR other config")
	}
	data.Tags, diags = types.ListValueFrom(ctx, types.StringType, record.Tags)
	if diags.HasError() {
		return errors.New("unable to read SR tags")
	}
	data.SmConfig, diags = types.MapValueFrom(ctx, types.StringType, record.SmConfig)
	if diags.HasError() {
		return errors.New("unable to read SR SM config")
	}
	data.Blobs, diags = types.MapValueFrom(ctx, types.StringType, record.Blobs)
	if diags.HasError() {
		return errors.New("unable to read SR blobs")
	}
	data.LocalCacheEnabled = types.BoolValue(record.LocalCacheEnabled)
	data.IntroducedBy = types.StringValue(string(record.IntroducedBy))
	data.Clustered = types.BoolValue(record.Clustered)
	data.IsToolsSr = types.BoolValue(record.IsToolsSr)
	return nil
}
