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
			err = createVBD(session, vmRef, planVBD, xenapi.VbdTypeDisk)
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

	return checkHardDriveExist(session, vmRef)
}

func getAllDiskTypeVBDs(session *xenapi.Session, vmRef xenapi.VMRef) ([]string, error) {
	var diskRefs []string
	vbdRefs, err := xenapi.VM.GetVBDs(session, vmRef)
	if err != nil {
		return diskRefs, errors.New(err.Error())
	}
	for _, vbdRef := range vbdRefs {
		vbdtype, err := xenapi.VBD.GetType(session, vbdRef)
		if err != nil {
			return diskRefs, errors.New(err.Error())
		}
		if vbdtype == xenapi.VbdTypeDisk {
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
		tflog.Debug(ctx, "---> CD-ROM is not set, use the default value from the VM template")
		return nil
	}
	planCDROM := plan.CDROM.ValueString()
	vmRecord, err := xenapi.VM.GetRecord(session, vmRef)
	if err != nil {
		return errors.New(err.Error())
	}
	templateCDROM, templateVBDRef, err := getISOFromVMRecord(ctx, session, vmRecord)
	if err != nil {
		return err
	}
	if templateVBDRef != "" && (planCDROM == "" || planCDROM != templateCDROM) {
		err := xenapi.VBD.Destroy(session, templateVBDRef)
		if err != nil {
			return errors.New(err.Error())
		}
	}
	if planCDROM != "" && (templateVBDRef == "" || planCDROM != templateCDROM) {
		err = createCDROM(session, vmRef, planCDROM)
		if err != nil {
			return err
		}
	}

	return nil
}

func updateCDROM(ctx context.Context, session *xenapi.Session, vmRef xenapi.VMRef, plan vmResourceModel, state vmResourceModel) error {
	if plan.CDROM.IsUnknown() {
		tflog.Debug(ctx, "---> use default CD-ROM, continue")
		return nil
	}
	stateCDROM := state.CDROM.ValueString()
	planCDROM := plan.CDROM.ValueString()

	if planCDROM == "" && stateCDROM == "" {
		tflog.Debug(ctx, "---> CD-ROM is not set, continue")
		return nil
	}

	if stateCDROM != "" && (planCDROM == "" || planCDROM != stateCDROM) {
		tflog.Debug(ctx, "---> Clean the exist CD-ROM")
		vmRecord, err := xenapi.VM.GetRecord(session, vmRef)
		if err != nil {
			return errors.New(err.Error())
		}
		_, vbdRef, err := getISOFromVMRecord(ctx, session, vmRecord)
		if err != nil {
			return err
		}
		err = xenapi.VBD.Destroy(session, vbdRef)
		if err != nil {
			return errors.New(err.Error())
		}
	}
	if planCDROM != "" && (stateCDROM == "" || planCDROM != stateCDROM) {
		tflog.Debug(ctx, "---> Create new CD-ROM: "+planCDROM)
		err := createCDROM(session, vmRef, planCDROM)
		if err != nil {
			return err
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

func getISOFromVMRecord(ctx context.Context, session *xenapi.Session, vmRecord xenapi.VMRecord) (string, xenapi.VBDRef, error) {
	var isoName = ""
	var vbdRef xenapi.VBDRef
	_, vbdSet, err := getVBDsFromVMRecord(ctx, session, vmRecord, xenapi.VbdTypeCD)
	if err != nil {
		return isoName, vbdRef, err
	}

	if len(vbdSet) == 0 {
		return isoName, vbdRef, nil
	}

	// if vbdSet is not empty, but it should only have one CDROM
	if len(vbdSet) != 0 && len(vbdSet) > 1 {
		return isoName, vbdRef, errors.New("multiple CD-ROMs found")
	}

	vbdRef = xenapi.VBDRef(vbdSet[0].VBD.ValueString())
	vdiUUID := vbdSet[0].VDI.ValueString()
	if vdiUUID != "" {
		vdiRef, err := xenapi.VDI.GetByUUID(session, vdiUUID)
		if err != nil {
			return isoName, vbdRef, errors.New(err.Error())
		}
		isoName, err = xenapi.VDI.GetNameLabel(session, vdiRef)
		if err != nil {
			return isoName, vbdRef, errors.New(err.Error())
		}
	}

	return isoName, vbdRef, nil
}
