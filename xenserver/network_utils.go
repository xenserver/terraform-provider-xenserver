package xenserver

import (
	"context"
	"errors"
	"fmt"
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

type vlanResourceModel struct {
	NameLabel       types.String `tfsdk:"name_label"`
	NameDescription types.String `tfsdk:"name_description"`
	MTU             types.Int64  `tfsdk:"mtu"`
	Managed         types.Bool   `tfsdk:"managed"`
	OtherConfig     types.Map    `tfsdk:"other_config"`
	Tag             types.Int64  `tfsdk:"vlan_tag"`
	NIC             types.String `tfsdk:"nic"`
	UUID            types.String `tfsdk:"id"`
}

type vlanCreateParams struct {
	PifRef     xenapi.PIFRef
	NetworkRef xenapi.NetworkRef
	Tag        int
}

func checkMTU(mtu int) error {
	if mtu <= 0 {
		return errors.New("MTU value must above 0 ")
	}
	return nil
}

func getNetworkCreateParams(ctx context.Context, data vlanResourceModel) (xenapi.NetworkRecord, error) {
	var record xenapi.NetworkRecord
	record.NameLabel = data.NameLabel.ValueString()
	record.NameDescription = data.NameDescription.ValueString()
	record.MTU = int(data.MTU.ValueInt64())
	err := checkMTU(record.MTU)
	if err != nil {
		return record, err
	}
	record.Managed = data.Managed.ValueBool()
	diags := data.OtherConfig.ElementsAs(ctx, &record.OtherConfig, false)
	if diags.HasError() {
		return record, errors.New("unable to access vlan other config")
	}

	return record, nil
}

func getBondNICDevice(session *xenapi.Session, nic string) (string, error) {
	// nic eg. "Bond 0+1+2" return eg. "bond0"
	// slavesDevices eg. ["0", "1", "2"]
	slavesDevices := strings.Split(strings.Split(nic, " ")[1], "+")
	bondRecords, err := xenapi.Bond.GetAllRecords(session)
	if err != nil {
		return "", errors.New(err.Error())
	}
	for _, bondRecord := range bondRecords {
		devices := []string{}
		for _, slave := range bondRecord.Slaves {
			pifRecord, err := xenapi.PIF.GetRecord(session, slave)
			if err != nil {
				return "", errors.New(err.Error())
			}
			devices = append(devices, strings.Split(pifRecord.Device, "eth")[1])
		}
		slices.Sort(devices)
		if slices.Equal(slavesDevices, devices) {
			record, err := xenapi.PIF.GetRecord(session, bondRecord.Master)
			if err != nil {
				return "", errors.New(err.Error())
			}
			return record.Device, nil
		}
	}
	return "", fmt.Errorf("unable to find device for %s", nic)
}

func getPifRefsForNIC(session *xenapi.Session, nic string) ([]xenapi.PIFRef, error) {
	// nic eg. 1. NIC 0 2. NIC-SR-IOV 0 3. Bond 0+1+2
	var pifRefs []xenapi.PIFRef
	pifRecords, err := xenapi.PIF.GetAllRecords(session)
	if err != nil {
		return pifRefs, errors.New(err.Error())
	}
	device := "eth" + strings.Split(nic, " ")[1]
	if strings.HasPrefix(nic, "Bond") {
		device, err = getBondNICDevice(session, nic)
		if err != nil {
			return pifRefs, err
		}
	}
	uuids := []string{}
	for _, pifRecord := range pifRecords {
		if pifRecord.Device == device && ((strings.HasPrefix(nic, "NIC-SR-IOV") && !pifRecord.Physical && len(pifRecord.SriovLogicalPIFOf) > 0) ||
			(strings.HasPrefix(nic, "NIC") && pifRecord.Physical && string(pifRecord.BondSlaveOf) == "OpaqueRef:NULL") ||
			(strings.HasPrefix(nic, "Bond") && !pifRecord.Physical && len(pifRecord.BondMasterOf) > 0)) {
			uuids = append(uuids, pifRecord.UUID)
		}
	}
	for _, uuid := range uuids {
		ref, err := xenapi.PIF.GetByUUID(session, uuid)
		if err != nil {
			return pifRefs, errors.New(err.Error())
		}
		pifRefs = append(pifRefs, ref)
	}

	return pifRefs, nil
}

func getVlanCreateParams(session *xenapi.Session, data vlanResourceModel, networkRef xenapi.NetworkRef) (vlanCreateParams, error) {
	var params vlanCreateParams
	pifRefs, err := getPifRefsForNIC(session, data.NIC.ValueString())
	if err != nil {
		return params, err
	}
	if len(pifRefs) == 0 {
		return params, errors.New("unable to find PIF for NIC")
	}
	params.PifRef = pifRefs[0]
	params.NetworkRef = networkRef
	params.Tag = int(data.Tag.ValueInt64())

	return params, nil
}

func getNICFromPIF(session *xenapi.Session, pifRecord xenapi.PIFRecord) (string, error) {
	// return eg. NIC 0, NIC-SR-IOV 0, Bond 0+1+2
	name := ""
	if strings.HasPrefix(pifRecord.Device, "eth") {
		index := strings.Split(pifRecord.Device, "eth")[1]
		name = "NIC " + index
		if !pifRecord.Physical && pifRecord.VLANMasterOf != "OpaqueRef:NULL" {
			vlanRecord, err := xenapi.VLAN.GetRecord(session, pifRecord.VLANMasterOf)
			if err != nil {
				return name, errors.New(err.Error())
			}
			taggedPifRecord, err := xenapi.PIF.GetRecord(session, vlanRecord.TaggedPIF)
			if err != nil {
				return name, errors.New(err.Error())
			}
			if len(taggedPifRecord.SriovLogicalPIFOf) > 0 {
				name = "NIC-SR-IOV " + index
			}
		}
	} else if strings.HasPrefix(pifRecord.Device, "bond") {
		vlanRecord, err := xenapi.VLAN.GetRecord(session, pifRecord.VLANMasterOf)
		if err != nil {
			return name, errors.New(err.Error())
		}
		taggedPifRecord, err := xenapi.PIF.GetRecord(session, vlanRecord.TaggedPIF)
		if err != nil {
			return name, errors.New(err.Error())
		}
		bondRecord, err := xenapi.Bond.GetRecord(session, taggedPifRecord.BondMasterOf[0])
		if err != nil {
			return name, errors.New(err.Error())
		}
		bondSlaveDevices, err := getBondSlaveDevices(session, bondRecord.Slaves)
		if err != nil {
			return name, err
		}
		name = getNICNameForBondDevices(bondSlaveDevices)
	}

	return name, nil
}

func updateVlanResourceModel(ctx context.Context, session *xenapi.Session, record xenapi.NetworkRecord, data *vlanResourceModel) error {
	data.NameLabel = types.StringValue(record.NameLabel)
	pifRecord, err := xenapi.PIF.GetRecord(session, record.PIFs[0])
	if err != nil {
		return errors.New(err.Error())
	}
	data.Tag = types.Int64Value(int64(pifRecord.VLAN))
	nicName, err := getNICFromPIF(session, pifRecord)
	if err != nil {
		return err
	}
	data.NIC = types.StringValue(nicName)

	return updateVlanResourceModelComputed(ctx, record, data)
}

func updateVlanResourceModelComputed(ctx context.Context, record xenapi.NetworkRecord, data *vlanResourceModel) error {
	data.UUID = types.StringValue(record.UUID)
	data.NameDescription = types.StringValue(record.NameDescription)
	data.MTU = types.Int64Value(int64(record.MTU))
	data.Managed = types.BoolValue(record.Managed)
	var diags diag.Diagnostics
	data.OtherConfig, diags = types.MapValueFrom(ctx, types.StringType, record.OtherConfig)
	if diags.HasError() {
		return errors.New("unable to update data for network_vlan other_config")
	}

	return nil
}

func vlanResourceModelUpdateCheck(data vlanResourceModel, dataState vlanResourceModel) error {
	if data.NIC != dataState.NIC {
		return errors.New(`"nic" doesn't expected to be updated`)
	}
	if data.Tag != dataState.Tag {
		return errors.New(`"vlan_tag" doesn't expected to be updated`)
	}
	if data.Managed != dataState.Managed {
		return errors.New(`"managed" doesn't expected to be updated`)
	}
	return nil
}

func vlanResourceModelUpdate(ctx context.Context, session *xenapi.Session, ref xenapi.NetworkRef, data vlanResourceModel) error {
	err := xenapi.Network.SetNameLabel(session, ref, data.NameLabel.ValueString())
	if err != nil {
		return errors.New(err.Error())
	}
	err = xenapi.Network.SetNameDescription(session, ref, data.NameDescription.ValueString())
	if err != nil {
		return errors.New(err.Error())
	}
	mtu := int(data.MTU.ValueInt64())
	err = checkMTU(mtu)
	if err != nil {
		return err
	}
	err = xenapi.Network.SetMTU(session, ref, mtu)
	if err != nil {
		return errors.New(err.Error())
	}
	otherConfig := make(map[string]string)
	diags := data.OtherConfig.ElementsAs(ctx, &otherConfig, false)
	if diags.HasError() {
		return errors.New("unable to access network other config")
	}
	err = xenapi.Network.SetOtherConfig(session, ref, otherConfig)
	if err != nil {
		return errors.New(err.Error())
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

func getBondSlaveDevices(session *xenapi.Session, bondSlaves []xenapi.PIFRef) ([]string, error) {
	var bondSlaveDevices []string
	for _, slave := range bondSlaves {
		record, err := xenapi.PIF.GetRecord(session, slave)
		if err != nil {
			return bondSlaveDevices, errors.New(err.Error())
		}
		bondSlaveDevices = append(bondSlaveDevices, record.Device)
	}
	return bondSlaveDevices, nil
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
			bondSlaveDevices, err := getBondSlaveDevices(session, bondRecord.Slaves)
			if err != nil {
				return nics, err
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
	slices.Sort(deviceNumberStrings)
	return name + " " + strings.Join(deviceNumberStrings, "+")
}
