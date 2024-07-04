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
	VBD      types.String `tfsdk:"vbd_ref"`
	Mode     types.String `tfsdk:"mode"`
	Bootable types.Bool   `tfsdk:"bootable"`
}

var vbdResourceModelAttrTypes = map[string]attr.Type{
	"vdi_uuid": types.StringType,
	"vbd_ref":  types.StringType,
	"mode":     types.StringType,
	"bootable": types.BoolType,
}

func VBDSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"vdi_uuid": schema.StringAttribute{
			MarkdownDescription: "VDI UUID to attach to VBD",
			Required:            true,
		},
		"vbd_ref": schema.StringAttribute{
			MarkdownDescription: "VBD Reference",
			Computed:            true,
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
		return vbdRef, errors.New(err.Error())
	}

	userDevices, err := xenapi.VM.GetAllowedVBDDevices(session, vmRef)
	if err != nil {
		return vbdRef, errors.New(err.Error())
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
		return vbdRef, errors.New(err.Error())
	}

	// plug VBDs if VM is running
	vmPowerState, err := xenapi.VM.GetPowerState(session, vmRef)
	if err != nil {
		return vbdRef, errors.New(err.Error())
	}

	if vmPowerState == xenapi.VMPowerStateRunning {
		err = xenapi.VBD.Plug(session, vbdRef)
		if err != nil {
			return vbdRef, errors.New(err.Error())
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

// sortHardDrive sorts the HardDrive list based on VDI UUID, this is required to compare the VBDs in plan and state
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
	for vdiUUID, planVBD := range planVDIsMap {
		stateVBD, ok := stateVDIsMap[vdiUUID]
		if !ok {
			tflog.Debug(ctx, "---> Create VBD for VDI: "+vdiUUID+" <---")
			_, err = createVBD(planVBD, vmRef, session)
			if err != nil {
				return err
			}
		} else {
			tflog.Debug(ctx, "---> Update VBD "+planVBD.VBD.String()+" for VDI: "+vdiUUID+" <---")
			if planVBD.Mode != stateVBD.Mode {
				err = xenapi.VBD.SetMode(session, xenapi.VBDRef(planVBD.VBD.ValueString()), xenapi.VbdMode(planVBD.Mode.ValueString()))
				if err != nil {
					return errors.New(err.Error())
				}
			}

			if planVBD.Bootable != stateVBD.Bootable {
				err = xenapi.VBD.SetBootable(session, xenapi.VBDRef(planVBD.VBD.ValueString()), planVBD.Bootable.ValueBool())
				if err != nil {
					return errors.New(err.Error())
				}
			}
		}
	}

	// Destroy VBDs that are not in plan
	for vdiUUID, stateVBD := range stateVDIsMap {
		if _, ok := planVDIsMap[vdiUUID]; !ok {
			tflog.Debug(ctx, "---> Destroy VBD:	"+stateVBD.VBD.String())
			err = xenapi.VBD.Destroy(session, xenapi.VBDRef(stateVBD.VBD.ValueString()))
			if err != nil {
				return errors.New(err.Error())
			}
		}
	}

	return nil
}
