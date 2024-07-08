package xenserver

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"xenapi"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &snapshotResource{}
	_ resource.ResourceWithConfigure   = &snapshotResource{}
	_ resource.ResourceWithImportState = &snapshotResource{}
)

func NewSnapshotResource() resource.Resource {
	return &snapshotResource{}
}

// snapshotResource defines the resource implementation.
type snapshotResource struct {
	session *xenapi.Session
}

func (r *snapshotResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_snapshot"
}

func (r *snapshotResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "VM snapshot resource",
		Attributes: map[string]schema.Attribute{
			"name_label": schema.StringAttribute{
				MarkdownDescription: "The name of the snapshot",
				Required:            true,
			},
			"vm_uuid": schema.StringAttribute{
				MarkdownDescription: `Snapshot from the VM with the given UUID`,
				Required:            true,
			},
			"with_memory": schema.BoolAttribute{
				MarkdownDescription: `True if snapshot with the VM's memory(VM must in running state), default to be false`,
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the snapshot",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the snapshot",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Set the parameter of the resource, pass value from provider
func (r *snapshotResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	session, ok := req.ProviderData.(*xenapi.Session)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *xenapi.Session, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.session = session
}

func (r *snapshotResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data snapshotResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating snapshot...")
	vmRef, err := xenapi.VM.GetByUUID(r.session, data.VM.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get VM by UUID",
			err.Error(),
		)
		return
	}
	vmPowerState, err := xenapi.VM.GetPowerState(r.session, vmRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get VM power state",
			err.Error(),
		)
		return
	}
	var snapshotRef xenapi.VMRef
	if !data.WithMemory.IsNull() && data.WithMemory.ValueBool() {
		if vmPowerState != xenapi.VMPowerStateRunning {
			resp.Diagnostics.AddError(
				"VM in wrong state",
				"VM must be in running state to create snapshot with memory",
			)
			return
		}
		snapshotRef, err = xenapi.VM.Checkpoint(r.session, vmRef, data.NameLabel.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to create snapshot with memory",
				err.Error(),
			)
			return
		}
	} else {
		snapshotRef, err = xenapi.VM.Snapshot(r.session, vmRef, data.NameLabel.ValueString(), []xenapi.VDIRef{})
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to create snapshot",
				err.Error(),
			)
			return
		}
	}

	snapshotRecord, err := xenapi.VM.GetRecord(r.session, snapshotRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get snapshot record",
			err.Error(),
		)
		err = cleanupSnapshotResource(r.session, snapshotRef)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error cleaning up snapshot resource",
				err.Error(),
			)
		}
		return
	}
	err = updateSnapshotResourceModelComputed(snapshotRecord, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update the computed fields of snapshotResourceModel",
			err.Error(),
		)
		err = cleanupSnapshotResource(r.session, snapshotRef)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error cleaning up snapshot resource",
				err.Error(),
			)
		}
		return
	}
	tflog.Debug(ctx, "Snapshot created")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *snapshotResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data snapshotResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Overwrite data with refreshed resource state
	snapshotRef, err := xenapi.VM.GetByUUID(r.session, data.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get snapshot by UUID",
			err.Error(),
		)
		return
	}
	snapshotRecord, err := xenapi.VM.GetRecord(r.session, snapshotRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get snapshot record",
			err.Error(),
		)
		return
	}

	if !snapshotRecord.IsASnapshot {
		resp.Diagnostics.AddError(
			"Resource is not a snapshot",
			"Resource is not a snapshot",
		)
		return
	}

	err = updateSnapshotResourceModel(r.session, snapshotRecord, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update the fields of snapshotResourceModel",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *snapshotResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state snapshotResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := snapshotResourceModelUpdateCheck(plan, state)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error update xenserver_snapshot configuration",
			err.Error(),
		)
		return
	}

	// Update the resource with new configuration
	snapshotRef, err := xenapi.VM.GetByUUID(r.session, plan.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get snapshot by UUID",
			err.Error(),
		)
		return
	}
	err = snapshotResourceModelUpdate(r.session, snapshotRef, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update snapshot resource",
			err.Error(),
		)
		return
	}
	snapshotRecord, err := xenapi.VM.GetRecord(r.session, snapshotRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get snapshot record",
			err.Error(),
		)
		return
	}
	err = updateSnapshotResourceModelComputed(snapshotRecord, &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update the computed fields of snapshotResourceModel",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *snapshotResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data snapshotResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting snapshot...")
	snapshotRef, err := xenapi.VM.GetByUUID(r.session, data.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get snapshot by UUID",
			err.Error(),
		)
		return
	}
	powerState, err := xenapi.VM.GetPowerState(r.session, snapshotRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get snapshot power state",
			err.Error(),
		)
		return
	}
	if powerState == xenapi.VMPowerStateSuspended {
		err = xenapi.VM.HardShutdown(r.session, snapshotRef)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to hard shutdown snapshot",
				err.Error(),
			)
			return
		}
	}

	err = cleanupSnapshotResource(r.session, snapshotRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete snapshot",
			err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "Snapshot deleted")
}

func (r *snapshotResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}
