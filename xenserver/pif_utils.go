package xenserver

import (
	"context"
	"errors"
	"xenapi"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// pifDataSourceModel describes the data source data model.
type pifDataSourceModel struct {
	Device     types.String    `tfsdk:"device"`
	Management types.Bool      `tfsdk:"management"`
	Network    types.String    `tfsdk:"network"`
	DataItems  []pifRecordData `tfsdk:"data_items"`
}

type pifRecordData struct {
	UUID                  types.String `tfsdk:"uuid"`
	Device                types.String `tfsdk:"device"`
	Management            types.Bool   `tfsdk:"management"`
	Network               types.String `tfsdk:"network"`
	Host                  types.String `tfsdk:"host"`
	MAC                   types.String `tfsdk:"mac"`
	MTU                   types.Int64  `tfsdk:"mtu"`
	VLAN                  types.Int64  `tfsdk:"vlan"`
	Physical              types.Bool   `tfsdk:"physical"`
	CurrentlyAttached     types.Bool   `tfsdk:"currently_attached"`
	IPConfigurationMode   types.String `tfsdk:"ip_configuration_mode"`
	IP                    types.String `tfsdk:"ip"`
	Netmask               types.String `tfsdk:"netmask"`
	Gateway               types.String `tfsdk:"gateway"`
	DNS                   types.String `tfsdk:"dns"`
	BondSlaveOf           types.String `tfsdk:"bond_slave_of"`
	BondMasterOf          types.List   `tfsdk:"bond_master_of"`
	VLANMasterOf          types.String `tfsdk:"vlan_master_of"`
	VLANSlaveOf           types.List   `tfsdk:"vlan_slave_of"`
	OtherConfig           types.Map    `tfsdk:"other_config"`
	DisallowUnplug        types.Bool   `tfsdk:"disallow_unplug"`
	TunnelAccessPIFOf     types.List   `tfsdk:"tunnel_access_pif_of"`
	TunnelTransportPIFOf  types.List   `tfsdk:"tunnel_transport_pif_of"`
	IPv5ConfigurationMode types.String `tfsdk:"ipv6_configuration_mode"`
	IPv5                  types.List   `tfsdk:"ipv6"`
	IPv5Gateway           types.String `tfsdk:"ipv6_gateway"`
	PrimaryAddressType    types.String `tfsdk:"primary_address_type"`
	Managed               types.Bool   `tfsdk:"managed"`
	Properties            types.Map    `tfsdk:"properties"`
	Capabilities          types.List   `tfsdk:"capabilities"`
	IGMPSnoopingStatus    types.String `tfsdk:"igmp_snooping_status"`
	SRIOVPhysicalPIFOf    types.List   `tfsdk:"sriov_physical_pif_of"`
	SRIOVLogicalPIFOf     types.List   `tfsdk:"sriov_logical_pif_of"`
	PCI                   types.String `tfsdk:"pci"`
}

func updatePIFRecordData(ctx context.Context, record xenapi.PIFRecord, data *pifRecordData) error {
	data.UUID = types.StringValue(record.UUID)
	data.Device = types.StringValue(record.Device)
	data.Management = types.BoolValue(record.Management)
	data.Network = types.StringValue(string(record.Network))
	data.Host = types.StringValue(string(record.Host))
	data.MAC = types.StringValue(record.MAC)
	data.MTU = types.Int64Value(int64(record.MTU))
	data.VLAN = types.Int64Value(int64(record.VLAN))
	data.Physical = types.BoolValue(record.Physical)
	data.CurrentlyAttached = types.BoolValue(record.CurrentlyAttached)
	data.IPConfigurationMode = types.StringValue(string(record.IPConfigurationMode))
	data.IP = types.StringValue(record.IP)
	data.Netmask = types.StringValue(record.Netmask)
	data.Gateway = types.StringValue(record.Gateway)
	data.DNS = types.StringValue(record.DNS)
	data.BondSlaveOf = types.StringValue(string(record.BondSlaveOf))
	var diags diag.Diagnostics
	data.BondMasterOf, diags = types.ListValueFrom(ctx, types.StringType, record.BondMasterOf)
	if diags.HasError() {
		return errors.New("unable to read PIF bond master of")
	}
	data.VLANMasterOf = types.StringValue(string(record.VLANMasterOf))
	data.VLANSlaveOf, diags = types.ListValueFrom(ctx, types.StringType, record.VLANSlaveOf)
	if diags.HasError() {
		return errors.New("unable to read PIF VLAN slave of")
	}
	data.OtherConfig, diags = types.MapValueFrom(ctx, types.StringType, record.OtherConfig)
	if diags.HasError() {
		return errors.New("unable to read PIF other config")
	}
	data.DisallowUnplug = types.BoolValue(record.DisallowUnplug)
	data.TunnelAccessPIFOf, diags = types.ListValueFrom(ctx, types.StringType, record.TunnelAccessPIFOf)
	if diags.HasError() {
		return errors.New("unable to read PIF tunnel access PIF of")
	}
	data.TunnelTransportPIFOf, diags = types.ListValueFrom(ctx, types.StringType, record.TunnelTransportPIFOf)
	if diags.HasError() {
		return errors.New("unable to read PIF tunnel transport PIF of")
	}
	data.IPv5ConfigurationMode = types.StringValue(string(record.Ipv6ConfigurationMode))
	data.IPv5, diags = types.ListValueFrom(ctx, types.StringType, record.IPv6)
	if diags.HasError() {
		return errors.New("unable to read PIF IPv6")
	}
	data.IPv5Gateway = types.StringValue(record.Ipv6Gateway)
	data.PrimaryAddressType = types.StringValue(string(record.PrimaryAddressType))
	data.Managed = types.BoolValue(record.Managed)
	data.Properties, diags = types.MapValueFrom(ctx, types.StringType, record.Properties)
	if diags.HasError() {
		return errors.New("unable to read PIF properties")
	}
	data.Capabilities, diags = types.ListValueFrom(ctx, types.StringType, record.Capabilities)
	if diags.HasError() {
		return errors.New("unable to read PIF capabilities")
	}
	data.IGMPSnoopingStatus = types.StringValue(string(record.IgmpSnoopingStatus))
	data.SRIOVPhysicalPIFOf, diags = types.ListValueFrom(ctx, types.StringType, record.SriovPhysicalPIFOf)
	if diags.HasError() {
		return errors.New("unable to read PIF SR-IOV physical PIF of")
	}
	data.SRIOVLogicalPIFOf, diags = types.ListValueFrom(ctx, types.StringType, record.SriovLogicalPIFOf)
	if diags.HasError() {
		return errors.New("unable to read PIF SR-IOV logical PIF of")
	}
	data.PCI = types.StringValue(string(record.PCI))
	return nil
}
