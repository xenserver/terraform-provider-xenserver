package xenserver

import (
	"errors"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"xenapi"
)

type snapshotResourceModel struct {
	NameLabel  types.String `tfsdk:"name_label"`
	VM         types.String `tfsdk:"vm_uuid"`
	WithMemory types.Bool   `tfsdk:"with_memory"`
	UUID       types.String `tfsdk:"uuid"`
	ID         types.String `tfsdk:"id"`
}

func updateSnapshotResourceModel(session *xenapi.Session, record xenapi.VMRecord, data *snapshotResourceModel) error {
	data.NameLabel = types.StringValue(record.NameLabel)
	vmUUID, err := xenapi.VM.GetUUID(session, record.SnapshotOf)
	if err != nil {
		return errors.New(err.Error())
	}
	data.VM = types.StringValue(vmUUID)

	return updateSnapshotResourceModelComputed(record, data)
}

func updateSnapshotResourceModelComputed(record xenapi.VMRecord, data *snapshotResourceModel) error {
	data.UUID = types.StringValue(record.UUID)
	data.ID = types.StringValue(record.UUID)
	if record.PowerState == xenapi.VMPowerStateSuspended {
		data.WithMemory = types.BoolValue(true)
	} else {
		data.WithMemory = types.BoolValue(false)
	}
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
	var vdiRefs []xenapi.VDIRef
	vbdRefs, err := xenapi.VM.GetVBDs(session, ref)
	if err != nil {
		return errors.New(err.Error())
	}
	for _, vbdRef := range vbdRefs {
		vdiRef, err := xenapi.VBD.GetVDI(session, vbdRef)
		if err != nil {
			return errors.New(err.Error())
		}
		vdiRefs = append(vdiRefs, vdiRef)
	}
	for _, vdiRef := range vdiRefs {
		err = xenapi.VDI.Destroy(session, vdiRef)
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
