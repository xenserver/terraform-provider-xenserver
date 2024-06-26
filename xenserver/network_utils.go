package xenserver

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"xenapi"
)

type networkDataSourceModel struct {
	NameLabel types.String        `tfsdk:"name_label"`
	UUID      types.String        `tfsdk:"uuid"`
	DataItems []networkRecordData `tfsdk:"data_items"`
}

type networkRecordData struct {
	UUID               types.String `tfsdk:"uuid"`
	NameLabel          types.String `tfsdk:"name_label"`
	NameDescription    types.String `tfsdk:"name_description"`
	AllowedOperations  types.List   `tfsdk:"allowed_operations"`
	CurrentOperations  types.Map    `tfsdk:"current_operations"`
	VIFs               types.List   `tfsdk:"vifs"`
	PIFs               types.List   `tfsdk:"pifs"`
	MTU                types.Int64  `tfsdk:"mtu"`
	OtherConfig        types.Map    `tfsdk:"other_config"`
	Bridge             types.String `tfsdk:"bridge"`
	Managed            types.Bool   `tfsdk:"managed"`
	Blobs              types.Map    `tfsdk:"blobs"`
	Tags               types.List   `tfsdk:"tags"`
	DefaultLockingMode types.String `tfsdk:"default_locking_mode"`
	AssignedIps        types.Map    `tfsdk:"assigned_ips"`
	Purpose            types.List   `tfsdk:"purpose"`
}

func updateNetworkRecordData(ctx context.Context, record xenapi.NetworkRecord, data *networkRecordData) error {
	data.UUID = types.StringValue(record.UUID)
	data.NameLabel = types.StringValue(record.NameLabel)
	data.NameDescription = types.StringValue(record.NameDescription)
	var diags diag.Diagnostics
	data.AllowedOperations, diags = types.ListValueFrom(ctx, types.StringType, record.AllowedOperations)
	if diags.HasError() {
		return errors.New("unable to read network allowed operations")
	}
	data.CurrentOperations, diags = types.MapValueFrom(ctx, types.StringType, record.CurrentOperations)
	if diags.HasError() {
		return errors.New("unable to read network current operation")
	}
	data.VIFs, diags = types.ListValueFrom(ctx, types.StringType, record.VIFs)
	if diags.HasError() {
		return errors.New("unable to read network VIFs")
	}
	data.PIFs, diags = types.ListValueFrom(ctx, types.StringType, record.PIFs)
	if diags.HasError() {
		return errors.New("unable to read network PIFs")
	}
	data.MTU = types.Int64Value(int64(record.MTU))
	data.OtherConfig, diags = types.MapValueFrom(ctx, types.StringType, record.OtherConfig)
	if diags.HasError() {
		return errors.New("unable to read network other config")
	}
	data.Bridge = types.StringValue(record.Bridge)
	data.Managed = types.BoolValue(record.Managed)
	data.Blobs, diags = types.MapValueFrom(ctx, types.StringType, record.Blobs)
	if diags.HasError() {
		return errors.New("unable to read network blobs")
	}
	data.Tags, diags = types.ListValueFrom(ctx, types.StringType, record.Tags)
	if diags.HasError() {
		return errors.New("unable to read network tags")
	}
	data.DefaultLockingMode = types.StringValue(string(record.DefaultLockingMode))
	data.AssignedIps, diags = types.MapValueFrom(ctx, types.StringType, record.AssignedIps)
	if diags.HasError() {
		return errors.New("unable to read network assigned_ips")
	}
	data.Purpose, diags = types.ListValueFrom(ctx, types.StringType, record.Purpose)
	if diags.HasError() {
		return errors.New("unable to read network purpose")
	}

	return nil
}

type networkResourceModel struct {
	NameLabel       types.String `tfsdk:"name_label"`
	NameDescription types.String `tfsdk:"name_description"`
	MTU             types.Int64  `tfsdk:"mtu"`
	Managed         types.Bool   `tfsdk:"managed"`
	OtherConfig     types.Map    `tfsdk:"other_config"`
	UUID            types.String `tfsdk:"id"`
}

func updateNetworkResourceModel(ctx context.Context, networkRecord xenapi.NetworkRecord, data *networkResourceModel) error {
	data.NameLabel = types.StringValue(networkRecord.NameLabel)

	err := updateNetworkResourceModelComputed(ctx, networkRecord, data)
	if err != nil {
		return err
	}
	return nil
}

func updateNetworkResourceModelComputed(ctx context.Context, networkRecord xenapi.NetworkRecord, data *networkResourceModel) error {
	data.UUID = types.StringValue(networkRecord.UUID)
	data.NameDescription = types.StringValue(networkRecord.NameDescription)
	data.MTU = types.Int64Value(int64(networkRecord.MTU))
	data.Managed = types.BoolValue(networkRecord.Managed)

	otherConfig, diags := types.MapValueFrom(ctx, types.StringType, networkRecord.OtherConfig)
	data.OtherConfig = otherConfig
	if diags.HasError() {
		return errors.New("unable to update data for network other_config")
	}
	return nil
}

func updateNetworkFields(ctx context.Context, session *xenapi.Session, networkRef xenapi.NetworkRef, data networkResourceModel) error {
	err := xenapi.Network.SetNameLabel(session, networkRef, data.NameLabel.ValueString())
	if err != nil {
		return errors.New("unable to update network name_label")
	}

	err = xenapi.Network.SetNameDescription(session, networkRef, data.NameDescription.ValueString())
	if err != nil {
		return errors.New("unable to update network name_description")
	}

	err = xenapi.Network.SetMTU(session, networkRef, int(data.MTU.ValueInt64()))
	if err != nil {
		return errors.New("unable to update network mtu")
	}

	otherConfig := make(map[string]string, len(data.OtherConfig.Elements()))
	diags := data.OtherConfig.ElementsAs(ctx, &otherConfig, false)
	if diags.HasError() {
		return errors.New("unable to update network other_config")
	}

	err = xenapi.Network.SetOtherConfig(session, networkRef, otherConfig)
	if err != nil {
		return errors.New("unable to update network other_config")
	}
	return nil
}

type nicDataSourceModel struct {
	NetworkType types.String `tfsdk:"network_type"`
	DataItems   []string     `tfsdk:"data_items"`
}

func unique(items []string) []string {
	slices.Sort(items)
	items = slices.Compact(items)
	return items
}

func getBondNICs(session *xenapi.Session) ([]string, error) {
	var nics []string
	bondRecords, err := xenapi.Bond.GetAllRecords(session)
	if err != nil {
		return nics, errors.New(err.Error())
	}
	var bondDevices []string
	for _, bondRecord := range bondRecords {
		pifRecord, err := xenapi.PIF.GetRecord(session, bondRecord.Master)
		if err != nil {
			return nics, errors.New(err.Error())
		}
		if !slices.Contains(bondDevices, pifRecord.Device) {
			bondDevices = append(bondDevices, pifRecord.Device)
			var bondSlaveDevices []string
			for _, slave := range bondRecord.Slaves {
				record, err := xenapi.PIF.GetRecord(session, slave)
				if err != nil {
					return nics, errors.New(err.Error())
				}
				bondSlaveDevices = append(bondSlaveDevices, record.Device)
			}
			nics = append(nics, getNICNameForBondDevices(bondSlaveDevices))
		}
	}
	return unique(nics), nil
}

func getPhysicalNICs(pifRecords map[xenapi.PIFRef]xenapi.PIFRecord) []string {
	var devices []string
	for _, pifRecord := range pifRecords {
		if pifRecord.Physical {
			devices = append(devices, pifRecord.Device)
		}
	}
	return getNICsNameForDevices(unique(devices), "NIC")
}

func getPhysicalWithoutBondNICs(pifRecords map[xenapi.PIFRef]xenapi.PIFRecord) []string {
	var devices []string
	for _, pifRecord := range pifRecords {
		if pifRecord.Physical && string(pifRecord.BondSlaveOf) == "OpaqueRef:NULL" {
			devices = append(devices, pifRecord.Device)
		}
	}
	return getNICsNameForDevices(unique(devices), "NIC")
}

func getNonPhysicalSRIOVNICs(pifRecords map[xenapi.PIFRef]xenapi.PIFRecord) []string {
	var devices []string
	for _, pifRecord := range pifRecords {
		if pifRecord.Physical && len(pifRecord.SriovPhysicalPIFOf) > 0 && string(pifRecord.BondSlaveOf) == "OpaqueRef:NULL" {
			devices = append(devices, pifRecord.Device)
		}
	}
	return getNICsNameForDevices(unique(devices), "NIC-SR-IOV")
}

func getPhysicalSRIOVNICs(pifRecords map[xenapi.PIFRef]xenapi.PIFRecord, available bool) []string {
	// At lease one of Host in Pool has the PIF with capabilities of "sriov"
	// If available is true, then return the NICs which are not been used by any SR-IOV Network
	var devices []string
	for _, pifRecord := range pifRecords {
		if pifRecord.Physical && slices.Contains(pifRecord.Capabilities, "sriov") {
			if available && len(pifRecord.SriovPhysicalPIFOf) > 0 {
				continue
			} else {
				devices = append(devices, pifRecord.Device)
			}
		}
	}
	return getNICsNameForDevices(unique(devices), "NIC")
}

func getNICsNameForDevices(devices []string, name string) []string {
	// devices := []string{"eth0", "eth1", "eth2"}
	// nics := []string{"NIC 0", "NIC 1", "NIC 2"}
	// nics := []string{"NIC-SR-IOV 0", "NIC-SR-IOV 1", "NIC-SR-IOV 2"}
	var nics []string
	for _, device := range devices {
		if strings.HasPrefix(device, "eth") {
			nics = append(nics, name+" "+strings.Split(device, "eth")[1])
		}
	}
	return nics
}

func getNICNameForBondDevices(devices []string) string {
	// devices := []string{"eth0", "eth1", "eth2"}
	// name := "Bond 0+1+2"
	name := "Bond"
	var deviceNumberStrings []string
	for _, device := range devices {
		if strings.HasPrefix(device, "eth") {
			deviceNumberStrings = append(deviceNumberStrings, strings.Split(device, "eth")[1])
		}
	}
	return name + " " + strings.Join(deviceNumberStrings, "+")
}
