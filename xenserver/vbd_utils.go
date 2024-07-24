package xenserver

import (
	"context"
	"errors"
	"xenapi"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
			MarkdownDescription: "VDI UUID to attach to VBD, Note that using the same VDI for multiple VBDs is not supported",
			Required:            true,
		},
		"vbd_ref": schema.StringAttribute{
			Computed: true,
		},
		"bootable": schema.BoolAttribute{
			MarkdownDescription: "Set VBD as bootable, Default: false",
			Optional:            true,
			Computed:            true,
		},
		"mode": schema.StringAttribute{
			MarkdownDescription: "The mode the VBD should be mounted with, Default: RW",
			Optional:            true,
			Computed:            true,
			Validators: []validator.String{
				stringvalidator.OneOf("RO", "RW"),
			},
		},
	}
}

func setVBDDefaults(vbd *vbdResourceModel) {
	// Work around for https://github.com/hashicorp/terraform-plugin-framework/issues/726
	if vbd.Mode.IsUnknown() {
		vbd.Mode = types.StringValue("RW")
	}

	if vbd.Bootable.IsUnknown() {
		vbd.Bootable = types.BoolValue(false)
	}
}

func createVBD(vbd vbdResourceModel, vmRef xenapi.VMRef, session *xenapi.Session) error {
	var vbdRef xenapi.VBDRef
	vdiRef, err := xenapi.VDI.GetByUUID(session, vbd.VDI.ValueString())
	if err != nil {
		return errors.New(err.Error())
	}

	userDevices, err := xenapi.VM.GetAllowedVBDDevices(session, vmRef)
	if err != nil {
		return errors.New(err.Error())
	}

	if len(userDevices) == 0 {
		return errors.New("unable to find available vbd devices to attach to vm " + string(vmRef))
	}

	setVBDDefaults(&vbd)

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
		return errors.New(err.Error())
	}

	// plug VBDs if VM is running
	vmPowerState, err := xenapi.VM.GetPowerState(session, vmRef)
	if err != nil {
		return errors.New(err.Error())
	}

	if vmPowerState == xenapi.VMPowerStateRunning {
		err = xenapi.VBD.Plug(session, vbdRef)
		if err != nil {
			return errors.New(err.Error())
		}
	}

	return nil
}

func createVBDs(ctx context.Context, data vmResourceModel, vmRef xenapi.VMRef, session *xenapi.Session) error {
	elements := make([]vbdResourceModel, 0, len(data.HardDrive.Elements()))
	diags := data.HardDrive.ElementsAs(ctx, &elements, false)
	if diags.HasError() {
		return errors.New("unable to get HardDrive elements")
	}

	for _, vbd := range elements {
		tflog.Debug(ctx, "---> Create VBD with VDI: "+vbd.VDI.String()+"  Mode: "+vbd.Mode.String()+"  Bootable: "+vbd.Bootable.String())
		err := createVBD(vbd, vmRef, session)
		if err != nil {
			return err
		}
	}
	return nil
}

func updateVBDs(ctx context.Context, plan vmResourceModel, state vmResourceModel, vmRef xenapi.VMRef, session *xenapi.Session) error {
	planHardDrives := make([]vbdResourceModel, 0, len(state.HardDrive.Elements()))
	diags := plan.HardDrive.ElementsAs(ctx, &planHardDrives, false)
	if diags.HasError() {
		return errors.New("unable to get HardDrives in plan data")
	}

	stateHardDrives := make([]vbdResourceModel, 0, len(state.HardDrive.Elements()))
	diags = state.HardDrive.ElementsAs(ctx, &stateHardDrives, false)
	if diags.HasError() {
		return errors.New("unable to get HardDrives in state data")
	}

	var err error
	planHardDrivesMap := make(map[string]vbdResourceModel)
	for _, vbd := range planHardDrives {
		planHardDrivesMap[vbd.VDI.ValueString()] = vbd
	}

	stateHardDrivesMap := make(map[string]vbdResourceModel)
	for _, vbd := range stateHardDrives {
		stateHardDrivesMap[vbd.VDI.ValueString()] = vbd
	}

	// Destroy VBDs that are not in plan
	for vdiUUID, stateVBD := range stateHardDrivesMap {
		if _, ok := planHardDrivesMap[vdiUUID]; !ok {
			tflog.Debug(ctx, "---> Destroy VBD:	"+stateVBD.VBD.String())
			err = xenapi.VBD.Destroy(session, xenapi.VBDRef(stateVBD.VBD.ValueString()))
			if err != nil {
				return errors.New(err.Error())
			}
		}
	}

	// Create VBDs that are in plan but not in state, Update VBDs if already exists and attributes changed
	for vdiUUID, planVBD := range planHardDrivesMap {
		stateVBD, ok := stateHardDrivesMap[vdiUUID]
		if !ok {
			tflog.Debug(ctx, "---> Create VBD for VDI: "+vdiUUID+" <---")
			err = createVBD(planVBD, vmRef, session)
			if err != nil {
				return err
			}
		} else {
			// Update VBD if attributes changed
			setVBDDefaults(&planVBD)

			if !planVBD.Mode.Equal(stateVBD.Mode) {
				tflog.Debug(ctx, "---> VBD.SetMode:	"+planVBD.Mode.String())
				err = xenapi.VBD.SetMode(session, xenapi.VBDRef(stateVBD.VBD.ValueString()), xenapi.VbdMode(planVBD.Mode.ValueString()))
				if err != nil {
					return errors.New(err.Error())
				}
			}

			if !planVBD.Bootable.Equal(stateVBD.Bootable) {
				tflog.Debug(ctx, "---> VBD.SetBootable:	"+planVBD.Bootable.String())
				err = xenapi.VBD.SetBootable(session, xenapi.VBDRef(stateVBD.VBD.ValueString()), planVBD.Bootable.ValueBool())
				if err != nil {
					return errors.New(err.Error())
				}
			}
		}
	}

	return nil
}
