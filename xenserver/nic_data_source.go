package xenserver

import (
	"context"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"xenapi"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &nicDataSource{}
	_ datasource.DataSourceWithConfigure = &nicDataSource{}
)

// NewNICDataSource is a helper function to simplify the provider implementation.
func NewNICDataSource() datasource.DataSource {
	return &nicDataSource{}
}

// nicDataSource is the data source implementation.
type nicDataSource struct {
	session *xenapi.Session
}

// Metadata returns the data source type name.
func (d *nicDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nic"
}

// Schema defines the schema for the data source.
func (d *nicDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Provides the available NIC list for create different types of network in XenServer",

		Attributes: map[string]schema.Attribute{
			"network_type": schema.StringAttribute{
				MarkdownDescription: "The type of the network, choose one of  [`bond` - Bonded networks | `vlan` - External networks | `sriov` - SR-IOV networks | `private` - Single-Server Private networks], learn more on [page](https://docs.xenserver.com/en-us/xenserver/8/networking.html#xenserver-networking-overview) ",
				Optional:            true,
			},
			"data_items": schema.ListAttribute{
				MarkdownDescription: "The return list of available NICs for selected network type",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (d *nicDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *nicDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data nicDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bondNICs, err := getBondNICs(d.session)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get bond type NICs", err.Error())
		return
	}
	pifRecords, err := xenapi.PIF.GetAllRecords(d.session)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get PIF records", err.Error())
		return
	}
	physicalWithoutBondNICs := getPhysicalWithoutBondNICs(pifRecords)
	nonPhysicalSRIOVNICs := getNonPhysicalSRIOVNICs(pifRecords)

	var availableNICs []string
	if !data.NetworkType.IsNull() {
		switch data.NetworkType.ValueString() {
		case "vlan":
			availableNICs = slices.Concat(bondNICs, physicalWithoutBondNICs, nonPhysicalSRIOVNICs)
		case "bond":
			availableNICs = physicalWithoutBondNICs
		case "sriov":
			availableNICs = getPhysicalSRIOVNICs(pifRecords, true)
		default:
			availableNICs = []string{}
		}
	} else {
		availableNICs = slices.Concat(bondNICs, getPhysicalNICs(pifRecords), nonPhysicalSRIOVNICs)
	}
	data.DataItems = unique(availableNICs)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
