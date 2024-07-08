// Example of data source

package xenserver

import (
	"context"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"xenapi"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &vmDataSource{}
	_ datasource.DataSourceWithConfigure = &vmDataSource{}
)

// NewVMDataSource is a helper function to simplify the provider implementation.
func NewVMDataSource() datasource.DataSource {
	return &vmDataSource{}
}

// vmDataSource is the data source implementation.
type vmDataSource struct {
	session *xenapi.Session
}

// Metadata returns the data source type name.
func (d *vmDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm"
}

func vmSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"uuid": schema.StringAttribute{
			MarkdownDescription: "The UUID of the virtual machine",
			Computed:            true,
		},
		"allowed_operations": schema.ListAttribute{
			MarkdownDescription: "The list of the operations allowed in this state",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"current_operations": schema.MapAttribute{
			MarkdownDescription: "The links each of the running tasks using this object (by reference) to a current_operation enum which describes the nature of the task",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"name_label": schema.StringAttribute{
			MarkdownDescription: "The name of the virtual machine",
			Computed:            true,
		},
		"name_description": schema.StringAttribute{
			MarkdownDescription: "The description of the virtual machine",
			Computed:            true,
		},
		"power_state": schema.StringAttribute{
			MarkdownDescription: "The current power state of the virtual machine",
			Computed:            true,
		},
		"user_version": schema.Int64Attribute{
			MarkdownDescription: "Creators of VMs and templates may store version information here",
			Computed:            true,
		},
		"suspend_vdi": schema.StringAttribute{
			MarkdownDescription: "The VDI that a suspend image is stored on. (Only has meaning if VM is currently suspended)",
			Computed:            true,
		},
		"memory_overhead": schema.Int64Attribute{
			MarkdownDescription: "Virtualization memory overhead (bytes)",
			Computed:            true,
		},
		"is_control_domain": schema.BoolAttribute{
			MarkdownDescription: "True if this is a control domain (domain 0 or a driver domain)",
			Computed:            true,
		},
		"vcpus_params": schema.MapAttribute{
			MarkdownDescription: "Configuration parameters for the selected VCPU policy",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"pci_bus": schema.StringAttribute{
			MarkdownDescription: "PCI bus path for pass-through devices",
			Computed:            true,
		},
		"other_config": schema.MapAttribute{
			MarkdownDescription: "Additional configuration",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"platform": schema.MapAttribute{
			MarkdownDescription: "Platform-specific configuration",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"is_a_template": schema.BoolAttribute{
			MarkdownDescription: "True if this is a template. Template VMs can never be started, they are used only for cloning other VMs",
			Computed:            true,
		},
		"is_default_template": schema.BoolAttribute{
			MarkdownDescription: "True if this is a default template. Default template VMs can never be started or migrated, they are used only for cloning other VMs",
			Computed:            true,
		},
		"resident_on": schema.StringAttribute{
			MarkdownDescription: "The host the VM is currently resident on",
			Computed:            true,
		},
		"scheduled_to_be_resident_on": schema.StringAttribute{
			MarkdownDescription: "The host on which the VM is due to be started/resumed/migrated. This acts as a memory reservation indicator",
			Computed:            true,
		},
		"affinity": schema.StringAttribute{
			MarkdownDescription: "A host which the VM has some affinity for (or NULL). This is used as a hint to the start call when it decides where to run the VM. Resource constraints may cause the VM to be started elsewhere",
			Computed:            true,
		},
		"memory_target": schema.Int64Attribute{
			MarkdownDescription: "Dynamically-set memory target (bytes). The value of this field indicates the current target for memory available to this VM",
			Computed:            true,
		},
		"memory_static_max": schema.Int64Attribute{
			MarkdownDescription: "Statically-set (i.e. absolute) maximum (bytes). The value of this field at VM start time acts as a hard limit of the amount of memory a guest can use. New values only take effect on reboot",
			Computed:            true,
		},
		"memory_dynamic_max": schema.Int64Attribute{
			MarkdownDescription: "Dynamic maximum (bytes) of memory",
			Computed:            true,
		},
		"memory_dynamic_min": schema.Int64Attribute{
			MarkdownDescription: "Dynamic minimum (bytes) of memory",
			Computed:            true,
		},
		"memory_static_min": schema.Int64Attribute{
			MarkdownDescription: "Statically-set (i.e. absolute) mininum (bytes). The value of this field indicates the least amount of memory this VM can boot with without crashing",
			Computed:            true,
		},
		"vcpus_max": schema.Int64Attribute{
			MarkdownDescription: "Max number of VCPUs",
			Computed:            true,
		},
		"vcpus_at_startup": schema.Int64Attribute{
			MarkdownDescription: "Boot number of VCPUs",
			Computed:            true,
		},
		"actions_after_softreboot": schema.StringAttribute{
			MarkdownDescription: "Action to take after soft reboot",
			Computed:            true,
		},
		"actions_after_shutdown": schema.StringAttribute{
			MarkdownDescription: "Action to take after the guest has shutdown itself",
			Computed:            true,
		},
		"actions_after_reboot": schema.StringAttribute{
			MarkdownDescription: "Action to take after the guest has rebooted itself",
			Computed:            true,
		},
		"actions_after_crash": schema.StringAttribute{
			MarkdownDescription: "Action to take if the guest crashes",
			Computed:            true,
		},
		"consoles": schema.ListAttribute{
			MarkdownDescription: "Virtual console devices",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"vifs": schema.ListAttribute{
			MarkdownDescription: "Virtual network interfaces",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"vbds": schema.ListAttribute{
			MarkdownDescription: "Virtual block devices",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"vusbs": schema.ListAttribute{
			MarkdownDescription: "Virtual USB devices",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"crash_dumps": schema.ListAttribute{
			MarkdownDescription: "Crash dumps associated with this VM",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"vtpms": schema.ListAttribute{
			MarkdownDescription: "Virtual TPMs",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"pv_bootloader": schema.StringAttribute{
			MarkdownDescription: "Name of or path to bootloader",
			Computed:            true,
		},
		"pv_kernel": schema.StringAttribute{
			MarkdownDescription: "Path to the kernel",
			Computed:            true,
		},
		"pv_ramdisk": schema.StringAttribute{
			MarkdownDescription: "Path to the initrd",
			Computed:            true,
		},
		"pv_args": schema.StringAttribute{
			MarkdownDescription: "Kernel command-line arguments",
			Computed:            true,
		},
		"pv_bootloader_args": schema.StringAttribute{
			MarkdownDescription: "Miscellaneous arguments for the bootloader",
			Computed:            true,
		},
		"pv_legacy_args": schema.StringAttribute{
			MarkdownDescription: "To make Zurich guests boot",
			Computed:            true,
		},
		"hvm_boot_policy": schema.StringAttribute{
			MarkdownDescription: "HVM boot policy",
			Computed:            true,
		},
		"hvm_boot_params": schema.MapAttribute{
			MarkdownDescription: "HVM boot parameters",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"hvm_shadow_multiplier": schema.Float64Attribute{
			MarkdownDescription: "Multiplier applied to the amount of shadow that will be made available to the guest",
			Computed:            true,
		},
		"domid": schema.Int64Attribute{
			MarkdownDescription: "Domain ID (if available, -1 otherwise)",
			Computed:            true,
		},
		"domarch": schema.StringAttribute{
			MarkdownDescription: "Domain architecture (if available, null string otherwise)",
			Computed:            true,
		},
		"last_boot_cpu_flags": schema.MapAttribute{
			MarkdownDescription: "Describes the CPU flags on which the VM was last booted",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"metrics": schema.StringAttribute{
			MarkdownDescription: "Metrics associated with this VM",
			Computed:            true,
		},
		"guest_metrics": schema.StringAttribute{
			MarkdownDescription: "Metrics associated with the running guest",
			Computed:            true,
		},
		"last_booted_record": schema.StringAttribute{
			MarkdownDescription: "Marshalled value containing VM record at time of last boot",
			Computed:            true,
		},
		"recommendations": schema.StringAttribute{
			MarkdownDescription: "An XML specification of recommended values and ranges for properties of this VM",
			Computed:            true,
		},
		"xenstore_data": schema.MapAttribute{
			MarkdownDescription: "Data to be inserted into the xenstore tree (/local/domain/<domid>/vm-data) after the VM is created",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"ha_always_run": schema.BoolAttribute{
			MarkdownDescription: "If true then the system will attempt to keep the VM running as much as possible",
			Computed:            true,
		},
		"ha_restart_priority": schema.StringAttribute{
			MarkdownDescription: "Has possible values: 'best-effort' meaning 'try to restart this VM if possible but don't consider the Pool to be overcommitted if this is not possible'; 'restart' meaning 'this VM should be restarted'; '' meaning 'do not try to restart this VM'",
			Computed:            true,
		},
		"is_a_snapshot": schema.BoolAttribute{
			MarkdownDescription: "True if this is a snapshot. Snapshotted VMs can never be started, they are used only for cloning other VMs",
			Computed:            true,
		},
		"snapshot_of": schema.StringAttribute{
			MarkdownDescription: "Ref pointing to the VM this snapshot is of",
			Computed:            true,
		},
		"snapshots": schema.ListAttribute{
			MarkdownDescription: "List pointing to all the VM snapshots",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"snapshot_time": schema.StringAttribute{
			MarkdownDescription: "Date/time when this snapshot was created",
			Computed:            true,
		},
		"transportable_snapshot_id": schema.StringAttribute{
			MarkdownDescription: "Transportable ID of the snapshot VM",
			Computed:            true,
		},
		"blobs": schema.MapAttribute{
			MarkdownDescription: "Binary blobs associated with this VM",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"tags": schema.ListAttribute{
			MarkdownDescription: "User-specified tags for categorization purposes",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"blocked_operations": schema.MapAttribute{
			MarkdownDescription: "List of operations which have been explicitly blocked and an error code",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"snapshot_info": schema.MapAttribute{
			MarkdownDescription: "Human-readable information concerning this snapshot",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"snapshot_metadata": schema.StringAttribute{
			MarkdownDescription: "Metadata concerning this snapshot",
			Computed:            true,
		},
		"parent": schema.StringAttribute{
			MarkdownDescription: "Ref pointing to the parent of this VM",
			Computed:            true,
		},
		"children": schema.ListAttribute{
			MarkdownDescription: "List pointing to all the children of this VM",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"bios_strings": schema.MapAttribute{
			MarkdownDescription: "BIOS strings",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"protection_policy": schema.StringAttribute{
			MarkdownDescription: "Ref pointing to a protection policy for this VM",
			Computed:            true,
		},
		"is_snapshot_from_vmpp": schema.BoolAttribute{
			MarkdownDescription: "True if this snapshot was created by the protection policy",
			Computed:            true,
		},
		"snapshot_schedule": schema.StringAttribute{
			MarkdownDescription: "Ref pointing to a snapshot schedule for this VM",
			Computed:            true,
		},
		"is_vmss_snapshot": schema.BoolAttribute{
			MarkdownDescription: "True if this snapshot was created by the snapshot schedule",
			Computed:            true,
		},
		"appliance": schema.StringAttribute{
			MarkdownDescription: "The appliance to which this VM belongs",
			Computed:            true,
		},
		"start_delay": schema.Int64Attribute{
			MarkdownDescription: "The delay to wait before proceeding to the next order in the startup sequence (seconds)",
			Computed:            true,
		},
		"shutdown_delay": schema.Int64Attribute{
			MarkdownDescription: "The delay to wait before proceeding to the next order in the shutdown sequence (seconds)",
			Computed:            true,
		},
		"order": schema.Int64Attribute{
			MarkdownDescription: "The point in the startup or shutdown sequence at which this VM will be started",
			Computed:            true,
		},
		"vgpus": schema.ListAttribute{
			MarkdownDescription: "Virtual GPUs",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"attached_pcis": schema.ListAttribute{
			MarkdownDescription: "Currently passed-through PCI devices",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"suspend_sr": schema.StringAttribute{
			MarkdownDescription: "The SR on which a suspend image is stored",
			Computed:            true,
		},
		"version": schema.Int64Attribute{
			MarkdownDescription: "The number of times this VM has been recovered",
			Computed:            true,
		},
		"generation_id": schema.StringAttribute{
			MarkdownDescription: "Generation ID of the VM",
			Computed:            true,
		},
		"hardware_platform_version": schema.Int64Attribute{
			MarkdownDescription: "The host virtual hardware platform version the VM can run on",
			Computed:            true,
		},
		"has_vendor_device": schema.BoolAttribute{
			MarkdownDescription: "When an HVM guest starts, this controls the presence of the emulated C000 PCI device which triggers Windows Update to fetch or update PV drivers",
			Computed:            true,
		},
		"requires_reboot": schema.BoolAttribute{
			MarkdownDescription: "Indicates whether a VM requires a reboot in order to update its configuration, e.g. its memory allocation",
			Computed:            true,
		},
		"reference_label": schema.StringAttribute{
			MarkdownDescription: "Textual reference to the template used to create a VM. This can be used by clients in need of an immutable reference to the template since the latter's uuid and name_label may change, for example, after a package installation or upgrade",
			Computed:            true,
		},
		"domain_type": schema.StringAttribute{
			MarkdownDescription: "The type of domain that will be created when the VM is started",
			Computed:            true,
		},
		"nvram": schema.MapAttribute{
			MarkdownDescription: "Initial value for guest NVRAM (containing UEFI variables, etc). Cannot be changed while the VM is running",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"pending_guidances": schema.ListAttribute{
			MarkdownDescription: "The set of pending mandatory guidances after applying updates, which must be applied, as otherwise there may be e.g. VM failures",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"pending_guidances_recommended": schema.ListAttribute{
			MarkdownDescription: "The set of pending recommended guidances after applying updates, which most users should follow to make the updates effective, but if not followed, will not cause a failure",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"pending_guidances_full": schema.ListAttribute{
			MarkdownDescription: "The set of pending full guidances after applying updates, which a user should follow to make some updates, e.g. specific hardware drivers or CPU features, fully effective, but the 'average user' doesn't need to",
			Computed:            true,
			ElementType:         types.StringType,
		},
	}
}

func (d *vmDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Provides information about the virtual machine (VM) of XenServer",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the virtual machine",
				Optional:            true,
			},
			"name_label": schema.StringAttribute{
				MarkdownDescription: "The name of the virtual machine",
				Optional:            true,
			},
			"data_items": schema.ListNestedAttribute{
				MarkdownDescription: "The return items of virtual machines",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: vmSchema(),
				},
			},
		},
	}
}

func (d *vmDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	session, ok := req.ProviderData.(*xenapi.Session)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *xenapi.Session, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	d.session = session
}

// Read refreshes the Terraform state with the latest data.
func (d *vmDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data vmDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vmRecords, err := xenapi.VM.GetAllRecords(d.session)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read VM records",
			err.Error(),
		)
		return
	}

	var vmItems []vmRecordData
	for _, vmRecord := range vmRecords {
		if !data.NameLabel.IsNull() && vmRecord.NameLabel != data.NameLabel.ValueString() {
			continue
		}

		if !data.UUID.IsNull() && vmRecord.UUID != data.UUID.ValueString() {
			continue
		}

		var vmItem vmRecordData
		err := updateVMRecordData(ctx, vmRecord, &vmItem)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to update VM data",
				err.Error(),
			)
			return
		}
		vmItems = append(vmItems, vmItem)
	}

	sort.Slice(vmItems, func(i, j int) bool {
		return vmItems[i].UUID.ValueString() < vmItems[j].UUID.ValueString()
	})
	data.DataItems = vmItems

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
