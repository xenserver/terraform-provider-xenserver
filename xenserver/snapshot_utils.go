package xenserver

import (
	"context"
	"errors"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"xenapi"
)

type snapshotResourceModel struct {
	NameLabel  types.String `tfsdk:"name_label"`
	VM         types.String `tfsdk:"vm_uuid"`
	WithMemory types.Bool   `tfsdk:"with_memory"`
	Revert     types.Bool   `tfsdk:"revert"`
	RevertVDIs types.Set    `tfsdk:"revert_vdis"`
	UUID       types.String `tfsdk:"uuid"`
	ID         types.String `tfsdk:"id"`
}

func updateSnapshotResourceModel(ctx context.Context, session *xenapi.Session, record xenapi.VMRecord, data *snapshotResourceModel) error {
	data.NameLabel = types.StringValue(record.NameLabel)
	vmUUID, err := xenapi.VM.GetUUID(session, record.SnapshotOf)
	if err != nil {
		return errors.New(err.Error())
	}
	data.VM = types.StringValue(vmUUID)

	return updateSnapshotResourceModelComputed(ctx, session, record, data)
}

func getAllDiskTypeVDIs(session *xenapi.Session, vmRef xenapi.VMRef) ([]xenapi.VDIRef, error) {
	vdiRefs := []xenapi.VDIRef{}
	vbdRefs, err := xenapi.VM.GetVBDs(session, vmRef)
	if err != nil {
		return vdiRefs, errors.New(err.Error())
	}
	for _, vbdRef := range vbdRefs {
		vbdType, err := xenapi.VBD.GetType(session, vbdRef)
		if err != nil {
			return vdiRefs, errors.New(err.Error())
		}
		if vbdType == xenapi.VbdTypeDisk {
			vdiRef, err := xenapi.VBD.GetVDI(session, vbdRef)
			if err != nil {
				return vdiRefs, errors.New(err.Error())
			}
			if string(vdiRef) != "OpaqueRef:NULL" {
				vdiRefs = append(vdiRefs, vdiRef)
			}
		}
	}
	return vdiRefs, nil
}

func updateSnapshotResourceModelComputed(ctx context.Context, session *xenapi.Session, record xenapi.VMRecord, data *snapshotResourceModel) error {
	data.UUID = types.StringValue(record.UUID)
	data.ID = types.StringValue(record.UUID)
	if record.PowerState == xenapi.VMPowerStateSuspended {
		data.WithMemory = types.BoolValue(true)
	} else {
		data.WithMemory = types.BoolValue(false)
	}
	// update the revert_vdis only when revert is true
	var vdiDataList []vdiResourceModel
	if !data.Revert.IsNull() && data.Revert.ValueBool() {
		vdiRefs, err := getAllDiskTypeVDIs(session, record.SnapshotOf)
		if err != nil {
			return err
		}
		for _, vdiRef := range vdiRefs {
			vdiRecord, err := xenapi.VDI.GetRecord(session, vdiRef)
			if err != nil {
				return errors.New(err.Error())
			}
			srUUID, err := xenapi.SR.GetUUID(session, vdiRecord.SR)
			if err != nil {
				return errors.New(err.Error())
			}
			otherConfig, diags := types.MapValueFrom(ctx, types.StringType, vdiRecord.OtherConfig)
			if diags.HasError() {
				return errors.New("unable to access VDI other config")
			}
			vdiData := vdiResourceModel{
				NameLabel:       types.StringValue(vdiRecord.NameLabel),
				NameDescription: types.StringValue(vdiRecord.NameDescription),
				SR:              types.StringValue(srUUID),
				VirtualSize:     types.Int64Value(int64(vdiRecord.VirtualSize)),
				UUID:            types.StringValue(vdiRecord.UUID),
				ID:              types.StringValue(vdiRecord.UUID),
				Type:            types.StringValue(string(vdiRecord.Type)),
				Sharable:        types.BoolValue(vdiRecord.Sharable),
				ReadOnly:        types.BoolValue(vdiRecord.ReadOnly),
				OtherConfig:     otherConfig,
			}
			vdiDataList = append(vdiDataList, vdiData)
		}
	}
	setValue, diags := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: vdiResourceModelAttrTypes}, vdiDataList)
	if diags.HasError() {
		return errors.New("unable to get VDI set value")
	}
	data.RevertVDIs = setValue

	return nil
}

func snapshotResourceModelUpdateCheck(plan snapshotResourceModel, state snapshotResourceModel) error {
	if plan.VM != state.VM {
		return errors.New(`"vm_uuid" doesn't expected to be updated`)
	}
	if plan.WithMemory != state.WithMemory {
		return errors.New(`"with_memory" doesn't expected to be updated`)
	}
	return nil
}

func snapshotResourceModelUpdate(session *xenapi.Session, ref xenapi.VMRef, data snapshotResourceModel) error {
	err := xenapi.VM.SetNameLabel(session, ref, data.NameLabel.ValueString())
	if err != nil {
		return errors.New(err.Error())
	}

	return nil
}

func cleanupSnapshotResource(session *xenapi.Session, ref xenapi.VMRef) error {
	vdiRefs, err := getAllDiskTypeVDIs(session, ref)
	if err != nil {
		return err
	}
	for _, vdiRef := range vdiRefs {
		err := xenapi.VDI.Destroy(session, vdiRef)
		if err != nil && !strings.Contains(err.Error(), "HANDLE_INVALID") {
			return errors.New(err.Error())
		}
	}
	err = xenapi.VM.Destroy(session, ref)
	if err != nil {
		return errors.New(err.Error())
	}
	return nil
}

func revertSnapshot(session *xenapi.Session, ref xenapi.VMRef) error {
	err := xenapi.VM.Revert(session, ref)
	if err != nil {
		return errors.New(err.Error())
	}

	return nil
}

func vmCanBootOnHost(session *xenapi.Session, vmRef xenapi.VMRef, hostRef xenapi.HostRef) bool {
	if string(hostRef) != "OpaqueRef:NULL" {
		err := xenapi.VM.AssertCanBootHere(session, vmRef, hostRef)
		if err == nil {
			return true
		}
	}
	return false
}

func revertPowerState(session *xenapi.Session, record xenapi.VMRecord) error {
	revertPowerState := false
	snapshotState, ok := record.SnapshotInfo["power-state-at-snapshot"]
	if ok && snapshotState == string(xenapi.VMPowerStateRunning) {
		revertPowerState = true
	}
	vmRecord, err := xenapi.VM.GetRecord(session, record.SnapshotOf)
	if err != nil {
		return errors.New(err.Error())
	}
	vmRef, err := xenapi.VM.GetByUUID(session, vmRecord.UUID)
	if err != nil {
		return errors.New(err.Error())
	}
	vmCanBootOnHost := vmCanBootOnHost(session, vmRef, vmRecord.ResidentOn)

	if revertPowerState {
		if vmRecord.PowerState == xenapi.VMPowerStateHalted {
			if vmCanBootOnHost {
				err := xenapi.VM.StartOn(session, vmRef, vmRecord.ResidentOn, false, false)
				if err != nil {
					return errors.New(err.Error())
				}
			} else {
				err := xenapi.VM.Start(session, vmRef, false, false)
				if err != nil {
					return errors.New(err.Error())
				}
			}
		} else if vmRecord.PowerState == xenapi.VMPowerStateSuspended {
			if vmCanBootOnHost {
				err := xenapi.VM.ResumeOn(session, vmRef, vmRecord.ResidentOn, false, false)
				if err != nil {
					return errors.New(err.Error())
				}
			} else {
				err := xenapi.VM.Resume(session, vmRef, false, false)
				if err != nil {
					return errors.New(err.Error())
				}
			}
		}
	}
	return nil
}
