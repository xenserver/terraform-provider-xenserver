package xenserver

import (
	"context"
	"errors"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"xenapi"
)

type vmDataSourceModel struct {
	UUID      types.String   `tfsdk:"uuid"`
	NameLabel types.String   `tfsdk:"name_label"`
	DataItems []vmRecordData `tfsdk:"data_items"`
}

type vmRecordData struct {
	UUID                        types.String  `tfsdk:"uuid"`
	AllowedOperations           types.List    `tfsdk:"allowed_operations"`
	CurrentOperations           types.Map     `tfsdk:"current_operations"`
	NameLabel                   types.String  `tfsdk:"name_label"`
	NameDescription             types.String  `tfsdk:"name_description"`
	PowerState                  types.String  `tfsdk:"power_state"`
	UserVersion                 types.Int64   `tfsdk:"user_version"`
	IsATemplate                 types.Bool    `tfsdk:"is_a_template"`
	IsDefaultTemplate           types.Bool    `tfsdk:"is_default_template"`
	SuspendVDI                  types.String  `tfsdk:"suspend_vdi"`
	ResidentOn                  types.String  `tfsdk:"resident_on"`
	ScheduledToBeResidentOn     types.String  `tfsdk:"scheduled_to_be_resident_on"`
	Affinity                    types.String  `tfsdk:"affinity"`
	MemoryOverhead              types.Int64   `tfsdk:"memory_overhead"`
	MemoryTarget                types.Int64   `tfsdk:"memory_target"`
	MemoryStaticMax             types.Int64   `tfsdk:"memory_static_max"`
	MemoryDynamicMax            types.Int64   `tfsdk:"memory_dynamic_max"`
	MemoryDynamicMin            types.Int64   `tfsdk:"memory_dynamic_min"`
	MemoryStaticMin             types.Int64   `tfsdk:"memory_static_min"`
	VCPUsParams                 types.Map     `tfsdk:"vcpus_params"`
	VCPUsMax                    types.Int64   `tfsdk:"vcpus_max"`
	VCPUsAtStartup              types.Int64   `tfsdk:"vcpus_at_startup"`
	ActionsAfterSoftreboot      types.String  `tfsdk:"actions_after_softreboot"`
	ActionsAfterShutdown        types.String  `tfsdk:"actions_after_shutdown"`
	ActionsAfterReboot          types.String  `tfsdk:"actions_after_reboot"`
	ActionsAfterCrash           types.String  `tfsdk:"actions_after_crash"`
	Consoles                    types.List    `tfsdk:"consoles"`
	VIFs                        types.List    `tfsdk:"vifs"`
	VBDs                        types.List    `tfsdk:"vbds"`
	VUSBs                       types.List    `tfsdk:"vusbs"`
	CrashDumps                  types.List    `tfsdk:"crash_dumps"`
	VTPMs                       types.List    `tfsdk:"vtpms"`
	PVBootloader                types.String  `tfsdk:"pv_bootloader"`
	PVKernel                    types.String  `tfsdk:"pv_kernel"`
	PVRamdisk                   types.String  `tfsdk:"pv_ramdisk"`
	PVArgs                      types.String  `tfsdk:"pv_args"`
	PVBootloaderArgs            types.String  `tfsdk:"pv_bootloader_args"`
	PVLegacyArgs                types.String  `tfsdk:"pv_legacy_args"`
	HVMBootPolicy               types.String  `tfsdk:"hvm_boot_policy"`
	HVMBootParams               types.Map     `tfsdk:"hvm_boot_params"`
	HVMShadowMultiplier         types.Float64 `tfsdk:"hvm_shadow_multiplier"`
	Platform                    types.Map     `tfsdk:"platform"`
	PCIBus                      types.String  `tfsdk:"pci_bus"`
	OtherConfig                 types.Map     `tfsdk:"other_config"`
	Domid                       types.Int64   `tfsdk:"domid"`
	Domarch                     types.String  `tfsdk:"domarch"`
	LastBootCPUFlags            types.Map     `tfsdk:"last_boot_cpu_flags"`
	IsControlDomain             types.Bool    `tfsdk:"is_control_domain"`
	Metrics                     types.String  `tfsdk:"metrics"`
	GuestMetrics                types.String  `tfsdk:"guest_metrics"`
	LastBootedRecord            types.String  `tfsdk:"last_booted_record"`
	Recommendations             types.String  `tfsdk:"recommendations"`
	XenstoreData                types.Map     `tfsdk:"xenstore_data"`
	HaAlwaysRun                 types.Bool    `tfsdk:"ha_always_run"`
	HaRestartPriority           types.String  `tfsdk:"ha_restart_priority"`
	IsASnapshot                 types.Bool    `tfsdk:"is_a_snapshot"`
	SnapshotOf                  types.String  `tfsdk:"snapshot_of"`
	Snapshots                   types.List    `tfsdk:"snapshots"`
	SnapshotTime                types.String  `tfsdk:"snapshot_time"`
	TransportableSnapshotID     types.String  `tfsdk:"transportable_snapshot_id"`
	Blobs                       types.Map     `tfsdk:"blobs"`
	Tags                        types.List    `tfsdk:"tags"`
	BlockedOperations           types.Map     `tfsdk:"blocked_operations"`
	SnapshotInfo                types.Map     `tfsdk:"snapshot_info"`
	SnapshotMetadata            types.String  `tfsdk:"snapshot_metadata"`
	Parent                      types.String  `tfsdk:"parent"`
	Children                    types.List    `tfsdk:"children"`
	BiosStrings                 types.Map     `tfsdk:"bios_strings"`
	ProtectionPolicy            types.String  `tfsdk:"protection_policy"`
	IsSnapshotFromVmpp          types.Bool    `tfsdk:"is_snapshot_from_vmpp"`
	SnapshotSchedule            types.String  `tfsdk:"snapshot_schedule"`
	IsVmssSnapshot              types.Bool    `tfsdk:"is_vmss_snapshot"`
	Appliance                   types.String  `tfsdk:"appliance"`
	StartDelay                  types.Int64   `tfsdk:"start_delay"`
	ShutdownDelay               types.Int64   `tfsdk:"shutdown_delay"`
	Order                       types.Int64   `tfsdk:"order"`
	VGPUs                       types.List    `tfsdk:"vgpus"`
	AttachedPCIs                types.List    `tfsdk:"attached_pcis"`
	SuspendSR                   types.String  `tfsdk:"suspend_sr"`
	Version                     types.Int64   `tfsdk:"version"`
	GenerationID                types.String  `tfsdk:"generation_id"`
	HardwarePlatformVersion     types.Int64   `tfsdk:"hardware_platform_version"`
	HasVendorDevice             types.Bool    `tfsdk:"has_vendor_device"`
	RequiresReboot              types.Bool    `tfsdk:"requires_reboot"`
	ReferenceLabel              types.String  `tfsdk:"reference_label"`
	DomainType                  types.String  `tfsdk:"domain_type"`
	NVRAM                       types.Map     `tfsdk:"nvram"`
	PendingGuidances            types.List    `tfsdk:"pending_guidances"`
	PendingGuidancesRecommended types.List    `tfsdk:"pending_guidances_recommended"`
	PendingGuidancesFull        types.List    `tfsdk:"pending_guidances_full"`
}

// vmResourceModel describes the resource data model.
type vmResourceModel struct {
	NameLabel        types.String `tfsdk:"name_label"`
	TemplateName     types.String `tfsdk:"template_name"`
	OtherConfig      types.Map    `tfsdk:"other_config"`
	HardDrive        types.Set    `tfsdk:"hard_drive"`
	NetworkInterface types.Set    `tfsdk:"network_interface"`
	UUID             types.String `tfsdk:"uuid"`
	ID               types.String `tfsdk:"id"`
}

func VMSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name_label": schema.StringAttribute{
			MarkdownDescription: "The name of the virtual machine",
			Required:            true,
		},
		"template_name": schema.StringAttribute{
			MarkdownDescription: "The template name of the virtual machine which cloned from",
			Required:            true,
		},
		"hard_drive": schema.SetNestedAttribute{
			MarkdownDescription: "A set of hard drive attributes to attach to the virtual machine",
			NestedObject: schema.NestedAttributeObject{
				Attributes: VBDSchema(),
			},
			Optional: true,
			Validators: []validator.Set{
				setvalidator.SizeAtLeast(1),
			},
		},
		"network_interface": schema.SetNestedAttribute{
			MarkdownDescription: "A set of network interface attributes to attach to the virtual machine",
			NestedObject: schema.NestedAttributeObject{
				Attributes: VIFSchema(),
			},
			Required: true,
			Validators: []validator.Set{
				setvalidator.SizeAtLeast(1),
			},
		},
		"other_config": schema.MapAttribute{
			MarkdownDescription: "The other config of the virtual machine",
			Optional:            true,
			Computed:            true,
			ElementType:         types.StringType,
			Default:             mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
		},
		"uuid": schema.StringAttribute{
			MarkdownDescription: "The UUID of the virtual machine",
			Computed:            true,
			// attributes which are not configurable and that should not show updates from the existing state value
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"id": schema.StringAttribute{
			MarkdownDescription: "The test id of the virtual machine",
			Computed:            true,
			// attributes which are not configurable and that should not show updates from the existing state value
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
	}
}

func updateVMRecordData(ctx context.Context, record xenapi.VMRecord, data *vmRecordData) error {
	data.UUID = types.StringValue(record.UUID)
	var diags diag.Diagnostics
	data.AllowedOperations, diags = types.ListValueFrom(ctx, types.StringType, record.AllowedOperations)
	if diags.HasError() {
		return errors.New("unable to read VM allowed operations")
	}
	data.CurrentOperations, diags = types.MapValueFrom(ctx, types.StringType, record.CurrentOperations)
	if diags.HasError() {
		return errors.New("unable to read VM current operations")
	}
	data.NameLabel = types.StringValue(record.NameLabel)
	data.NameDescription = types.StringValue(record.NameDescription)
	data.PowerState = types.StringValue(string(record.PowerState))
	data.UserVersion = types.Int64Value(int64(record.UserVersion))
	data.IsATemplate = types.BoolValue(record.IsATemplate)
	data.IsDefaultTemplate = types.BoolValue(record.IsDefaultTemplate)
	data.SuspendVDI = types.StringValue(string(record.SuspendVDI))
	data.ResidentOn = types.StringValue(string(record.ResidentOn))
	data.ScheduledToBeResidentOn = types.StringValue(string(record.ScheduledToBeResidentOn))
	data.Affinity = types.StringValue(string(record.Affinity))
	data.MemoryOverhead = types.Int64Value(int64(record.MemoryOverhead))
	data.MemoryTarget = types.Int64Value(int64(record.MemoryTarget))
	data.MemoryStaticMax = types.Int64Value(int64(record.MemoryStaticMax))
	data.MemoryDynamicMax = types.Int64Value(int64(record.MemoryDynamicMax))
	data.MemoryDynamicMin = types.Int64Value(int64(record.MemoryDynamicMin))
	data.MemoryStaticMin = types.Int64Value(int64(record.MemoryStaticMin))
	data.VCPUsParams, diags = types.MapValueFrom(ctx, types.StringType, record.VCPUsParams)
	if diags.HasError() {
		return errors.New("unable to read VM VCPUs params")
	}
	data.VCPUsMax = types.Int64Value(int64(record.VCPUsMax))
	data.VCPUsAtStartup = types.Int64Value(int64(record.VCPUsAtStartup))
	data.ActionsAfterSoftreboot = types.StringValue(string(record.ActionsAfterSoftreboot))
	data.ActionsAfterShutdown = types.StringValue(string(record.ActionsAfterShutdown))
	data.ActionsAfterReboot = types.StringValue(string(record.ActionsAfterReboot))
	data.ActionsAfterCrash = types.StringValue(string(record.ActionsAfterCrash))
	data.Consoles, diags = types.ListValueFrom(ctx, types.StringType, record.Consoles)
	if diags.HasError() {
		return errors.New("unable to read VM consoles")
	}
	data.VIFs, diags = types.ListValueFrom(ctx, types.StringType, record.VIFs)
	if diags.HasError() {
		return errors.New("unable to read VM VIFs")
	}
	data.VBDs, diags = types.ListValueFrom(ctx, types.StringType, record.VBDs)
	if diags.HasError() {
		return errors.New("unable to read VM VBDs")
	}
	data.VUSBs, diags = types.ListValueFrom(ctx, types.StringType, record.VUSBs)
	if diags.HasError() {
		return errors.New("unable to read VM VUSBs")
	}
	data.CrashDumps, diags = types.ListValueFrom(ctx, types.StringType, record.CrashDumps)
	if diags.HasError() {
		return errors.New("unable to read VM crash dumps")
	}
	data.VTPMs, diags = types.ListValueFrom(ctx, types.StringType, record.VTPMs)
	if diags.HasError() {
		return errors.New("unable to read VM VTPMs")
	}
	data.PVBootloader = types.StringValue(record.PVBootloader)
	data.PVKernel = types.StringValue(record.PVKernel)
	data.PVRamdisk = types.StringValue(record.PVRamdisk)
	data.PVArgs = types.StringValue(record.PVArgs)
	data.PVBootloaderArgs = types.StringValue(record.PVBootloaderArgs)
	data.PVLegacyArgs = types.StringValue(record.PVLegacyArgs)
	data.HVMBootPolicy = types.StringValue(record.HVMBootPolicy)
	data.HVMBootParams, diags = types.MapValueFrom(ctx, types.StringType, record.HVMBootParams)
	if diags.HasError() {
		return errors.New("unable to read VM HVM boot params")
	}
	data.HVMShadowMultiplier = types.Float64Value(float64(record.HVMShadowMultiplier))
	data.Platform, diags = types.MapValueFrom(ctx, types.StringType, record.Platform)
	if diags.HasError() {
		return errors.New("unable to read VM platform")
	}
	data.PCIBus = types.StringValue(record.PCIBus)
	data.OtherConfig, diags = types.MapValueFrom(ctx, types.StringType, record.OtherConfig)
	if diags.HasError() {
		return errors.New("unable to read VM other config")
	}
	data.Domid = types.Int64Value(int64(record.Domid))
	data.Domarch = types.StringValue(record.Domarch)
	data.LastBootCPUFlags, diags = types.MapValueFrom(ctx, types.StringType, record.LastBootCPUFlags)
	if diags.HasError() {
		return errors.New("unable to read VM last boot CPU flags")
	}
	data.IsControlDomain = types.BoolValue(record.IsControlDomain)
	data.Metrics = types.StringValue(string(record.Metrics))
	data.GuestMetrics = types.StringValue(string(record.GuestMetrics))
	data.LastBootedRecord = types.StringValue(record.LastBootedRecord)
	data.Recommendations = types.StringValue(record.Recommendations)
	data.XenstoreData, diags = types.MapValueFrom(ctx, types.StringType, record.XenstoreData)
	if diags.HasError() {
		return errors.New("unable to read VM xenstore data")
	}
	data.HaAlwaysRun = types.BoolValue(record.HaAlwaysRun)
	data.HaRestartPriority = types.StringValue(record.HaRestartPriority)
	data.IsASnapshot = types.BoolValue(record.IsASnapshot)
	data.SnapshotOf = types.StringValue(string(record.SnapshotOf))
	data.Snapshots, diags = types.ListValueFrom(ctx, types.StringType, record.Snapshots)
	if diags.HasError() {
		return errors.New("unable to read VM snapshots")
	}
	// Transfer time.Time to string
	data.SnapshotTime = types.StringValue(record.SnapshotTime.String())
	data.TransportableSnapshotID = types.StringValue(record.TransportableSnapshotID)
	data.Blobs, diags = types.MapValueFrom(ctx, types.StringType, record.Blobs)
	if diags.HasError() {
		return errors.New("unable to read VM blobs")
	}
	data.Tags, diags = types.ListValueFrom(ctx, types.StringType, record.Tags)
	if diags.HasError() {
		return errors.New("unable to read VM tags")
	}
	data.BlockedOperations, diags = types.MapValueFrom(ctx, types.StringType, record.BlockedOperations)
	if diags.HasError() {
		return errors.New("unable to read VM blocked operations")
	}
	data.SnapshotInfo, diags = types.MapValueFrom(ctx, types.StringType, record.SnapshotInfo)
	if diags.HasError() {
		return errors.New("unable to read VM snapshot info")
	}
	data.SnapshotMetadata = types.StringValue(record.SnapshotMetadata)
	data.Parent = types.StringValue(string(record.Parent))
	data.Children, diags = types.ListValueFrom(ctx, types.StringType, record.Children)
	if diags.HasError() {
		return errors.New("unable to read VM children")
	}
	data.BiosStrings, diags = types.MapValueFrom(ctx, types.StringType, record.BiosStrings)
	if diags.HasError() {
		return errors.New("unable to read VM bios strings")
	}
	data.ProtectionPolicy = types.StringValue(string(record.ProtectionPolicy))
	data.IsSnapshotFromVmpp = types.BoolValue(record.IsSnapshotFromVmpp)
	data.SnapshotSchedule = types.StringValue(string(record.SnapshotSchedule))
	data.IsVmssSnapshot = types.BoolValue(record.IsVmssSnapshot)
	data.Appliance = types.StringValue(string(record.Appliance))
	data.StartDelay = types.Int64Value(int64(record.StartDelay))
	data.ShutdownDelay = types.Int64Value(int64(record.ShutdownDelay))
	data.Order = types.Int64Value(int64(record.Order))
	data.VGPUs, diags = types.ListValueFrom(ctx, types.StringType, record.VGPUs)
	if diags.HasError() {
		return errors.New("unable to read VM VGPUs")
	}
	data.AttachedPCIs, diags = types.ListValueFrom(ctx, types.StringType, record.AttachedPCIs)
	if diags.HasError() {
		return errors.New("unable to read VM attached PCIs")
	}
	data.SuspendSR = types.StringValue(string(record.SuspendSR))
	data.Version = types.Int64Value(int64(record.Version))
	data.GenerationID = types.StringValue(record.GenerationID)
	data.HardwarePlatformVersion = types.Int64Value(int64(record.HardwarePlatformVersion))
	data.HasVendorDevice = types.BoolValue(record.HasVendorDevice)
	data.RequiresReboot = types.BoolValue(record.RequiresReboot)
	data.ReferenceLabel = types.StringValue(record.ReferenceLabel)
	data.DomainType = types.StringValue(string(record.DomainType))
	data.NVRAM, diags = types.MapValueFrom(ctx, types.StringType, record.NVRAM)
	if diags.HasError() {
		return errors.New("unable to read VM NVRAM")
	}
	data.PendingGuidances, diags = types.ListValueFrom(ctx, types.StringType, record.PendingGuidances)
	if diags.HasError() {
		return errors.New("unable to read VM pending guidances")
	}
	data.PendingGuidancesRecommended, diags = types.ListValueFrom(ctx, types.StringType, record.PendingGuidancesRecommended)
	if diags.HasError() {
		return errors.New("unable to read VM pending guidances recommended")
	}
	data.PendingGuidancesFull, diags = types.ListValueFrom(ctx, types.StringType, record.PendingGuidancesFull)
	if diags.HasError() {
		return errors.New("unable to read VM pending guidances full")
	}
	return nil
}

func getFirstTemplate(session *xenapi.Session, templateName string) (xenapi.VMRef, error) {
	var vmRef xenapi.VMRef
	records, err := xenapi.VM.GetAllRecords(session)
	if err != nil {
		return vmRef, errors.New(err.Error())
	}

	// Get the first VM template ref
	for vmRef, record := range records {
		if record.IsATemplate && strings.Contains(record.NameLabel, templateName) {
			return vmRef, nil
		}
	}
	return vmRef, errors.New("unable to find VM template ref")
}

func setOtherConfigFromPlan(ctx context.Context, session *xenapi.Session, plan vmResourceModel, vmRef xenapi.VMRef) error {
	planOtherConfig := make(map[string]string)
	if !plan.OtherConfig.IsUnknown() {
		diags := plan.OtherConfig.ElementsAs(ctx, &planOtherConfig, false)
		if diags.HasError() {
			return errors.New("unable to read VM other config")
		}
	}

	vmOtherConfig, err := xenapi.VM.GetOtherConfig(session, vmRef)
	if err != nil {
		return errors.New(err.Error())
	}

	var tfOtherConfigKeys string
	for key, value := range planOtherConfig {
		tfOtherConfigKeys += key + ","
		// if the key already exists in originalOtherConfig, update it, otherwise add it
		vmOtherConfig[key] = value
		tflog.Debug(ctx, "-----> setOtherConfig key: "+key+" value: "+value)
	}

	originalTFOtherConfigKeys := vmOtherConfig["tf_other_config_keys"]
	// To compare originalTFOtherConfigKeys with tfOtherConfigKeys, if the key is not in tfOtherConfigKeys, delete it
	for key := range vmOtherConfig {
		if !strings.Contains(tfOtherConfigKeys, key+",") && strings.Contains(originalTFOtherConfigKeys, key+",") {
			delete(vmOtherConfig, key)
		}
	}

	vmOtherConfig["tf_other_config_keys"] = tfOtherConfigKeys
	vmOtherConfig["tf_template_name"] = plan.TemplateName.ValueString()

	err = xenapi.VM.SetOtherConfig(session, vmRef, vmOtherConfig)
	if err != nil {
		return errors.New(err.Error())
	}

	return nil
}

func updateVMResourceModelComputed(ctx context.Context, session *xenapi.Session, vmRecord xenapi.VMRecord, data *vmResourceModel) error {
	var err error
	data.UUID = types.StringValue(vmRecord.UUID)
	data.ID = types.StringValue(vmRecord.UUID)
	data.HardDrive, err = getVBDsFromVMRecord(ctx, session, vmRecord)
	if err != nil {
		return err
	}

	data.NetworkInterface, err = getVIFsFromVMRecord(ctx, session, vmRecord)
	if err != nil {
		return err
	}

	// only keep the key which configured by user
	data.OtherConfig, err = getOtherConfigFromVMRecord(ctx, vmRecord)
	if err != nil {
		return err
	}

	return nil
}

// Update vmResourceModel base on new vmRecord, except uuid
func updateVMResourceModel(ctx context.Context, session *xenapi.Session, vmRecord xenapi.VMRecord, data *vmResourceModel) error {
	data.NameLabel = types.StringValue(vmRecord.NameLabel)
	data.TemplateName = types.StringValue(vmRecord.OtherConfig["tf_template_name"])
	return updateVMResourceModelComputed(ctx, session, vmRecord, data)
}

func getVBDsFromVMRecord(ctx context.Context, session *xenapi.Session, vmRecord xenapi.VMRecord) (basetypes.SetValue, error) {
	var vbdSet []vbdResourceModel
	var setValue basetypes.SetValue
	for _, vbdRef := range vmRecord.VBDs {
		vbdRecord, err := xenapi.VBD.GetRecord(session, vbdRef)
		if err != nil {
			return setValue, errors.New("unable to get VBD record")
		}

		vdiRecord, err := xenapi.VDI.GetRecord(session, vbdRecord.VDI)
		if err != nil {
			return setValue, errors.New("unable to get VDI record")
		}

		vbd := vbdResourceModel{
			VDI:      types.StringValue(vdiRecord.UUID),
			VBD:      types.StringValue(string(vbdRef)),
			Bootable: types.BoolValue(vbdRecord.Bootable),
			Mode:     types.StringValue(string(vbdRecord.Mode)),
		}

		vbdSet = append(vbdSet, vbd)
	}

	setValue, diags := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: vbdResourceModelAttrTypes}, vbdSet)
	if diags.HasError() {
		return setValue, errors.New("unable to get VBD set value")
	}

	tflog.Debug(ctx, "-----> setVaule VDB "+setValue.String())
	return setValue, nil
}

func getOtherConfigFromVMRecord(ctx context.Context, vmRecord xenapi.VMRecord) (basetypes.MapValue, error) {
	tfOtherConfigKeys := vmRecord.OtherConfig["tf_other_config_keys"]
	tflog.Debug(ctx, "-----> tfOtherConfigKeys: "+tfOtherConfigKeys)
	otherConfig := make(map[string]string)
	for key := range vmRecord.OtherConfig {
		if strings.Contains(tfOtherConfigKeys, key+",") {
			otherConfig[key] = vmRecord.OtherConfig[key]
		}
	}

	otherConfigMap, diags := types.MapValueFrom(ctx, types.StringType, otherConfig)
	if diags.HasError() {
		return otherConfigMap, errors.New("unable to get other config map value")
	}

	return otherConfigMap, nil
}

func getVIFsFromVMRecord(ctx context.Context, session *xenapi.Session, vmRecord xenapi.VMRecord) (basetypes.SetValue, error) {
	var vifSet []vifResourceModel
	var setValue basetypes.SetValue
	var diags diag.Diagnostics
	for _, vifRef := range vmRecord.VIFs {
		vifRecord, err := xenapi.VIF.GetRecord(session, vifRef)
		if err != nil {
			return setValue, errors.New(err.Error())
		}

		// get network uuid
		networkRecord, err := xenapi.Network.GetRecord(session, vifRecord.Network)
		if err != nil {
			return setValue, errors.New(err.Error())
		}

		vif := vifResourceModel{
			Network: types.StringValue(networkRecord.UUID),
			VIF:     types.StringValue(string(vifRef)),
			MTU:     types.Int32Value(int32(vifRecord.MTU)),
			MAC:     types.StringValue(vifRecord.MAC),
			Device:  types.StringValue(vifRecord.Device),
		}

		vif.OtherConfig, diags = types.MapValueFrom(ctx, types.StringType, vifRecord.OtherConfig)
		if diags.HasError() {
			return setValue, errors.New("unable to read VIF other config")
		}

		vifSet = append(vifSet, vif)
	}

	setValue, diags = types.SetValueFrom(ctx, types.ObjectType{AttrTypes: vifResourceModelAttrTypes}, vifSet)
	if diags.HasError() {
		return setValue, errors.New("unable to get VIF set value")
	}

	tflog.Debug(ctx, "-----> setVaule VIF "+setValue.String())
	return setValue, nil
}

func vmResourceModelUpdate(ctx context.Context, session *xenapi.Session, vmRef xenapi.VMRef, plan vmResourceModel, state vmResourceModel) error {
	err := xenapi.VM.SetNameLabel(session, vmRef, plan.NameLabel.ValueString())
	if err != nil {
		return errors.New(err.Error())
	}

	err = setOtherConfigFromPlan(ctx, session, plan, vmRef)
	if err != nil {
		return err
	}

	err = updateVBDs(ctx, plan, state, vmRef, session)
	if err != nil {
		return err
	}

	err = updateVIFs(ctx, plan, state, vmRef, session)
	if err != nil {
		return err
	}

	return nil
}
