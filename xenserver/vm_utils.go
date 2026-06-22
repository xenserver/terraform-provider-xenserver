package xenserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
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
	UserVersion                 types.Int32   `tfsdk:"user_version"`
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
	VCPUsMax                    types.Int32   `tfsdk:"vcpus_max"`
	VCPUsAtStartup              types.Int32   `tfsdk:"vcpus_at_startup"`
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
	Domid                       types.Int32   `tfsdk:"domid"`
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
	Order                       types.Int32   `tfsdk:"order"`
	VGPUs                       types.List    `tfsdk:"vgpus"`
	AttachedPCIs                types.List    `tfsdk:"attached_pcis"`
	SuspendSR                   types.String  `tfsdk:"suspend_sr"`
	Version                     types.Int32   `tfsdk:"version"`
	GenerationID                types.String  `tfsdk:"generation_id"`
	HardwarePlatformVersion     types.Int32   `tfsdk:"hardware_platform_version"`
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
	NameDescription  types.String `tfsdk:"name_description"`
	TemplateName     types.String `tfsdk:"template_name"`
	StaticMemMin     types.Int64  `tfsdk:"static_mem_min"`
	StaticMemMax     types.Int64  `tfsdk:"static_mem_max"`
	DynamicMemMin    types.Int64  `tfsdk:"dynamic_mem_min"`
	DynamicMemMax    types.Int64  `tfsdk:"dynamic_mem_max"`
	VCPUs            types.Int32  `tfsdk:"vcpus"`
	BootMode         types.String `tfsdk:"boot_mode"`
	BootOrder        types.String `tfsdk:"boot_order"`
	CorePerSocket    types.Int32  `tfsdk:"cores_per_socket"`
	OtherConfig      types.Map    `tfsdk:"other_config"`
	HardDrive        types.Set    `tfsdk:"hard_drive"`
	NetworkInterface types.Set    `tfsdk:"network_interface"`
	CDROM            types.String `tfsdk:"cdrom"`
	UUID             types.String `tfsdk:"uuid"`
	ID               types.String `tfsdk:"id"`
	DefaultIP        types.String `tfsdk:"default_ip"`
	CheckIPTimeout   types.Int64  `tfsdk:"check_ip_timeout"`
}

func VMSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name_label": schema.StringAttribute{
			MarkdownDescription: "The name of the virtual machine.",
			Required:            true,
		},
		"name_description": schema.StringAttribute{
			MarkdownDescription: "The description of the virtual machine, default to be `\"\"`.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"template_name": schema.StringAttribute{
			MarkdownDescription: "The template name of the virtual machine which cloned from." +
				"\n\n-> **Note:** `template_name` is not allowed to be updated.",
			Required: true,
		},
		"static_mem_min": schema.Int64Attribute{
			MarkdownDescription: "Statically-set (absolute) minimum memory (bytes), default same with `static_mem_max`. The least amount of memory this VM can boot with without crashing.",
			Optional:            true,
			Computed:            true,
		},
		"static_mem_max": schema.Int64Attribute{
			MarkdownDescription: "Statically-set (absolute) maximum memory (bytes). This value acts as a hard limit of the amount of memory a guest can use at VM start time. New values only take effect on reboot.",
			Required:            true,
		},
		"dynamic_mem_min": schema.Int64Attribute{
			MarkdownDescription: "Dynamic minimum memory (bytes), default same with `static_mem_max`.",
			Optional:            true,
			Computed:            true,
		},
		"dynamic_mem_max": schema.Int64Attribute{
			MarkdownDescription: "Dynamic maximum memory (bytes), default same with `static_mem_max`.",
			Optional:            true,
			Computed:            true,
		},
		"vcpus": schema.Int32Attribute{
			MarkdownDescription: "The number of VCPUs for the virtual machine.",
			Required:            true,
		},
		"cores_per_socket": schema.Int32Attribute{
			MarkdownDescription: "The number of core pre socket for the virtual machine, default inherited from the template.",
			Optional:            true,
			Computed:            true,
		},
		"boot_mode": schema.StringAttribute{
			MarkdownDescription: "The boot mode of the virtual machine, default inherited from the template." + "<br />" +
				"This value can be one of [`\"bios\", \"uefi\", \"uefi_security\"`]." +
				"\n\n-> **Note:** `boot_mode` is not allowed to be updated.",
			Optional: true,
			Computed: true,
			Validators: []validator.String{
				stringvalidator.OneOf("bios", "uefi", "uefi_security"),
			},
		},
		"boot_order": schema.StringAttribute{
			MarkdownDescription: "The boot order of the virtual machine, default inherited from the template." + "<br />" +
				"This value is a combination string of [`\"c\", \"d\", \"n\"`]. Find more details in [Setting boot order for domUs](https://wiki.xenproject.org/wiki/Setting_boot_order_for_domUs).",
			Optional: true,
			Computed: true,
			Validators: []validator.String{
				stringvalidator.RegexMatches(regexp.MustCompile(`^[cdn]{1,3}$`), "the value is combination string of ['c', 'd', 'n']"),
			},
		},
		"cdrom": schema.StringAttribute{
			MarkdownDescription: "The VDI name in ISO library to attach to the virtual machine, default inherited from the template.",
			Optional:            true,
			Computed:            true,
		},
		"hard_drive": schema.SetNestedAttribute{
			MarkdownDescription: "A set of hard drive attributes to attach to the virtual machine, default inherited from the template." + "<br />" +
				"Set at least one item in this attribute when use it.",
			NestedObject: schema.NestedAttributeObject{
				Attributes: VBDSchema(),
			},
			Optional: true,
			Computed: true,
			Validators: []validator.Set{
				setvalidator.SizeAtLeast(1),
			},
		},
		"network_interface": schema.SetNestedAttribute{
			MarkdownDescription: "A set of network interface attributes to attach to the virtual machine." + "<br />" +
				"Set at least one item in this attribute when use it.",
			NestedObject: schema.NestedAttributeObject{
				Attributes: VIFSchema(),
			},
			Required: true,
			Validators: []validator.Set{
				setvalidator.SizeAtLeast(1),
			},
		},
		"other_config": schema.MapAttribute{
			MarkdownDescription: "The additional configuration of the virtual machine, default to be `{}`.",
			Optional:            true,
			Computed:            true,
			ElementType:         types.StringType,
			Default:             mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
		},
		"check_ip_timeout": schema.Int64Attribute{
			MarkdownDescription: "The duration for checking the IP address of the virtual machine. default is 0 seconds, once the value greater than 0, the provider will check the IP address of the virtual machine in the specified duration.",
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(0),
			Validators: []validator.Int64{
				int64validator.AtLeast(0),
			},
		},
		"default_ip": schema.StringAttribute{
			MarkdownDescription: "The default IP address of the virtual machine.",
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"uuid": schema.StringAttribute{
			MarkdownDescription: "The UUID of the virtual machine.",
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"id": schema.StringAttribute{
			MarkdownDescription: "The test ID of the virtual machine.",
			Computed:            true,
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
	data.UserVersion = types.Int32Value(int32(record.UserVersion))
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
	data.VCPUsMax = types.Int32Value(int32(record.VCPUsMax))
	data.VCPUsAtStartup = types.Int32Value(int32(record.VCPUsAtStartup))
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
	data.Domid = types.Int32Value(int32(record.Domid))
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
	data.Order = types.Int32Value(int32(record.Order))
	data.VGPUs, diags = types.ListValueFrom(ctx, types.StringType, record.VGPUs)
	if diags.HasError() {
		return errors.New("unable to read VM VGPUs")
	}
	data.AttachedPCIs, diags = types.ListValueFrom(ctx, types.StringType, record.AttachedPCIs)
	if diags.HasError() {
		return errors.New("unable to read VM attached PCIs")
	}
	data.SuspendSR = types.StringValue(string(record.SuspendSR))
	data.Version = types.Int32Value(int32(record.Version))
	data.GenerationID = types.StringValue(record.GenerationID)
	data.HardwarePlatformVersion = types.Int32Value(int32(record.HardwarePlatformVersion))
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
		if record.IsATemplate && record.NameLabel == templateName {
			return vmRef, nil
		}
	}
	return vmRef, errors.New("unable to find the VM template with the name: " + templateName)
}

func setOtherConfigWhenCreate(session *xenapi.Session, vmRef xenapi.VMRef) error {
	vmOtherConfig, err := xenapi.VM.GetOtherConfig(session, vmRef)
	if err != nil {
		return errors.New(err.Error())
	}

	// Remove "disks" from other-config for VM.Provision
	_, ok := vmOtherConfig["disks"]
	if ok {
		delete(vmOtherConfig, "disks")
	}

	// Get VM template Disk Type VBDs (which are not managed by the TF)
	templateHardDrives, err := getAllDiskTypeVBDs(session, vmRef)
	if err != nil {
		return err
	}
	templateVBDs := strings.Join(templateHardDrives, ",")
	// Set the template VBD refs only once after the VM is cloned from a template
	if templateVBDs != "" {
		vmOtherConfig["tf_template_vbds"] = templateVBDs
	}

	err = xenapi.VM.SetOtherConfig(session, vmRef, vmOtherConfig)
	if err != nil {
		return errors.New(err.Error())
	}

	return nil
}

func updateOtherConfigFromPlan(ctx context.Context, session *xenapi.Session, vmRef xenapi.VMRef, plan vmResourceModel) error {
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

	originalTFOtherConfigKeys := vmOtherConfig["tf_other_config_keys"]
	// Remove all originalTFOtherConfigKeys
	originalKeys := strings.Split(originalTFOtherConfigKeys, ",")
	for _, key := range originalKeys {
		delete(vmOtherConfig, key)
	}

	var tfOtherConfigKeys []string
	for key, value := range planOtherConfig {
		vmOtherConfig[key] = value
		tfOtherConfigKeys = append(tfOtherConfigKeys, key)
		tflog.Debug(ctx, "-----> setOtherConfig key: "+key+" value: "+value)
	}

	vmOtherConfig["tf_other_config_keys"] = strings.Join(tfOtherConfigKeys, ",")
	vmOtherConfig["tf_check_ip_timeout"] = plan.CheckIPTimeout.String()
	vmOtherConfig["tf_template_name"] = plan.TemplateName.ValueString()

	err = xenapi.VM.SetOtherConfig(session, vmRef, vmOtherConfig)
	if err != nil {
		return errors.New(err.Error())
	}

	return nil
}

func getBootModeFromVMRecord(vmRecord xenapi.VMRecord) (string, error) {
	bootMode, ok := vmRecord.HVMBootParams["firmware"]
	if !ok {
		return "", errors.New("unable to read VM HVM boot firmware")
	}
	secureBoot, ok := vmRecord.Platform["secureboot"]
	if !ok {
		return "", errors.New("unable to read VM platform secureboot")
	}

	// keep tf state consistent with the boot mode, especially user didn't provide the boot mode attribute
	if bootMode == "uefi" && secureBoot != "false" {
		bootMode = "uefi_security"
	}

	return bootMode, nil
}

func getCorePerSocket(vmRecord xenapi.VMRecord) (int32, error) {
	socket, ok := vmRecord.Platform["cores-per-socket"]
	if !ok {
		return 0, errors.New("unable to read VM platform cores-per-socket")
	}
	socketInt, err := strconv.Atoi(socket)
	if err != nil {
		return 0, errors.New("unable to convert cores-per-socket to an int value")
	}

	return int32(socketInt), nil // #nosec G109
}

func updateVMResourceModelComputed(ctx context.Context, session *xenapi.Session, vmRecord xenapi.VMRecord, data *vmResourceModel) error {
	var err error
	data.NameDescription = types.StringValue(vmRecord.NameDescription)
	data.UUID = types.StringValue(vmRecord.UUID)
	data.ID = types.StringValue(vmRecord.UUID)
	data.StaticMemMin = types.Int64Value(int64(vmRecord.MemoryStaticMin))
	data.DynamicMemMin = types.Int64Value(int64(vmRecord.MemoryDynamicMin))
	data.DynamicMemMax = types.Int64Value(int64(vmRecord.MemoryDynamicMax))

	socketInt, err := getCorePerSocket(vmRecord)
	if err != nil {
		return err
	}
	data.CorePerSocket = types.Int32Value(socketInt)

	data.NetworkInterface, err = getVIFsFromVMRecord(ctx, session, vmRecord)
	if err != nil {
		return err
	}

	data.HardDrive, _, err = getVBDsFromVMRecord(ctx, session, vmRecord, xenapi.VbdTypeDisk)
	if err != nil {
		return err
	}

	cd, err := getCDFromVMRecord(ctx, session, vmRecord)
	if err != nil {
		return err
	}
	data.CDROM = types.StringValue(cd.isoName)

	bootMode, err := getBootModeFromVMRecord(vmRecord)
	if err != nil {
		return err
	}
	data.BootMode = types.StringValue(bootMode)

	bootOrder, ok := vmRecord.HVMBootParams["order"]
	if !ok {
		return errors.New("unable to read VM HVM boot order")
	}
	data.BootOrder = types.StringValue(bootOrder)

	// only keep the key which configured by user
	data.OtherConfig, err = getOtherConfigFromVMRecord(ctx, vmRecord)
	if err != nil {
		return err
	}

	checkIPDuration, err := strconv.Atoi(vmRecord.OtherConfig["tf_check_ip_timeout"])
	if err != nil {
		return errors.New("unable to convert check_ip_timeout to an int value")
	}
	data.CheckIPTimeout = types.Int64Value(int64(checkIPDuration))

	ip, err := checkIP(ctx, session, vmRecord)
	if err != nil {
		return err
	}
	data.DefaultIP = types.StringValue(ip)

	return nil
}

// Update vmResourceModel base on new vmRecord, except uuid
func updateVMResourceModel(ctx context.Context, session *xenapi.Session, vmRecord xenapi.VMRecord, data *vmResourceModel) error {
	data.NameLabel = types.StringValue(vmRecord.NameLabel)
	data.TemplateName = types.StringValue(vmRecord.OtherConfig["tf_template_name"])
	data.StaticMemMax = types.Int64Value(int64(vmRecord.MemoryStaticMax))
	data.VCPUs = types.Int32Value(int32(vmRecord.VCPUsMax))
	return updateVMResourceModelComputed(ctx, session, vmRecord, data)
}

func getVBDsFromVMRecord(ctx context.Context, session *xenapi.Session, vmRecord xenapi.VMRecord, vbdType xenapi.VbdType) (basetypes.SetValue, []vbdResourceModel, error) {
	var vbdSet []vbdResourceModel
	var setValue basetypes.SetValue

	for _, vbdRef := range vmRecord.VBDs {
		vbdRecord, err := xenapi.VBD.GetRecord(session, vbdRef)
		if err != nil {
			return setValue, vbdSet, errors.New("unable to get VBD record")
		}

		if vbdRecord.Type != vbdType || slices.Contains(getTemplateVBDRefListFromVMRecord(vmRecord), vbdRef) {
			continue
		}

		// for CD type VBD, VDI can be NULL
		vdiUUID := ""
		if vbdRecord.VDI != "OpaqueRef:NULL" {
			vdiRecord, err := xenapi.VDI.GetRecord(session, vbdRecord.VDI)
			if err != nil {
				return setValue, vbdSet, errors.New("unable to get VDI record")
			}
			vdiUUID = vdiRecord.UUID
		}
		vbd := vbdResourceModel{
			VDI:      types.StringValue(vdiUUID),
			VBD:      types.StringValue(string(vbdRef)),
			Bootable: types.BoolValue(vbdRecord.Bootable),
			Mode:     types.StringValue(string(vbdRecord.Mode)),
		}
		vbdSet = append(vbdSet, vbd)
	}

	setValue, diags := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: vbdResourceModelAttrTypes}, vbdSet)
	if diags.HasError() {
		return setValue, vbdSet, errors.New("unable to get VBD set value")
	}

	tflog.Debug(ctx, "-----> setVaule VBD "+setValue.String())
	return setValue, vbdSet, nil
}

func getOtherConfigFromVMRecord(ctx context.Context, vmRecord xenapi.VMRecord) (basetypes.MapValue, error) {
	otherConfig := make(map[string]string)
	for key := range vmRecord.OtherConfig {
		if slices.Contains(strings.Split(vmRecord.OtherConfig["tf_other_config_keys"], ","), key) {
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

type vmMemorySetting struct {
	staticMemMin  int
	staticMemMax  int
	dynamicMemMin int
	dynamicMemMax int
}

func getVMMemory(data vmResourceModel) vmMemorySetting {
	staticMemMax := int(data.StaticMemMax.ValueInt64())
	staticMemMin := staticMemMax
	dynamicMemMin := staticMemMax
	dynamicMemMax := staticMemMax
	if !data.StaticMemMin.IsUnknown() {
		staticMemMin = int(data.StaticMemMin.ValueInt64())
	}
	if !data.DynamicMemMin.IsUnknown() {
		dynamicMemMin = int(data.DynamicMemMin.ValueInt64())
	}
	if !data.DynamicMemMax.IsUnknown() {
		dynamicMemMax = int(data.DynamicMemMax.ValueInt64())
	}

	return vmMemorySetting{staticMemMin, staticMemMax, dynamicMemMin, dynamicMemMax}
}

func setVMMemory(session *xenapi.Session, vmRef xenapi.VMRef, plan vmResourceModel) error {
	memorySetting := getVMMemory(plan)
	err := xenapi.VM.SetMemoryLimits(session, vmRef, memorySetting.staticMemMin, memorySetting.staticMemMax, memorySetting.dynamicMemMin, memorySetting.dynamicMemMax)
	if err != nil {
		return errors.New(err.Error())
	}

	return nil
}

func updateVMMemory(ctx context.Context, session *xenapi.Session, vmRef xenapi.VMRef, plan vmResourceModel, state vmResourceModel) error {
	planMemorySetting := getVMMemory(plan)
	stateMemorySetting := getVMMemory(state)
	if planMemorySetting == stateMemorySetting {
		tflog.Debug(ctx, "---> No memory change, skip update VM Memory. <---")
		return nil
	}
	vmState, err := xenapi.VM.GetPowerState(session, vmRef)
	if err != nil {
		return errors.New(err.Error())
	}
	if vmState == xenapi.VMPowerStateRunning {
		return errors.New("unable to change memory for a running VM")
	}
	err = xenapi.VM.SetMemoryLimits(session, vmRef, planMemorySetting.staticMemMin, planMemorySetting.staticMemMax, planMemorySetting.dynamicMemMin, planMemorySetting.dynamicMemMax)
	if err != nil {
		return errors.New(err.Error())
	}

	return nil
}

func changeVCPUSettings(session *xenapi.Session, vmRef xenapi.VMRef, plan vmResourceModel) error {
	vmPowerState, err := xenapi.VM.GetPowerState(session, vmRef)
	if err != nil {
		return errors.New(err.Error())
	}
	if vmPowerState == xenapi.VMPowerStateRunning {
		return errors.New("unable to change vcpus for a running VM")
	}

	vcpus := int(plan.VCPUs.ValueInt32())
	vcpusAtStartup, err := xenapi.VM.GetVCPUsAtStartup(session, vmRef)
	if err != nil {
		return errors.New(err.Error())
	}
	// VCPU values must satisfy: 0 < VCPUs_at_startup ≤ VCPUs_max
	if vcpusAtStartup > vcpus {
		// reducing VCPUs_at_startup: we need to change this value first, and then the VCPUs_max
		err := xenapi.VM.SetVCPUsAtStartup(session, vmRef, vcpus)
		if err != nil {
			return errors.New(err.Error())
		}
		err = xenapi.VM.SetVCPUsMax(session, vmRef, vcpus)
		if err != nil {
			return errors.New(err.Error())
		}
	} else {
		// increasing VCPUs_at_startup: we need to change the VCPUs_max first
		err := xenapi.VM.SetVCPUsMax(session, vmRef, vcpus)
		if err != nil {
			return errors.New(err.Error())
		}
		err = xenapi.VM.SetVCPUsAtStartup(session, vmRef, vcpus)
		if err != nil {
			return errors.New(err.Error())
		}
	}

	return nil
}

func updateVMCPUs(ctx context.Context, session *xenapi.Session, vmRef xenapi.VMRef, plan vmResourceModel, state vmResourceModel) error {
	if plan.VCPUs == state.VCPUs {
		tflog.Debug(ctx, "---> No vcpus change, skip update VM CPUs. <---")
		return nil
	}
	return changeVCPUSettings(session, vmRef, plan)
}

func updateCorePerSocket(session *xenapi.Session, vmRef xenapi.VMRef, plan vmResourceModel) error {
	platform, err := xenapi.VM.GetPlatform(session, vmRef)
	if err != nil {
		return errors.New(err.Error())
	}
	if plan.CorePerSocket.IsUnknown() {
		// if user doesn't set cores-per-socket and it is not found in template, set it to VCPUs num as the default value
		if _, ok := platform["cores-per-socket"]; !ok {
			platform["cores-per-socket"] = plan.VCPUs.String()
			err := xenapi.VM.SetPlatform(session, vmRef, platform)
			if err != nil {
				return errors.New(err.Error())
			}
		}
	} else {
		coresPerSocket := int(plan.CorePerSocket.ValueInt32())
		vcpus := int(plan.VCPUs.ValueInt32())
		if vcpus%coresPerSocket != 0 {
			return fmt.Errorf("%d cores could not fit to %d cores-per-socket topology", vcpus, coresPerSocket)
		}
		platform["cores-per-socket"] = strconv.Itoa(coresPerSocket)
		err := xenapi.VM.SetPlatform(session, vmRef, platform)
		if err != nil {
			return errors.New(err.Error())
		}
	}

	return nil
}

func updateBootOrder(session *xenapi.Session, vmRef xenapi.VMRef, plan vmResourceModel) error {
	// don't set boot order if it is unknown, using the default value from the template
	if plan.BootOrder.IsUnknown() {
		return nil
	}

	hvmBootParams, err := xenapi.VM.GetHVMBootParams(session, vmRef)
	if err != nil {
		return errors.New(err.Error())
	}
	hvmBootParams["order"] = plan.BootOrder.ValueString()
	err = xenapi.VM.SetHVMBootParams(session, vmRef, hvmBootParams)
	if err != nil {
		return errors.New(err.Error())
	}

	return nil
}

func updateBootMode(session *xenapi.Session, vmRef xenapi.VMRef, plan vmResourceModel) error {
	// don't set boot mode if it is unknown, using the default value from the template
	if plan.BootMode.IsUnknown() {
		return nil
	}

	vmRecord, err := xenapi.VM.GetRecord(session, vmRef)
	if err != nil {
		return errors.New(err.Error())
	}

	secureBoot := "false"
	bootMode := plan.BootMode.ValueString()
	if bootMode == "uefi_security" {
		bootMode = "uefi"
		secureBoot = "true"
	}

	platform := vmRecord.Platform
	platform["secureboot"] = secureBoot
	err = xenapi.VM.SetPlatform(session, vmRef, platform)
	if err != nil {
		return errors.New(err.Error())
	}

	hvmBootParams := vmRecord.HVMBootParams
	hvmBootParams["firmware"] = bootMode
	err = xenapi.VM.SetHVMBootParams(session, vmRef, hvmBootParams)
	if err != nil {
		return errors.New(err.Error())
	}

	return nil
}

func vmResourceModelUpdate(ctx context.Context, session *xenapi.Session, vmRef xenapi.VMRef, plan vmResourceModel, state vmResourceModel) error {
	// set other config before getting the VM record for tf_ fields update
	err := updateOtherConfigFromPlan(ctx, session, vmRef, plan)
	if err != nil {
		return err
	}

	err = xenapi.VM.SetNameLabel(session, vmRef, plan.NameLabel.ValueString())
	if err != nil {
		return errors.New(err.Error())
	}

	err = xenapi.VM.SetNameDescription(session, vmRef, plan.NameDescription.ValueString())
	if err != nil {
		return errors.New(err.Error())
	}

	err = updateVBDs(ctx, plan, state, vmRef, session)
	if err != nil {
		return err
	}

	err = setCDROM(ctx, session, vmRef, plan)
	if err != nil {
		return err
	}

	err = updateVIFs(ctx, plan, state, vmRef, session)
	if err != nil {
		return err
	}

	err = updateVMMemory(ctx, session, vmRef, plan, state)
	if err != nil {
		return err
	}

	err = updateVMCPUs(ctx, session, vmRef, plan, state)
	if err != nil {
		return err
	}

	err = updateCorePerSocket(session, vmRef, plan)
	if err != nil {
		return err
	}

	err = updateBootMode(session, vmRef, plan)
	if err != nil {
		return err
	}

	err = updateBootOrder(session, vmRef, plan)
	if err != nil {
		return err
	}

	err = startVM(session, vmRef, plan)
	if err != nil {
		return err
	}

	return nil
}

func setVMResourceModel(ctx context.Context, session *xenapi.Session, vmRef xenapi.VMRef, plan vmResourceModel) error {
	err := setOtherConfigWhenCreate(session, vmRef)
	if err != nil {
		return err
	}

	// set other config before getting the VM record for tf_ fields update
	err = updateOtherConfigFromPlan(ctx, session, vmRef, plan)
	if err != nil {
		return err
	}

	err = xenapi.VM.SetNameLabel(session, vmRef, plan.NameLabel.ValueString())
	if err != nil {
		return errors.New(err.Error())
	}

	// set name description
	err = xenapi.VM.SetNameDescription(session, vmRef, plan.NameDescription.ValueString())
	if err != nil {
		return errors.New(err.Error())
	}

	// set memory
	err = setVMMemory(session, vmRef, plan)
	if err != nil {
		return err
	}

	// set VCPUs
	err = changeVCPUSettings(session, vmRef, plan)
	if err != nil {
		return err
	}

	err = updateCorePerSocket(session, vmRef, plan)
	if err != nil {
		return err
	}

	// set boot mode
	err = updateBootMode(session, vmRef, plan)
	if err != nil {
		return err
	}

	// set boot order
	err = updateBootOrder(session, vmRef, plan)
	if err != nil {
		return err
	}

	// add hard_drive
	err = createVBDs(ctx, session, vmRef, plan, xenapi.VbdTypeDisk)
	if err != nil {
		return err
	}

	// set CDROM and it should be set after hard_drive to keep device order
	err = setCDROM(ctx, session, vmRef, plan)
	if err != nil {
		return err
	}

	// add network_interface
	err = createVIFs(ctx, session, vmRef, plan)
	if err != nil {
		return err
	}

	err = xenapi.VM.Provision(session, vmRef)
	if err != nil {
		return errors.New(err.Error())
	}

	// reset template flag
	err = xenapi.VM.SetIsATemplate(session, vmRef, false)
	if err != nil {
		return errors.New(err.Error())
	}

	err = startVM(session, vmRef, plan)
	if err != nil {
		return err
	}
	return nil
}

func isValidIpAddress(ip net.IP) bool {
	if ip == nil {
		return false
	}
	return !(ip.IsLinkLocalMulticast() || ip.IsLinkLocalUnicast() || ip.IsLoopback() || ip.IsMulticast())
}

func startVM(session *xenapi.Session, vmRef xenapi.VMRef, plan vmResourceModel) error {
	// start a VM automatically if the check_ip_timeout is set and not equal to 0
	if plan.CheckIPTimeout.IsUnknown() || plan.CheckIPTimeout.ValueInt64() == 0 {
		return nil
	}
	vmPowerState, err := xenapi.VM.GetPowerState(session, vmRef)
	if err != nil {
		return errors.New(err.Error())
	}

	if vmPowerState != xenapi.VMPowerStateRunning {
		err := xenapi.VM.Start(session, vmRef, false, true)
		if err != nil {
			return errors.New(err.Error())
		}
	}

	return nil
}

func checkIP(ctx context.Context, session *xenapi.Session, vmRecord xenapi.VMRecord) (string, error) {
	checkIPTimeout, err := strconv.Atoi(vmRecord.OtherConfig["tf_check_ip_timeout"])
	if err != nil {
		return "", errors.New(err.Error())
	}

	// check_ip_timeout is 0 that means won't need to checkIP, return directly
	if checkIPTimeout == 0 {
		return "", nil
	}

	// set timeout channel to check if IP address is available
	timeoutChan := time.After(time.Duration(checkIPTimeout) * time.Second)
	for {
		select {
		case <-timeoutChan:
			return "", errors.New("get IP timeout in " + vmRecord.OtherConfig["tf_check_ip_timeout"] + " seconds")
		default:
			ip, _ := getIPAddressFromMetrics(session, vmRecord)
			if ip != "" {
				return ip, nil
			}
			tflog.Debug(ctx, "-----> Retry getIPAddressFromMetrics")
			time.Sleep(5 * time.Second)
		}
	}
}

func getIPAddressFromMetrics(session *xenapi.Session, vmRecord xenapi.VMRecord) (string, error) {
	vmGuestMetricRecord, err := xenapi.VMGuestMetrics.GetRecord(session, vmRecord.GuestMetrics)
	if err != nil {
		return "", errors.New(err.Error())
	}

	for k, v := range vmGuestMetricRecord.Networks {
		if strings.HasSuffix(k, "ip") {
			if isValidIpAddress(net.ParseIP(v)) {
				return v, nil
			}
		}
	}

	return "", errors.New("unable to get IP address from metrics")
}

func cleanupVMResource(session *xenapi.Session, vmRef xenapi.VMRef) error {
	// delete VIFs and VBDs, then destroy VM
	vmRecord, err := xenapi.VM.GetRecord(session, vmRef)
	if err != nil {
		return errors.New(err.Error())
	}

	// if VM is runing, stop it first
	if vmRecord.PowerState == xenapi.VMPowerStateRunning {
		err := xenapi.VM.HardShutdown(session, vmRef)
		if err != nil {
			return errors.New(err.Error())
		}
	}

	for _, vifRef := range vmRecord.VIFs {
		err := xenapi.VIF.Destroy(session, vifRef)
		if err != nil {
			return errors.New(err.Error())
		}
	}

	var vdiRefs []xenapi.VDIRef
	for _, vbdRef := range vmRecord.VBDs {
		if slices.Contains(getTemplateVBDRefListFromVMRecord(vmRecord), vbdRef) {
			vdiRef, err := xenapi.VBD.GetVDI(session, vbdRef)
			if err != nil {
				return errors.New(err.Error())
			}
			vdiRefs = append(vdiRefs, vdiRef)
		}
		err := xenapi.VBD.Destroy(session, vbdRef)
		if err != nil {
			return errors.New(err.Error())
		}
	}

	for _, vdiRef := range vdiRefs {
		err := xenapi.VDI.Destroy(session, vdiRef)
		if err != nil {
			return errors.New(err.Error())
		}
	}

	err = xenapi.VM.Destroy(session, vmRef)
	if err != nil {
		return errors.New(err.Error())
	}

	return nil
}

func vmResourceModelUpdateCheck(plan vmResourceModel, state vmResourceModel) error {
	if plan.TemplateName != state.TemplateName {
		return errors.New(`"template_name" doesn't expected to be updated`)
	}
	if !plan.BootMode.IsUnknown() && plan.BootMode != state.BootMode {
		return errors.New(`"boot_mode" doesn't expected to be updated`)
	}
	return nil
}
