package xenserver

import (
	"context"
	"errors"
	"sort"
	"strings"
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
			MarkdownDescription: "VDI UUID to attach to VBD." + "<br />" +
				"**Note**: Using the same VDI UUID for multiple VBDs is not supported.",
			Required: true,
		},
		"vbd_ref": schema.StringAttribute{
			Computed: true,
		},
		"bootable": schema.BoolAttribute{
			MarkdownDescription: "Set VBD as bootable, default to be `false`.",
			Optional:            true,
			Computed:            true,
		},
		"mode": schema.StringAttribute{
			MarkdownDescription: "The mode the VBD should be mounted with, default to be `\"RW\"`." + "<br />" +
				"Can be set as `\"RO\"` or `\"RW\"`.",
			Optional: true,
			Computed: true,
			Validators: []validator.String{
				stringvalidator.OneOf("RO", "RW"),
			},
		},
	}
}

func setVBDDefaults(vbd *vbdResourceModel) {
	// Work around for https://github.com/hashicorp/terraform-plugin-framework/issues/726
	if vbd.Mode.IsUnknown() || vbd.Mode.IsNull() {
		vbd.Mode = types.StringValue("RW")
	}

	if vbd.Bootable.IsUnknown() || vbd.Bootable.IsNull() {
		vbd.Bootable = types.BoolValue(false)
	}
}

func createVBD(session *xenapi.Session, vmRef xenapi.VMRef, vbd vbdResourceModel, vbdType xenapi.VbdType) error {
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

	vbdMode := xenapi.VbdMode(vbd.Mode.ValueString())
	if vbdType == xenapi.VbdTypeCD {
		vbdMode = xenapi.VbdModeRO
	}

	vbdRecord := xenapi.VBDRecord{
		VM:         vmRef,
		VDI:        vdiRef,
		Type:       vbdType,
		Mode:       vbdMode,
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

func createVBDs(ctx context.Context, session *xenapi.Session, vmRef xenapi.VMRef, data vmResourceModel, vbdType xenapi.VbdType) error {
	elements := make([]vbdResourceModel, 0, len(data.HardDrive.Elements()))
	if !data.HardDrive.IsUnknown() {
		diags := data.HardDrive.ElementsAs(ctx, &elements, false)
		if diags.HasError() {
			return errors.New("unable to get HardDrive elements")
		}
	}

	// Sort based on the `Bootable` field, with `true` values coming first.
	sort.Slice(elements, func(i, j int) bool {
		return elements[i].Bootable.ValueBool() && !elements[j].Bootable.ValueBool()
	})

	for _, vbd := range elements {
		tflog.Debug(ctx, "---> Create VBD with VDI: "+vbd.VDI.String()+"  Mode: "+vbd.Mode.String()+"  Bootable: "+vbd.Bootable.String())
		err := createVBD(session, vmRef, vbd, vbdType)
		if err != nil {
			return err
		}
	}

	return checkHardDriveExist(session, vmRef)
}

func updateVBDs(ctx context.Context, plan vmResourceModel, state vmResourceModel, vmRef xenapi.VMRef, session *xenapi.Session) error {
	planHardDrives := make([]vbdResourceModel, 0, len(state.HardDrive.Elements()))
	if !plan.HardDrive.IsUnknown() {
		diags := plan.HardDrive.ElementsAs(ctx, &planHardDrives, false)
		if diags.HasError() {
			return errors.New("unable to get HardDrives in plan data")
		}
	}

	stateHardDrives := make([]vbdResourceModel, 0, len(state.HardDrive.Elements()))
	if !state.HardDrive.IsUnknown() && !state.HardDrive.IsNull() {
		diags := state.HardDrive.ElementsAs(ctx, &stateHardDrives, false)
		if diags.HasError() {
			return errors.New("unable to get HardDrives in state data")
		}
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

	vmState, err := xenapi.VM.GetPowerState(session, vmRef)
	if err != nil {
		return errors.New(err.Error())
	}

	// Destroy VBDs that are not in plan
	for vdiUUID, stateVBD := range stateHardDrivesMap {
		if _, ok := planHardDrivesMap[vdiUUID]; !ok {
			if vmState == xenapi.VMPowerStateRunning {
				return errors.New("unable to delete the item in hard_drive for a running VM")
			}
			tflog.Debug(ctx, "---> Destroy VBD:	"+stateVBD.VBD.String())
			err = xenapi.VBD.Destroy(session, xenapi.VBDRef(stateVBD.VBD.ValueString()))
			if err != nil {
				if !strings.Contains(err.Error(), "HANDLE_INVALID") {
					return errors.New(err.Error())
				}
				tflog.Debug(ctx, "HANDLE_INVALID: VBD already been destroyed.")
			}
		}
	}

	// Create VBDs that are in plan but not in state, Update VBDs if already exists and attributes changed
	for vdiUUID, planVBD := range planHardDrivesMap {
		stateVBD, ok := stateHardDrivesMap[vdiUUID]
		if !ok {
			if vmState == xenapi.VMPowerStateRunning && planVBD.Mode.ValueString() == "RO" {
				return errors.New("unable to create the item with 'RO' mode in hard_drive for a running VM")
			}
			tflog.Debug(ctx, "---> Create VBD for VDI: "+vdiUUID+" <---")
			err = createVBD(session, vmRef, planVBD, xenapi.VbdTypeDisk)
			if err != nil {
				return err
			}
		} else {
			// Update VBD if attributes changed
			setVBDDefaults(&planVBD)

			if !planVBD.Mode.Equal(stateVBD.Mode) {
				if vmState == xenapi.VMPowerStateRunning {
					return errors.New("unable to update the item's mode in hard_drive for a running VM")
				}
				tflog.Debug(ctx, "---> VBD.SetMode:	"+planVBD.Mode.String())
				err = xenapi.VBD.SetMode(session, xenapi.VBDRef(stateVBD.VBD.ValueString()), xenapi.VbdMode(planVBD.Mode.ValueString()))
				if err != nil {
					return errors.New(err.Error())
				}
			}

			if !planVBD.Bootable.Equal(stateVBD.Bootable) {
				if vmState == xenapi.VMPowerStateRunning {
					return errors.New("unable to update the item's bootable in hard_drive for a running VM")
				}
				tflog.Debug(ctx, "---> VBD.SetBootable:	"+planVBD.Bootable.String())
				err = xenapi.VBD.SetBootable(session, xenapi.VBDRef(stateVBD.VBD.ValueString()), planVBD.Bootable.ValueBool())
				if err != nil {
					return errors.New(err.Error())
				}
			}
		}
	}

	return checkHardDriveExist(session, vmRef)
}

func getAllDiskTypeVBDs(session *xenapi.Session, vmRef xenapi.VMRef) ([]string, error) {
	var diskRefs []string
	vbdRefs, err := xenapi.VM.GetVBDs(session, vmRef)
	if err != nil {
		return diskRefs, errors.New(err.Error())
	}
	for _, vbdRef := range vbdRefs {
		vbdType, err := xenapi.VBD.GetType(session, vbdRef)
		if err != nil {
			return diskRefs, errors.New(err.Error())
		}
		if vbdType == xenapi.VbdTypeDisk {
			diskRefs = append(diskRefs, string(vbdRef))
		}
	}
	return diskRefs, nil
}

func checkHardDriveExist(session *xenapi.Session, vmRef xenapi.VMRef) error {
	hardDrives, err := getAllDiskTypeVBDs(session, vmRef)
	if err != nil {
		return err
	}
	if len(hardDrives) < 1 {
		return errors.New("no hard drive found on VM, please set at least one to VM")
	}
	return nil
}

func getTemplateVBDRefListFromVMRecord(vmRecord xenapi.VMRecord) []xenapi.VBDRef {
	templateVBDRefs, ok := vmRecord.OtherConfig["tf_template_vbds"]
	if !ok {
		templateVBDRefs = ""
	}
	templateVBDRefList := []xenapi.VBDRef{}
	if templateVBDRefs != "" {
		refs := strings.Split(templateVBDRefs, ",")
		for _, ref := range refs {
			templateVBDRefList = append(templateVBDRefList, xenapi.VBDRef(ref))
		}
	}
	return templateVBDRefList
}

func getVDIUUIDFromISOName(session *xenapi.Session, isoName string) (string, error) {
	var vdiUUID string
	vdiRecords, err := xenapi.VDI.GetAllRecords(session)
	if err != nil {
		return vdiUUID, errors.New(err.Error())
	}

	vdiUUIDList := make([]string, 0)
	for _, vdiRecord := range vdiRecords {
		if vdiRecord.NameLabel == isoName {
			vdiUUIDList = append(vdiUUIDList, vdiRecord.UUID)
		}
	}

	if len(vdiUUIDList) == 0 {
		return vdiUUID, errors.New("no VDI found with name: " + isoName)
	}

	if len(vdiUUIDList) != 0 && len(vdiUUIDList) > 1 {
		return vdiUUID, errors.New("multiple VDIs found with name: " + isoName)
	}

	vdiUUID = vdiUUIDList[0]

	return vdiUUID, nil
}

func setCDROM(ctx context.Context, session *xenapi.Session, vmRef xenapi.VMRef, plan vmResourceModel) error {
	if plan.CDROM.IsUnknown() {
		tflog.Debug(ctx, "---> CD-ROM is not set, use the default value")
		return nil
	}
	planCDROM := plan.CDROM.ValueString()
	vmRecord, err := xenapi.VM.GetRecord(session, vmRef)
	if err != nil {
		return errors.New(err.Error())
	}
	baseCD, err := getCDFromVMRecord(ctx, session, vmRecord)
	if err != nil {
		return err
	}

	if string(baseCD.vbdRef) == "OpaqueRef:NULL" || string(baseCD.vbdRef) == "" {
		if planCDROM != "" {
			// create the CD-ROM if not exist
			err = createCDROM(session, vmRef, planCDROM)
			if err != nil {
				return err
			}
		}
	} else {
		// get the new vdiUUID
		vdiUUID := ""
		if planCDROM != "" && planCDROM != baseCD.isoName {
			uuid, err := getVDIUUIDFromISOName(session, planCDROM)
			if err != nil {
				return err
			}
			vdiUUID = uuid
		}
		if planCDROM != baseCD.isoName {
			// change the CD-ROM
			err = changeVMISO(ctx, session, baseCD, vdiUUID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func changeVMISO(ctx context.Context, session *xenapi.Session, cd cdVBD, vdiUUID string) error {
	if !cd.empty {
		tflog.Debug(ctx, "---> Eject the exist ISO")
		err := xenapi.VBD.Eject(session, cd.vbdRef)
		if err != nil {
			return errors.New(err.Error())
		}
	}
	if vdiUUID != "" {
		tflog.Debug(ctx, "---> Insert the new ISO")
		vdiRef, err := xenapi.VDI.GetByUUID(session, vdiUUID)
		if err != nil {
			return errors.New(err.Error())
		}
		err = xenapi.VBD.Insert(session, cd.vbdRef, vdiRef)
		if err != nil {
			return errors.New(err.Error())
		}
	}
	return nil
}

func createCDROM(session *xenapi.Session, vmRef xenapi.VMRef, isoName string) error {
	vdiUUID, err := getVDIUUIDFromISOName(session, isoName)
	if err != nil {
		return err
	}
	var vbdRes vbdResourceModel
	vbdRes.VDI = types.StringValue(vdiUUID)
	err = createVBD(session, vmRef, vbdRes, xenapi.VbdTypeCD)
	if err != nil {
		return err
	}

	return nil
}

type cdVBD struct {
	vbdRef  xenapi.VBDRef
	empty   bool
	isoName string
}

func getCDFromVMRecord(ctx context.Context, session *xenapi.Session, vmRecord xenapi.VMRecord) (cdVBD, error) {
	var cd cdVBD
	_, vbdSet, err := getVBDsFromVMRecord(ctx, session, vmRecord, xenapi.VbdTypeCD)
	if err != nil {
		return cd, err
	}

	if len(vbdSet) == 0 {
		return cd, nil
	}

	// if vbdSet is not empty, but it should only have one CDROM
	if len(vbdSet) != 0 && len(vbdSet) > 1 {
		return cd, errors.New("multiple CD-ROMs found")
	}

	cd.vbdRef = xenapi.VBDRef(vbdSet[0].VBD.ValueString())
	if string(cd.vbdRef) != "OpaqueRef:NULL" {
		empty, err := xenapi.VBD.GetEmpty(session, cd.vbdRef)
		if err != nil {
			return cd, errors.New(err.Error())
		}
		cd.empty = empty
	}
	vdiUUID := vbdSet[0].VDI.ValueString()
	if vdiUUID != "" {
		vdiRef, err := xenapi.VDI.GetByUUID(session, vdiUUID)
		if err != nil {
			return cd, errors.New(err.Error())
		}
		isoName, err := xenapi.VDI.GetNameLabel(session, vdiRef)
		if err != nil {
			return cd, errors.New(err.Error())
		}
		cd.isoName = isoName
	}

	return cd, nil
}
