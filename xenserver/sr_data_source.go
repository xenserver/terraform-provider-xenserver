package xenserver

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"xenapi"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &SRDataSource{}
	_ datasource.DataSourceWithConfigure = &SRDataSource{}
)

// NewSRDataSource is a helper function to simplify the provider implementation.
func NewSRDataSource() datasource.DataSource {
	return &SRDataSource{}
}

// SRDataSource is the data source implementation.
type SRDataSource struct {
	session *xenapi.Session
}

// SRDataSourceModel describes the data source data model.
type SRDataSourceModel struct {
	NameLabel types.String   `tfsdk:"name_label"`
	UUID      types.String   `tfsdk:"uuid"`
	DataItems []SRRecordData `tfsdk:"data_items"`
}

type SRRecordData struct {
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

// Metadata returns the data source type name.
func (d *SRDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sr"
}

// Schema defines the schema for the data source.
func (d *SRDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The data source of XenServer storage repository",

		Attributes: map[string]schema.Attribute{
			"name_label": schema.StringAttribute{
				MarkdownDescription: "The name of the storage repository",
				Optional:            true,
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the storage repository",
				Optional:            true,
			},
			"data_items": schema.ListNestedAttribute{
				MarkdownDescription: "The return items of storage repositories",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{
							MarkdownDescription: "The UUID of the storage repository",
							Computed:            true,
						},
						"name_label": schema.StringAttribute{
							MarkdownDescription: "The name of the storage repository",
							Computed:            true,
						},
						"name_description": schema.StringAttribute{
							MarkdownDescription: "The human-readable description of the storage repository",
							Computed:            true,
						},
						"allowed_operations": schema.ListAttribute{
							MarkdownDescription: "The list of the operations allowed in this state",
							Computed:            true,
							ElementType:         types.StringType,
						},
						"current_operations": schema.MapAttribute{
							MarkdownDescription: "The links each of the running tasks using this object (by reference) to a current_operation enum which describes the nature of the task",
							Computed:            true,
							ElementType:         types.StringType,
						},
						"vdis": schema.ListAttribute{
							MarkdownDescription: "The all virtual disks known to this storage repository",
							Computed:            true,
							ElementType:         types.StringType,
						},
						"pbds": schema.ListAttribute{
							MarkdownDescription: "Describes how particular hosts can see this storage repository",
							Computed:            true,
							ElementType:         types.StringType,
						},
						"virtual_allocation": schema.Int64Attribute{
							MarkdownDescription: "The sum of virtual_sizes of all VDIs in this storage repository (in bytes)",
							Computed:            true,
						},
						"physical_utilisation": schema.Int64Attribute{
							MarkdownDescription: "The physical space currently utilised on this storage repository (in bytes)",
							Computed:            true,
						},
						"physical_size": schema.Int64Attribute{
							MarkdownDescription: "The total physical size of the storage repository (in bytes)",
							Computed:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "The type of the storage repository",
							Computed:            true,
						},
						"content_type": schema.StringAttribute{
							MarkdownDescription: "The type of the SR's content, if required (e.g. ISOs)",
							Computed:            true,
						},
						"shared": schema.BoolAttribute{
							MarkdownDescription: "True if this SR is (capable of being) shared between multiple hosts",
							Computed:            true,
						},
						"other_config": schema.MapAttribute{
							MarkdownDescription: "The additional configuration",
							Computed:            true,
							ElementType:         types.StringType,
						},
						"tags": schema.ListAttribute{
							MarkdownDescription: "The user-specified tags for categorization purposes",
							Computed:            true,
							ElementType:         types.StringType,
						},
						"sm_config": schema.MapAttribute{
							MarkdownDescription: "The SM dependent data",
							Computed:            true,
							ElementType:         types.StringType,
						},
						"blobs": schema.MapAttribute{
							MarkdownDescription: "The binary blobs associated with this SR",
							Computed:            true,
							ElementType:         types.StringType,
						},
						"local_cache_enabled": schema.BoolAttribute{
							MarkdownDescription: "True if this SR is assigned to be the local cache for its host",
							Computed:            true,
						},
						"introduced_by": schema.StringAttribute{
							MarkdownDescription: "The disaster recovery task which introduced this SR",
							Computed:            true,
						},
						"clustered": schema.BoolAttribute{
							MarkdownDescription: "True if the SR is using aggregated local storage",
							Computed:            true,
						},
						"is_tools_sr": schema.BoolAttribute{
							MarkdownDescription: "True if this is the SR that contains the Tools ISO VDIs",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *SRDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	session, ok := req.ProviderData.(*xenapi.Session)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *xenapi.Session, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	d.session = session
}

// Read refreshes the Terraform state with the latest data.
func (d *SRDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SRDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	srRecords, err := xenapi.SR.GetAllRecords(d.session)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get SR records",
			err.Error(),
		)
		return
	}

	var srItems []SRRecordData
	var diags diag.Diagnostics
	for _, srRecord := range srRecords {
		if !data.NameLabel.IsNull() && srRecord.NameLabel != data.NameLabel.ValueString() {
			continue
		}
		if !data.UUID.IsNull() && srRecord.UUID != data.UUID.ValueString() {
			continue
		}
		var srData SRRecordData
		srData.UUID = types.StringValue(srRecord.UUID)
		srData.NameLabel = types.StringValue(srRecord.NameLabel)
		srData.NameDescription = types.StringValue(srRecord.NameDescription)
		srData.AllowedOperations, diags = types.ListValueFrom(ctx, types.StringType, srRecord.AllowedOperations)
		if diags.HasError() {
			resp.Diagnostics.AddError(
				"Unable to read SR allowed operations",
				err.Error(),
			)
			return
		}
		srData.CurrentOperations, diags = types.MapValueFrom(ctx, types.StringType, srRecord.CurrentOperations)
		if diags.HasError() {
			resp.Diagnostics.AddError(
				"Unable to read SR current operations",
				err.Error(),
			)
			return
		}
		srData.VDIs, diags = types.ListValueFrom(ctx, types.StringType, srRecord.VDIs)
		if diags.HasError() {
			resp.Diagnostics.AddError(
				"Unable to read SR VDIs",
				err.Error(),
			)
			return
		}
		srData.PBDs, diags = types.ListValueFrom(ctx, types.StringType, srRecord.PBDs)
		if diags.HasError() {
			resp.Diagnostics.AddError(
				"Unable to read SR PBDs",
				err.Error(),
			)
			return
		}
		srData.VirtualAllocation = types.Int64Value(int64(srRecord.VirtualAllocation))
		srData.PhysicalUtilisation = types.Int64Value(int64(srRecord.PhysicalUtilisation))
		srData.PhysicalSize = types.Int64Value(int64(srRecord.PhysicalSize))
		srData.Type = types.StringValue(srRecord.Type)
		srData.ContentType = types.StringValue(srRecord.ContentType)
		srData.Shared = types.BoolValue(srRecord.Shared)
		srData.OtherConfig, diags = types.MapValueFrom(ctx, types.StringType, srRecord.OtherConfig)
		if diags.HasError() {
			resp.Diagnostics.AddError(
				"Unable to read SR other config",
				err.Error(),
			)
			return
		}
		srData.Tags, diags = types.ListValueFrom(ctx, types.StringType, srRecord.Tags)
		if diags.HasError() {
			resp.Diagnostics.AddError(
				"Unable to Read SR Tags",
				err.Error(),
			)
			return
		}
		srData.SmConfig, diags = types.MapValueFrom(ctx, types.StringType, srRecord.SmConfig)
		if diags.HasError() {
			resp.Diagnostics.AddError(
				"Unable to read SR SM config",
				err.Error(),
			)
			return
		}
		srData.Blobs, diags = types.MapValueFrom(ctx, types.StringType, srRecord.Blobs)
		if diags.HasError() {
			resp.Diagnostics.AddError(
				"Unable to read SR Blobs",
				err.Error(),
			)
			return
		}
		srData.LocalCacheEnabled = types.BoolValue(srRecord.LocalCacheEnabled)
		srData.IntroducedBy = types.StringValue(string(srRecord.IntroducedBy))
		srData.Clustered = types.BoolValue(srRecord.Clustered)
		srData.IsToolsSr = types.BoolValue(srRecord.IsToolsSr)

		srItems = append(srItems, srData)
	}
	data.DataItems = srItems

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
