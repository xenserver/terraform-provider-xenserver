package xenserver

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"xenapi"
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
	MTU                   types.Int32  `tfsdk:"mtu"`
	VLAN                  types.Int32  `tfsdk:"vlan"`
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

func updatePIFRecordData(ctx context.Context, session *xenapi.Session, record xenapi.PIFRecord, data *pifRecordData) error {
	data.UUID = types.StringValue(record.UUID)
	data.Device = types.StringValue(record.Device)
	data.Management = types.BoolValue(record.Management)

	var err error
	networkUUID := ""
	if record.Network != "OpaqueRef:NULL" {
		networkUUID, err = xenapi.Network.GetUUID(session, record.Network)
		if err != nil {
			return errors.New("unable to read PIF network UUID")
		}
	}
	data.Network = types.StringValue(networkUUID)

	hostUUID, err := xenapi.Host.GetUUID(session, record.Host)
	if err != nil {
		return errors.New("unable to read PIF host UUID")
	}
	data.Host = types.StringValue(hostUUID)
	data.MAC = types.StringValue(record.MAC)
	data.MTU = types.Int32Value(int32(record.MTU))
	data.VLAN = types.Int32Value(int32(record.VLAN))
	data.Physical = types.BoolValue(record.Physical)
	data.CurrentlyAttached = types.BoolValue(record.CurrentlyAttached)
	data.IPConfigurationMode = types.StringValue(string(record.IPConfigurationMode))
	data.IP = types.StringValue(record.IP)
	data.Netmask = types.StringValue(record.Netmask)
	data.Gateway = types.StringValue(record.Gateway)
	data.DNS = types.StringValue(record.DNS)

	bondUUID := ""
	if record.BondSlaveOf != "OpaqueRef:NULL" {
		bondUUID, err = xenapi.Bond.GetUUID(session, record.BondSlaveOf)
		if err != nil {
			return errors.New(err.Error())
		}
	}
	data.BondSlaveOf = types.StringValue(bondUUID)

	var diags diag.Diagnostics
	bondMasterOf := []string{}
	for _, bondMasterRef := range record.BondMasterOf {
		bondUUID, err := xenapi.Bond.GetUUID(session, bondMasterRef)
		if err != nil {
			return errors.New(err.Error())
		}
		bondMasterOf = append(bondMasterOf, bondUUID)
	}
	data.BondMasterOf, diags = types.ListValueFrom(ctx, types.StringType, bondMasterOf)
	if diags.HasError() {
		return errors.New("unable to read PIF bond master of")
	}

	vlanUUID := ""
	if record.VLANMasterOf != "OpaqueRef:NULL" {
		vlanUUID, err = xenapi.VLAN.GetUUID(session, record.VLANMasterOf)
		if err != nil {
			return errors.New(err.Error())
		}
	}
	data.VLANMasterOf = types.StringValue(vlanUUID)

	vlanSlaveOf := []string{}
	for _, vlanSlaveRef := range record.VLANSlaveOf {
		vlanUUID, err := xenapi.VLAN.GetUUID(session, vlanSlaveRef)
		if err != nil {
			return errors.New(err.Error())
		}
		vlanSlaveOf = append(vlanSlaveOf, vlanUUID)
	}
	data.VLANSlaveOf, diags = types.ListValueFrom(ctx, types.StringType, vlanSlaveOf)
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

	pciUUID := ""
	if record.PCI != "OpaqueRef:NULL" {
		pciUUID, err = xenapi.PCI.GetUUID(session, record.PCI)
		if err != nil {
			return errors.New("unable to read PIF PCI UUID" + string(record.PCI))
		}
	}
	data.PCI = types.StringValue(pciUUID)
	return nil
}

type pifConfigureResourceModel struct {
	DisallowUnplug types.Bool   `tfsdk:"disallow_unplug"`
	Interface      types.Object `tfsdk:"interface"`
	UUID           types.String `tfsdk:"uuid"`
	ID             types.String `tfsdk:"id"`
}

type InterfaceObject struct {
	NameLabel types.String `tfsdk:"name_label"`
	Mode      types.String `tfsdk:"mode"`
	IP        types.String `tfsdk:"ip"`
	Gateway   types.String `tfsdk:"gateway"`
	Netmask   types.String `tfsdk:"netmask"`
	DNS       types.String `tfsdk:"dns"`
}

func getIPConfigurationMode(mode string) xenapi.IPConfigurationMode {
	var value xenapi.IPConfigurationMode
	switch mode {
	case "None":
		value = xenapi.IPConfigurationModeNone
	case "DHCP":
		value = xenapi.IPConfigurationModeDHCP
	case "Static":
		value = xenapi.IPConfigurationModeStatic
	default:
		value = xenapi.IPConfigurationModeUnrecognized
	}
	return value
}

func pifConfigureResourceModelUpdate(ctx context.Context, session *xenapi.Session, data pifConfigureResourceModel) error {
	pifRef, err := xenapi.PIF.GetByUUID(session, data.UUID.ValueString())
	if err != nil {
		return errors.New(err.Error() + ", uuid: " + data.UUID.ValueString())
	}

	if !data.DisallowUnplug.IsNull() {
		err := xenapi.PIF.SetDisallowUnplug(session, pifRef, data.DisallowUnplug.ValueBool())
		if err != nil {
			tflog.Error(ctx, "unable to update the PIF 'disallow_unplug'")
			return errors.New(err.Error())
		}
	}

	if !data.Interface.IsNull() {
		pifMetricsRef, err := xenapi.PIF.GetMetrics(session, pifRef)
		if err != nil {
			return errors.New(err.Error())
		}

		isPIFConnected, err := xenapi.PIFMetrics.GetCarrier(session, pifMetricsRef)
		if err != nil {
			return errors.New(err.Error())
		}

		if !isPIFConnected {
			return errors.New("the PIF with uuid " + data.UUID.ValueString() + " is not connected")
		}

		var interfaceObject InterfaceObject
		diags := data.Interface.As(ctx, &interfaceObject, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return errors.New("unable to read PIF interface config")
		}

		if !interfaceObject.NameLabel.IsNull() {
			oc, err := xenapi.PIF.GetOtherConfig(session, pifRef)
			if err != nil {
				return errors.New(err.Error())
			}

			oc["management_purpose"] = interfaceObject.NameLabel.ValueString()

			err = xenapi.PIF.SetOtherConfig(session, pifRef, oc)
			if err != nil {
				return errors.New(err.Error())
			}
		}

		mode := getIPConfigurationMode(interfaceObject.Mode.ValueString())
		ip := interfaceObject.IP.ValueString()
		netmask := interfaceObject.Netmask.ValueString()
		gateway := interfaceObject.Gateway.ValueString()
		dns := interfaceObject.DNS.ValueString()

		tflog.Debug(ctx, "Reconfigure PIF IP with mode: "+string(mode)+", ip: "+ip+", netmask: "+netmask+", gateway: "+gateway+", dns: "+dns)
		err = xenapi.PIF.ReconfigureIP(session, pifRef, mode, ip, netmask, gateway, dns)
		if err != nil {
			tflog.Error(ctx, "unable to update the PIF 'interface'")
			return errors.New(err.Error())
		}
		if string(mode) == "DHCP" {
			err := checkPIFHasIP(ctx, session, pifRef)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func checkPIFHasIP(ctx context.Context, session *xenapi.Session, ref xenapi.PIFRef) error {
	// set timeout channel to check if IP address is available
	timeoutChan := time.After(time.Duration(60) * time.Second)
	for {
		select {
		case <-timeoutChan:
			return errors.New("get PIF IP timeout in 60 seconds, please check if the interface is connected")
		default:
			ip, err := xenapi.PIF.GetIP(session, ref)
			if err != nil {
				tflog.Error(ctx, "unable to get the PIF IP")
				return errors.New(err.Error())
			}
			if isValidIpAddress(net.ParseIP(ip)) {
				tflog.Debug(ctx, "PIF IP is available: "+ip)
				return nil
			}

			tflog.Debug(ctx, "-----> Retry get PIF IP")
			time.Sleep(5 * time.Second)
		}
	}
}
