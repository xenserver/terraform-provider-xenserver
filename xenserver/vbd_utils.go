package xenserver

import (
	"context"
	"errors"
	"sort"
	"xenapi"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type vbdResourceModel struct {
	VDI      types.String `tfsdk:"vdi_uuid"`
	Mode     types.String `tfsdk:"mode"`
	Bootable types.Bool   `tfsdk:"bootable"`
}

var vbdResourceModelAttrTypes = map[string]attr.Type{
	"vdi_uuid": types.StringType,
	"mode":     types.StringType,
	"bootable": types.BoolType,
}

func VBDSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"vdi_uuid": schema.StringAttribute{
			MarkdownDescription: "VDI UUID to attach to VBD",
			Required:            true,
		},
		"bootable": schema.BoolAttribute{
			MarkdownDescription: "Set VBD as bootable, Default: false",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"mode": schema.StringAttribute{
			MarkdownDescription: "The mode the VBD should be mounted with, Default: RW",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("RW"),
		},
	}
}

func createVBD(vbd vbdResourceModel, vmRef xenapi.VMRef, session *xenapi.Session) (xenapi.VBDRef, error) {
	var vbdRef xenapi.VBDRef
	vdiRef, err := xenapi.VDI.GetByUUID(session, vbd.VDI.ValueString())
	if err != nil {
		return vbdRef, errors.New("unable to get VDI ref, vdi UUID: " + vbd.VDI.String())
	}

	userDevices, err := xenapi.VM.GetAllowedVBDDevices(session, vmRef)
	if err != nil {
		return vbdRef, errors.New("unable to get allowed VBD devices for vm " + string(vmRef))
	}

	if len(userDevices) == 0 {
		return vbdRef, errors.New("unable to find available devices to attach to vm " + string(vmRef))
	}

	vbdRecord := xenapi.VBDRecord{
		VM:         vmRef,
		VDI:        vdiRef,
		Type:       "Disk",
		Mode:       xenapi.VbdMode(vbd.Mode.ValueString()),
		Bootable:   vbd.Bootable.ValueBool(),
		Empty:      false,
		Userdevice: userDevices[0],
	}

	vbdRef, err = xenapi.VBD.Create(session, vbdRecord)
	if err != nil {
		return vbdRef, errors.New("unable to create VBD, vdi UUID: " + vbd.VDI.String())
	}

	// plug VBDs if VM is running
	vmPowerState, err := xenapi.VM.GetPowerState(session, vmRef)
	if err != nil {
		return vbdRef, errors.New("unable to get VM power state, vm ref: " + string(vmRef))
	}

	if vmPowerState == xenapi.VMPowerStateRunning {
		err = xenapi.VBD.Plug(session, vbdRef)
		if err != nil {
			return vbdRef, errors.New("unable to plug VBD, vdi UUID: " + vbd.VDI.String())
		}
	}

	return vbdRef, nil
}

func createVBDs(ctx context.Context, data vmResourceModel, vmRef xenapi.VMRef, session *xenapi.Session) ([]xenapi.VBDRef, error) {
	elements := make([]vbdResourceModel, 0, len(data.HardDrive.Elements()))
	diags := data.HardDrive.ElementsAs(ctx, &elements, false)
	if diags.HasError() {
		return nil, errors.New("unable to get HardDrive elements")
	}

	var vbdRefs []xenapi.VBDRef
	for _, vbd := range elements {
		vbdRef, err := createVBD(vbd, vmRef, session)
		if err != nil {
			return nil, err
		}
		vbdRefs = append(vbdRefs, vbdRef)
	}
	return vbdRefs, nil
}

func sortHardDrive(ctx context.Context, unSortedList basetypes.ListValue) (basetypes.ListValue, error) {
	var listValue basetypes.ListValue
	vbdList := make([]vbdResourceModel, 0, len(unSortedList.Elements()))
	diags := unSortedList.ElementsAs(ctx, &vbdList, false)
	if diags.HasError() {
		return listValue, errors.New("unable to get VBD list")
	}

	sort.Slice(vbdList, func(i, j int) bool {
		return vbdList[i].VDI.ValueString() < vbdList[j].VDI.ValueString()
	})

	listValue, diags = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: vbdResourceModelAttrTypes}, vbdList)
	if diags.HasError() {
		return listValue, errors.New("unable to get VBD list value")
	}

	return listValue, nil
}

func updateVBDs(ctx context.Context, plan vmResourceModel, state vmResourceModel, vmRef xenapi.VMRef, session *xenapi.Session) error {
	// Get VBDs from plan and state
	planVBDs := make([]vbdResourceModel, 0, len(state.HardDrive.Elements()))
	diags := plan.HardDrive.ElementsAs(ctx, &planVBDs, false)
	if diags.HasError() {
		return errors.New("unable to get VBDs in plan data")
	}

	stateVBDs := make([]vbdResourceModel, 0, len(state.HardDrive.Elements()))
	diags = state.HardDrive.ElementsAs(ctx, &stateVBDs, false)
	if diags.HasError() {
		return errors.New("unable to get VBDs in state data")
	}

	var err error
	planVDIsMap := make(map[string]vbdResourceModel)
	for _, vbd := range planVBDs {
		planVDIsMap[vbd.VDI.ValueString()] = vbd
	}

	stateVDIsMap := make(map[string]vbdResourceModel)
	for _, vbd := range stateVBDs {
		stateVDIsMap[vbd.VDI.ValueString()] = vbd
	}

	// Create VBDs that are in plan but not in state, Update VBDs if already exists and attributes changed
	for vdiUUID, vbd := range planVDIsMap {
		_, ok := stateVDIsMap[vdiUUID]
		if !ok {
			tflog.Debug(ctx, "---> Create VBD for VDI: "+vdiUUID+" <---")
			_, err = createVBD(vbd, vmRef, session)
			if err != nil {
				return err
			}
		} else {
			vbdRef, err := getVBDRef(session, vdiUUID, vmRef)
			if err != nil {
				return err
			}

			tflog.Debug(ctx, "---> Update VBD "+string(vbdRef)+" for VDI: "+vdiUUID+" <---")

			if planVDIsMap[vdiUUID].Mode != stateVDIsMap[vdiUUID].Mode {
				err = xenapi.VBD.SetMode(session, vbdRef, xenapi.VbdMode(planVDIsMap[vdiUUID].Mode.ValueString()))
				if err != nil {
					return errors.New(err.Error())
				}
			}

			if planVDIsMap[vdiUUID].Bootable != stateVDIsMap[vdiUUID].Bootable {
				err = xenapi.VBD.SetBootable(session, vbdRef, planVDIsMap[vdiUUID].Bootable.ValueBool())
				if err != nil {
					return errors.New(err.Error())
				}
			}
		}
	}

	// Destroy VBDs that are not in plan
	for vdiUUID := range stateVDIsMap {
		if _, ok := planVDIsMap[vdiUUID]; !ok {
			tflog.Debug(ctx, "---> Destroy VBD for VDI: "+vdiUUID+" <---")
			err = removeVBD(session, vdiUUID, vmRef)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
func getVBDRef(session *xenapi.Session, vdiUUID string, vmRef xenapi.VMRef) (xenapi.VBDRef, error) {
	var vbdRef xenapi.VBDRef

	VDIRef, err := xenapi.VDI.GetByUUID(session, vdiUUID)
	if err != nil {
		return vbdRef, errors.New("unable to get VDI ref, vdi UUID: " + vdiUUID)
	}

	VBDRefs, err := xenapi.VDI.GetVBDs(session, VDIRef)
	if err != nil {
		return vbdRef, errors.New("unable to get VBDs for VDI, vdi UUID: " + vdiUUID)
	}

	for _, vbdRef := range VBDRefs {
		vbdRecord, err := xenapi.VBD.GetRecord(session, vbdRef)
		if err != nil {
			return vbdRef, errors.New("unable to get VBD record, vbd ref: " + string(vbdRef))
		}

		if vbdRecord.VM == vmRef {
			return vbdRef, nil
		}
	}

	return vbdRef, nil
}

func removeVBD(session *xenapi.Session, vdiUUID string, vmRef xenapi.VMRef) error {
	vbdRef, err := getVBDRef(session, vdiUUID, vmRef)
	if err != nil {
		return err
	}

	err = xenapi.VBD.Destroy(session, vbdRef)
	if err != nil {
		return errors.New("unable to destroy VBD, vbd ref: " + string(vbdRef))
	}
	return nil
}
