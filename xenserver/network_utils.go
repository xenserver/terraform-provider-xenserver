// Copyright © 2026. Citrix Systems, Inc. All Rights Reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

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
	MTU                types.Int32  `tfsdk:"mtu"`
	OtherConfig        types.Map    `tfsdk:"other_config"`
	Bridge             types.String `tfsdk:"bridge"`
	Managed            types.Bool   `tfsdk:"managed"`
	Blobs              types.Map    `tfsdk:"blobs"`
	Tags               types.List   `tfsdk:"tags"`
	DefaultLockingMode types.String `tfsdk:"default_locking_mode"`
	AssignedIps        types.Map    `tfsdk:"assigned_ips"`
	Purpose            types.List   `tfsdk:"purpose"`
}

func updateNetworkRecordData(ctx context.Context, session *xenapi.Session, record xenapi.NetworkRecord, data *networkRecordData) error {
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
	vifUUIDs, err := getVIFUUIDs(session, record.VIFs)
	if err != nil {
		return err
	}
	data.VIFs, diags = types.ListValueFrom(ctx, types.StringType, vifUUIDs)
	if diags.HasError() {
		return errors.New("unable to read network VIFs")
	}
	pifUUIDs, err := getPIFUUIDs(session, record.PIFs)
	if err != nil {
		return err
	}
	data.PIFs, diags = types.ListValueFrom(ctx, types.StringType, pifUUIDs)
	if diags.HasError() {
		return errors.New("unable to read network PIFs")
	}

	mtu, err := ToInt32(record.MTU)
	if err != nil {
		return err
	}
	data.MTU = types.Int32Value(mtu)
	data.OtherConfig, diags = types.MapValueFrom(ctx, types.StringType, record.OtherConfig)
	if diags.HasError() {
		return errors.New("unable to read network other config")
	}
	data.Bridge = types.StringValue(record.Bridge)
	data.Managed = types.BoolValue(record.Managed)
	blobs, err := getBlobUUIDsMap(session, record.Blobs)
	if err != nil {
		return err
	}
	data.Blobs, diags = types.MapValueFrom(ctx, types.StringType, blobs)
	if diags.HasError() {
		return errors.New("unable to read network blobs")
	}
	data.Tags, diags = types.ListValueFrom(ctx, types.StringType, record.Tags)
	if diags.HasError() {
		return errors.New("unable to read network tags")
	}
	data.DefaultLockingMode = types.StringValue(string(record.DefaultLockingMode))
	assignedIps, err := getVIFUUIDsMap(session, record.AssignedIps)
	if err != nil {
		return err
	}
	data.AssignedIps, diags = types.MapValueFrom(ctx, types.StringType, assignedIps)
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
	MTU             types.Int32  `tfsdk:"mtu"`
	Managed         types.Bool   `tfsdk:"managed"`
	OtherConfig     types.Map    `tfsdk:"other_config"`
	Tag             types.Int32  `tfsdk:"vlan_tag"`
	NIC             types.String `tfsdk:"nic"`
	UUID            types.String `tfsdk:"uuid"`
	ID              types.String `tfsdk:"id"`
}

type vlanCreateParams struct {
	PifRef     xenapi.PIFRef
	NetworkRef xenapi.NetworkRef
	Tag        int
}

func getNetworkCreateParams(ctx context.Context, data vlanResourceModel) (xenapi.NetworkRecord, error) {
	var record xenapi.NetworkRecord
	record.NameLabel = data.NameLabel.ValueString()
	record.NameDescription = data.NameDescription.ValueString()
	record.MTU = int(data.MTU.ValueInt32())
	record.Managed = data.Managed.ValueBool()
	diags := data.OtherConfig.ElementsAs(ctx, &record.OtherConfig, false)
	if diags.HasError() {
		return record, errors.New("unable to access vlan other config")
	}

	return record, nil
}

func getPifRefsForNIC(session *xenapi.Session, nic string) ([]xenapi.PIFRef, error) {
	// nic eg. 1. NIC 0 2. NIC-SR-IOV 0 3. Bond 0+1+2
	var pifRefs []xenapi.PIFRef
	pifRecords, err := xenapi.PIF.GetAllRecords(session)
	if err != nil {
		return pifRefs, errors.New(err.Error())
	}
	// identifier is the trailing part of the nic name: "0" for "NIC 0"/"NIC-SR-IOV 0",
	// or the sorted member numbers "0+1+2" for "Bond 0+1+2".
	identifier := strings.Split(nic, " ")[1]
	isSriov := strings.HasPrefix(nic, "NIC-SR-IOV")
	isBond := strings.HasPrefix(nic, "Bond")

	for ref, pifRecord := range pifRecords {
		var match bool
		switch {
		case isSriov:
			if !pifRecord.Physical && len(pifRecord.SriovLogicalPIFOf) > 0 {
				number, err := getNICNumber(session, pifRecord)
				if err != nil {
					return pifRefs, err
				}
				match = number == identifier
			}
		case isBond:
			if !pifRecord.Physical && len(pifRecord.BondMasterOf) > 0 {
				bondRecord, err := xenapi.Bond.GetRecord(session, pifRecord.BondMasterOf[0])
				if err != nil {
					return pifRefs, errors.New(err.Error())
				}
				numbers, err := getBondSlaveNICNumbers(session, bondRecord.Slaves)
				if err != nil {
					return pifRefs, err
				}
				slices.Sort(numbers)
				match = strings.Join(numbers, "+") == identifier
			}
		default: // "NIC N"
			if pifRecord.Physical && string(pifRecord.BondSlaveOf) == "OpaqueRef:NULL" {
				number, err := getNICNumber(session, pifRecord)
				if err != nil {
					return pifRefs, err
				}
				match = number == identifier
			}
		}
		if match {
			pifRefs = append(pifRefs, ref)
		}
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
	params.Tag = int(data.Tag.ValueInt32())

	return params, nil
}

func getNICFromPIF(session *xenapi.Session, pifRecord xenapi.PIFRecord) (string, error) {
	// pifRecord is the VLAN master PIF of an external network. Resolve the underlying
	// tagged PIF to derive the NIC name. eg. NIC 0, NIC-SR-IOV 0, Bond 0+1+2.
	vlanRecord, err := xenapi.VLAN.GetRecord(session, pifRecord.VLANMasterOf)
	if err != nil {
		return "", errors.New(err.Error())
	}
	taggedPifRecord, err := xenapi.PIF.GetRecord(session, vlanRecord.TaggedPIF)
	if err != nil {
		return "", errors.New(err.Error())
	}

	// Bond: name from the bond members' NIC numbers.
	if len(taggedPifRecord.BondMasterOf) > 0 {
		bondRecord, err := xenapi.Bond.GetRecord(session, taggedPifRecord.BondMasterOf[0])
		if err != nil {
			return "", errors.New(err.Error())
		}
		numbers, err := getBondSlaveNICNumbers(session, bondRecord.Slaves)
		if err != nil {
			return "", err
		}
		return getNICNameForBondNumbers(numbers), nil
	}

	// Physical NIC or SR-IOV: number from the (effective) physical network bridge.
	number, err := getNICNumber(session, taggedPifRecord)
	if err != nil {
		return "", err
	}
	if len(taggedPifRecord.SriovLogicalPIFOf) > 0 {
		return "NIC-SR-IOV " + number, nil
	}
	return "NIC " + number, nil
}

func updateVlanResourceModel(ctx context.Context, session *xenapi.Session, record xenapi.NetworkRecord, data *vlanResourceModel) error {
	data.NameLabel = types.StringValue(record.NameLabel)
	pifRecord, err := xenapi.PIF.GetRecord(session, record.PIFs[0])
	if err != nil {
		return errors.New(err.Error())
	}

	vlan, err := ToInt32(pifRecord.VLAN)
	if err != nil {
		return err
	}
	data.Tag = types.Int32Value(vlan)
	nicName, err := getNICFromPIF(session, pifRecord)
	if err != nil {
		return err
	}
	data.NIC = types.StringValue(nicName)

	return updateVlanResourceModelComputed(ctx, record, data)
}

func updateVlanResourceModelComputed(ctx context.Context, record xenapi.NetworkRecord, data *vlanResourceModel) error {
	data.UUID = types.StringValue(record.UUID)
	data.ID = types.StringValue(record.UUID)
	data.NameDescription = types.StringValue(record.NameDescription)
	mtu, err := ToInt32(record.MTU)
	if err != nil {
		return err
	}
	data.MTU = types.Int32Value(mtu)
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
	mtu := int(data.MTU.ValueInt32())
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

func cleanupVlanResource(session *xenapi.Session, ref xenapi.NetworkRef) error {
	networkRecord, err := xenapi.Network.GetRecord(session, ref)
	if err != nil {
		return errors.New(err.Error())
	}
	for _, pifRef := range networkRecord.PIFs {
		pifRecord, err := xenapi.PIF.GetRecord(session, pifRef)
		if err != nil {
			return errors.New(err.Error())
		}
		err = xenapi.VLAN.Destroy(session, pifRecord.VLANMasterOf)
		if err != nil {
			return errors.New(err.Error())
		}
	}
	err = xenapi.Network.Destroy(session, ref)
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

// getNICNumber returns the NIC number for a PIF, derived from the bridge name of
// the physical network it maps to (eg. bridge "xenbr0" -> "0"). Since the network
// rename feature the device name (eg. "eno12419np2") is no longer "ethN", so the
// number is taken from the network bridge instead. For SR-IOV logical PIFs the
// effective physical PIF's network is used.
func getNICNumber(session *xenapi.Session, pifRecord xenapi.PIFRecord) (string, error) {
	networkRef := pifRecord.Network
	if !pifRecord.Physical && len(pifRecord.SriovLogicalPIFOf) > 0 {
		physicalPIF, err := xenapi.NetworkSriov.GetPhysicalPIF(session, pifRecord.SriovLogicalPIFOf[0])
		if err != nil {
			return "", errors.New(err.Error())
		}
		networkRef, err = xenapi.PIF.GetNetwork(session, physicalPIF)
		if err != nil {
			return "", errors.New(err.Error())
		}
	}
	bridge, err := xenapi.Network.GetBridge(session, networkRef)
	if err != nil {
		return "", errors.New(err.Error())
	}
	return strings.TrimPrefix(bridge, "xenbr"), nil
}

func getBondSlaveNICNumbers(session *xenapi.Session, bondSlaves []xenapi.PIFRef) ([]string, error) {
	// Each bond slave keeps its own pool-wide network, so its NIC number is taken
	// from that network's bridge. eg. slaves on "xenbr2" and "xenbr3" -> ["2", "3"].
	var numbers []string
	for _, slave := range bondSlaves {
		record, err := xenapi.PIF.GetRecord(session, slave)
		if err != nil {
			return numbers, errors.New(err.Error())
		}
		number, err := getNICNumber(session, record)
		if err != nil {
			return numbers, err
		}
		numbers = append(numbers, number)
	}
	return numbers, nil
}

func getNICNameForBondNumbers(numbers []string) string {
	// numbers := []string{"2", "3"} -> "Bond 2+3"
	slices.Sort(numbers)
	return "Bond " + strings.Join(numbers, "+")
}

func getBondNICs(session *xenapi.Session) ([]string, error) {
	var nics []string
	bondRecords, err := xenapi.Bond.GetAllRecords(session)
	if err != nil {
		return nics, errors.New(err.Error())
	}
	for _, bondRecord := range bondRecords {
		numbers, err := getBondSlaveNICNumbers(session, bondRecord.Slaves)
		if err != nil {
			return nics, err
		}
		nics = append(nics, getNICNameForBondNumbers(numbers))
	}
	return unique(nics), nil
}

func getNICNames(session *xenapi.Session, pifRecords []xenapi.PIFRecord, name string) ([]string, error) {
	// name eg. "NIC" or "NIC-SR-IOV"; the number for each PIF is taken from its
	// network bridge, then duplicates (same NIC across pool hosts) are removed.
	var nics []string
	for _, pifRecord := range pifRecords {
		number, err := getNICNumber(session, pifRecord)
		if err != nil {
			return nics, err
		}
		nics = append(nics, name+" "+number)
	}
	return unique(nics), nil
}

func getPhysicalNICs(session *xenapi.Session, pifRecords map[xenapi.PIFRef]xenapi.PIFRecord) ([]string, error) {
	var physicalPIFs []xenapi.PIFRecord
	for _, pifRecord := range pifRecords {
		if pifRecord.Physical {
			physicalPIFs = append(physicalPIFs, pifRecord)
		}
	}
	return getNICNames(session, physicalPIFs, "NIC")
}

func getPhysicalWithoutBondNICs(session *xenapi.Session, pifRecords map[xenapi.PIFRef]xenapi.PIFRecord) ([]string, error) {
	var physicalPIFs []xenapi.PIFRecord
	for _, pifRecord := range pifRecords {
		if pifRecord.Physical && string(pifRecord.BondSlaveOf) == "OpaqueRef:NULL" {
			physicalPIFs = append(physicalPIFs, pifRecord)
		}
	}
	return getNICNames(session, physicalPIFs, "NIC")
}

func getNonPhysicalSRIOVNICs(session *xenapi.Session, pifRecords map[xenapi.PIFRef]xenapi.PIFRecord) ([]string, error) {
	var sriovPIFs []xenapi.PIFRecord
	for _, pifRecord := range pifRecords {
		if pifRecord.Physical && len(pifRecord.SriovPhysicalPIFOf) > 0 && string(pifRecord.BondSlaveOf) == "OpaqueRef:NULL" {
			sriovPIFs = append(sriovPIFs, pifRecord)
		}
	}
	return getNICNames(session, sriovPIFs, "NIC-SR-IOV")
}

func getPhysicalSRIOVNICs(session *xenapi.Session, pifRecords map[xenapi.PIFRef]xenapi.PIFRecord, available bool) ([]string, error) {
	// At lease one of Host in Pool has the PIF with capabilities of "sriov"
	// If available is true, then return the NICs which are not been used by any SR-IOV Network
	var sriovPIFs []xenapi.PIFRecord
	for _, pifRecord := range pifRecords {
		if pifRecord.Physical && slices.Contains(pifRecord.Capabilities, "sriov") {
			if available && len(pifRecord.SriovPhysicalPIFOf) > 0 {
				continue
			} else {
				sriovPIFs = append(sriovPIFs, pifRecord)
			}
		}
	}
	return getNICNames(session, sriovPIFs, "NIC")
}
