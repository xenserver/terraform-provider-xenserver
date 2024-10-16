// Example of data source

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

func pifDataSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"uuid": schema.StringAttribute{
			MarkdownDescription: "The UUID of the storage repository.",
			Computed:            true,
		},
		"device": schema.StringAttribute{
			MarkdownDescription: "The machine-readable name of the physical interface (PIF). (For example, `\"eth0\"`)",
			Computed:            true,
		},
		"management": schema.BoolAttribute{
			MarkdownDescription: "Indicates whether the control software is listening for connections on this physical interface.",
			Computed:            true,
		},
		"network": schema.StringAttribute{
			MarkdownDescription: "The UUID of the virtual network to which this PIF is connected.",
			Computed:            true,
		},
		"host": schema.StringAttribute{
			MarkdownDescription: "The UUID of the physical machine to which this PIF is connected.",
			Computed:            true,
		},
		"mac": schema.StringAttribute{
			MarkdownDescription: "Ethernet MAC address of the physical interface.",
			Computed:            true,
		},
		"mtu": schema.Int32Attribute{
			MarkdownDescription: "MTU in octets.",
			Computed:            true,
		},
		"vlan": schema.Int32Attribute{
			MarkdownDescription: "VLAN tag for all traffic passing through this interface.",
			Computed:            true,
		},
		"physical": schema.BoolAttribute{
			MarkdownDescription: "True if this represents a physical network interface.",
			Computed:            true,
		},
		"currently_attached": schema.BoolAttribute{
			MarkdownDescription: "True if this interface is online.",
			Computed:            true,
		},
		"ip_configuration_mode": schema.StringAttribute{
			MarkdownDescription: "Sets if and how this interface gets an IP address.",
			Computed:            true,
		},
		"ip": schema.StringAttribute{
			MarkdownDescription: "IP address.",
			Computed:            true,
		},
		"netmask": schema.StringAttribute{
			MarkdownDescription: "IP netmask.",
			Computed:            true,
		},
		"gateway": schema.StringAttribute{
			MarkdownDescription: "IP gateway.",
			Computed:            true,
		},
		"dns": schema.StringAttribute{
			MarkdownDescription: "Comma-separated list of the IP addresses of the DNS servers to use.",
			Computed:            true,
		},
		"bond_slave_of": schema.StringAttribute{
			MarkdownDescription: "Indicates which bond this interface is part of.",
			Computed:            true,
		},
		"bond_master_of": schema.ListAttribute{
			MarkdownDescription: "Indicates this PIF represents the results of a bond.",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"vlan_master_of": schema.StringAttribute{
			MarkdownDescription: "Indicates which VLAN this interface receives untagged traffic from.",
			Computed:            true,
		},
		"vlan_slave_of": schema.ListAttribute{
			MarkdownDescription: "Indicates which VLANs this interface transmits tagged traffic to.",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"other_config": schema.MapAttribute{
			MarkdownDescription: "Additional configuration.",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"disallow_unplug": schema.BoolAttribute{
			MarkdownDescription: "Prevent this PIF from being unplugged; set this to notify the management toolstack that the PIF has a special use and should not be unplugged under any circumstances. (For example, because you're running storage traffic over it)",
			Computed:            true,
		},
		"tunnel_access_pif_of": schema.ListAttribute{
			MarkdownDescription: "Indicates to which tunnel this PIF gives access.",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"tunnel_transport_pif_of": schema.ListAttribute{
			MarkdownDescription: "Indicates to which tunnel this PIF provides transport.",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"ipv6_configuration_mode": schema.StringAttribute{
			MarkdownDescription: "Sets if and how this interface gets an IPv6 address.",
			Computed:            true,
		},
		"ipv6": schema.ListAttribute{
			MarkdownDescription: "IPv6 address.",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"ipv6_gateway": schema.StringAttribute{
			MarkdownDescription: "IPv6 gateway.",
			Computed:            true,
		},
		"primary_address_type": schema.StringAttribute{
			MarkdownDescription: "Which protocol should define the primary address of this interface.",
			Computed:            true,
		},
		"managed": schema.BoolAttribute{
			MarkdownDescription: "Indicates whether the interface is managed by [XAPI](https://github.com/xapi-project/xen-api). If it is not, then XAPI will not configure the interface, the commands PIF.plug/unplug/reconfigure_ip(v6) cannot be used, nor can the interface be bonded or have VLANs based on top through XAPI.",
			Computed:            true,
		},
		"properties": schema.MapAttribute{
			MarkdownDescription: "Additional configuration properties for the interface.",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"capabilities": schema.ListAttribute{
			MarkdownDescription: "Additional capabilities on the interface.",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"igmp_snooping_status": schema.StringAttribute{
			MarkdownDescription: "The IGMP snooping status of the corresponding network bridge.",
			Computed:            true,
		},
		"sriov_physical_pif_of": schema.ListAttribute{
			MarkdownDescription: "Indicates which network_sriov this interface is the physical PIF of.",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"sriov_logical_pif_of": schema.ListAttribute{
			MarkdownDescription: "Indicates which network_sriov this interface is the logical PIF of.",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"pci": schema.StringAttribute{
			MarkdownDescription: "Link to underlying PCI device.",
			Computed:            true,
		},
	}
}

func (d *pifDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Provides information about the physical network interface (PIF).",
		Attributes: map[string]schema.Attribute{
			"device": schema.StringAttribute{
				MarkdownDescription: "The machine-readable name of the physical interface (PIF). (eg. `\"eth0\"`)",
				Optional:            true,
			},
			"management": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether the control software is listening for connections on this physical interface.",
				Optional:            true,
			},
			"network": schema.StringAttribute{
				MarkdownDescription: "The UUID of the virtual network to which this PIF is connected.",
				Optional:            true,
			},
			"data_items": schema.ListNestedAttribute{
				MarkdownDescription: "The return items of physical network interfaces.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: pifDataSchema(),
				},
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
			"Unable to read PIF records",
			err.Error(),
		)
		return
	}

	var pifItems []pifRecordData
	for _, pifRecord := range pifRecords {
		if !data.Network.IsNull() {
			NetworkRef, err := xenapi.Network.GetByUUID(d.session, data.Network.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Unable to get network reference",
					err.Error(),
				)
				return
			}
			if pifRecord.Network != NetworkRef {
				continue
			}
		}

		if !data.Device.IsNull() && pifRecord.Device != data.Device.ValueString() {
			continue
		}

		if !data.Management.IsNull() && pifRecord.Management != data.Management.ValueBool() {
			continue
		}

		var pifData pifRecordData
		err = updatePIFRecordData(ctx, d.session, pifRecord, &pifData)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to update PIF record data",
				err.Error(),
			)
			return
		}
		pifItems = append(pifItems, pifData)
	}

	sort.Slice(pifItems, func(i, j int) bool {
		return pifItems[i].UUID.ValueString() < pifItems[j].UUID.ValueString()
	})
	data.DataItems = pifItems

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
