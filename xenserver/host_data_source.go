package xenserver

import (
	"context"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"xenapi"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &hostDataSource{}
	_ datasource.DataSourceWithConfigure = &hostDataSource{}
)

// NewHostDataSource is a helper function to simplify the provider implementation.
func NewHostDataSource() datasource.DataSource {
	return &hostDataSource{}
}

// hostDataSource is the data source implementation.
type hostDataSource struct {
	session *xenapi.Session
}

// Metadata returns the data source type name.
func (d *hostDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_host"
}

func (d *hostDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Provides information about the host.",
		Attributes: map[string]schema.Attribute{
			"name_label": schema.StringAttribute{
				MarkdownDescription: "The name of the host.",
				Optional:            true,
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the host.",
				Optional:            true,
			},
			"address": schema.StringAttribute{
				MarkdownDescription: "The address by which this host can be contacted from any other host in the pool.",
				Optional:            true,
			},
			"data_items": schema.ListNestedAttribute{
				MarkdownDescription: "The return items of host.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: hostDataSchema(),
				},
			},
		},
	}
}

func (d *hostDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *hostDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data hostDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hostRecords, err := xenapi.Host.GetAllRecords(d.session)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read Host records",
			err.Error(),
		)
		return
	}

	var hostItems []hostRecordData
	for _, hostRecord := range hostRecords {
		if !data.NameLabel.IsNull() && hostRecord.NameLabel != data.NameLabel.ValueString() {
			continue
		}
		if !data.UUID.IsNull() && hostRecord.UUID != data.UUID.ValueString() {
			continue
		}
		if !data.Address.IsNull() && hostRecord.Address != data.Address.ValueString() {
			continue
		}

		var hostData hostRecordData
		err = updateHostRecordData(ctx, d.session, hostRecord, &hostData)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to update Host record data",
				err.Error(),
			)
			return
		}
		hostItems = append(hostItems, hostData)
	}

	sort.Slice(hostItems, func(i, j int) bool {
		return hostItems[i].UUID.ValueString() < hostItems[j].UUID.ValueString()
	})
	data.DataItems = hostItems

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
