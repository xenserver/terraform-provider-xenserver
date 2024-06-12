// Example of data source

package xenserver

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"xenapi"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &pifDataSource{}
	_ datasource.DataSourceWithConfigure = &pifDataSource{}
)

// NewPIFDataSource is a helper function to simplify the provider implementation.
func NewPIFDataSource() datasource.DataSource {
	return &pifDataSource{}
}

// pifDataSource is the data source implementation.
type pifDataSource struct {
	session *xenapi.Session
}

// Metadata returns the data source type name.
func (d *pifDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pif"
}

// Schema defines the schema for the data source.
func (d *pifDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "PIF data source",

		Attributes: map[string]schema.Attribute{
			"device": schema.StringAttribute{
				MarkdownDescription: "The machine-readable name of the physical interface (PIF) (e.g. eth0)",
				Optional:            true,
			},
			"management": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether the control software is listening for connections on this physical interface",
				Optional:            true,
			},
			"network": schema.StringAttribute{
				MarkdownDescription: "UUID of the virtual network to which this PIF is connected",
				Computed:            true,
			},
		},
	}
}

func (d *pifDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *pifDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data pifDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pifRecords, err := xenapi.PIF.GetAllRecords(d.session)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read PIF records",
			err.Error(),
		)
		return
	}

	for _, pifRecord := range pifRecords {
		if pifRecord.Device == data.Device.ValueString() && pifRecord.Management == data.Management.ValueBool() {
			data.Network = types.StringValue(pifRecord.UUID)
			break
		}
	}

	tflog.Debug(ctx, "read a data source")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
