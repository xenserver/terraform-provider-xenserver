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
	_ datasource.DataSource              = &networkDataSource{}
	_ datasource.DataSourceWithConfigure = &networkDataSource{}
)

// NewNetworkDataSource is a helper function to simplify the provider implementation.
func NewNetworkDataSource() datasource.DataSource {
	return &networkDataSource{}
}

// networkDataSource is the data source implementation.
type networkDataSource struct {
	session *xenapi.Session
}

// Metadata returns the data source type name.
func (d *networkDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network"
}

// Schema defines the schema for the data source.
func (d *networkDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Provides information about the network of XenServer",

		Attributes: map[string]schema.Attribute{
			"name_label": schema.StringAttribute{
				MarkdownDescription: "The name of the network",
				Optional:            true,
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the network",
				Optional:            true,
			},
			"data_items": schema.ListNestedAttribute{
				MarkdownDescription: "The return items of networks",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{
							MarkdownDescription: "The UUID of the network",
							Computed:            true,
						},
						"name_label": schema.StringAttribute{
							MarkdownDescription: "The name of the network",
							Computed:            true,
						},
						"name_description": schema.StringAttribute{
							MarkdownDescription: "The human-readable description of the network",
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
						"vifs": schema.ListAttribute{
							MarkdownDescription: "The list of connected vifs",
							Computed:            true,
							ElementType:         types.StringType,
						},
						"pifs": schema.ListAttribute{
							MarkdownDescription: "The list of connected pifs",
							Computed:            true,
							ElementType:         types.StringType,
						},
						"mtu": schema.Int32Attribute{
							MarkdownDescription: "The MTU in octets",
							Computed:            true,
						},
						"other_config": schema.MapAttribute{
							MarkdownDescription: "The additional configuration",
							Computed:            true,
							ElementType:         types.StringType,
						},
						"bridge": schema.StringAttribute{
							MarkdownDescription: "The name of the bridge corresponding to this network on the local host",
							Computed:            true,
						},
						"managed": schema.BoolAttribute{
							MarkdownDescription: "True if the bridge is managed by xapi",
							Computed:            true,
						},
						"blobs": schema.MapAttribute{
							MarkdownDescription: "The binary blobs associated with this SR",
							Computed:            true,
							ElementType:         types.StringType,
						},
						"tags": schema.ListAttribute{
							MarkdownDescription: "The user-specified tags for categorization purposes",
							Computed:            true,
							ElementType:         types.StringType,
						},
						"default_locking_mode": schema.StringAttribute{
							MarkdownDescription: "The network will use this value to determine the behavior of all VIFs where locking_mode = default",
							Computed:            true,
						},
						"assigned_ips": schema.MapAttribute{
							MarkdownDescription: "The IP addresses assigned to VIFs on networks that have active xapi-managed DHCP",
							Computed:            true,
							ElementType:         types.StringType,
						},
						"purpose": schema.ListAttribute{
							MarkdownDescription: "Set of purposes for which the server will use this network",
							Computed:            true,
							ElementType:         types.StringType,
						},
					},
				},
			},
		},
	}
}

func (d *networkDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *networkDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data networkDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	networkRecords, err := xenapi.Network.GetAllRecords(d.session)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get network records",
			err.Error(),
		)
		return
	}

	var networkItem []networkRecordData

	for _, networkRecord := range networkRecords {
		if !data.NameLabel.IsNull() && networkRecord.NameLabel != data.NameLabel.ValueString() {
			continue
		}
		if !data.UUID.IsNull() && networkRecord.UUID != data.UUID.ValueString() {
			continue
		}

		var networkData networkRecordData
		err = updateNetworkRecordData(ctx, networkRecord, &networkData)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to update network record data",
				err.Error(),
			)
			return
		}
		networkItem = append(networkItem, networkData)
	}

	// sort networkItem by UUID
	sort.Slice(networkItem, func(i, j int) bool {
		return networkItem[i].UUID.ValueString() < networkItem[j].UUID.ValueString()
	})

	data.DataItems = networkItem

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
