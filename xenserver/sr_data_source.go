package xenserver

import (
	"context"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"xenapi"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &srDataSource{}
	_ datasource.DataSourceWithConfigure = &srDataSource{}
)

// NewSRDataSource is a helper function to simplify the provider implementation.
func NewSRDataSource() datasource.DataSource {
	return &srDataSource{}
}

// srDataSource is the data source implementation.
type srDataSource struct {
	session *xenapi.Session
}

// Metadata returns the data source type name.
func (d *srDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sr"
}

// Schema defines the schema for the data source.
func (d *srDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Provides information about the storage repository of XenServer",

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

func (d *srDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *srDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data srDataSourceModel
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

	var srItems []srRecordData

	for _, srRecord := range srRecords {
		if !data.NameLabel.IsNull() && srRecord.NameLabel != data.NameLabel.ValueString() {
			continue
		}
		if !data.UUID.IsNull() && srRecord.UUID != data.UUID.ValueString() {
			continue
		}

		var srData srRecordData
		err = updateSRRecordData(ctx, srRecord, &srData)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to update SR record data",
				err.Error(),
			)
			return
		}
		srItems = append(srItems, srData)
	}

	sort.Slice(srItems, func(i, j int) bool {
		return srItems[i].UUID.ValueString() < srItems[j].UUID.ValueString()
	})
	data.DataItems = srItems

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
