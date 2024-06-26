package xenserver

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"xenapi"
)

var (
	_ resource.Resource                = &vmResource{}
	_ resource.ResourceWithConfigure   = &vmResource{}
	_ resource.ResourceWithImportState = &vmResource{}
)

func NewVMResource() resource.Resource {
	return &vmResource{}
}

type vmResource struct {
	session *xenapi.Session
}

func (r *vmResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm"
}

func (r *vmResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "VM resource",
		Attributes:          VMSchema(),
	}
}

func (r *vmResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *vmResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data vmResourceModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// create new resource
	templateRef, err := getFirstTemplate(r.session, data.TemplateName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get template Ref",
			err.Error(),
		)
		return
	}
	tflog.Debug(ctx, "Clone VM from a template")
	vmRef, err := xenapi.VM.Clone(r.session, templateRef, data.NameLabel.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to clone VM from template",
			err.Error(),
		)
		return
	}

	err = xenapi.VM.SetIsATemplate(r.session, vmRef, false)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to set VM from a template",
			err.Error(),
		)
		return
	}

	// Set some configure field
	otherConfig, err := getVMOtherConfig(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get VM other config",
			err.Error(),
		)
		return
	}

	err = xenapi.VM.SetOtherConfig(r.session, vmRef, otherConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to set VM other config",
			err.Error(),
		)
		return
	}

	// set VBDs
	_, err = createVBDs(ctx, data.HardDrive, vmRef, r.session)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create VBDs",
			err.Error(),
		)
		return
	}

	// Overwrite data with refreshed resource state
	vmRecord, err := xenapi.VM.GetRecord(r.session, vmRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get VM record",
			err.Error(),
		)
		return
	}

	// Set all computed valueunable to get VDI refs
	data.UUID = types.StringValue(vmRecord.UUID)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vmResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state vmResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Overwrite state with refreshed resource state
	vmRef, err := xenapi.VM.GetByUUID(r.session, state.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get VM ref",
			err.Error(),
		)
		return
	}

	vmRecord, err := xenapi.VM.GetRecord(r.session, vmRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get VM record",
			err.Error(),
		)
		return
	}

	err = updateVMResourceModel(ctx, r.session, vmRecord, &state)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update VM resource model state",
			err.Error(),
		)
		return
	}

	// Save updated state into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *vmResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state vmResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.TemplateName != state.TemplateName {
		resp.Diagnostics.AddError(
			"Unable to change template name",
			"The template name doesn't expected to be updated",
		)
		return
	}

	// Get existing vm record
	vmRef, err := xenapi.VM.GetByUUID(r.session, plan.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get VM ref",
			err.Error(),
		)
		return
	}

	// Update existing vm resource with new plan
	err = xenapi.VM.SetNameLabel(r.session, vmRef, plan.NameLabel.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to set VM name label",
			err.Error(),
		)
		return
	}

	otherConfig, err := getVMOtherConfig(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get VM other config",
			err.Error(),
		)
		return
	}

	err = xenapi.VM.SetOtherConfig(r.session, vmRef, otherConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to set VM other config",
			err.Error(),
		)
		return
	}

	// Judge if the hard drive field in plan has changed, if added, create new VBDs, if removed, destroy VBDs
	planVDIUUIDs := make([]string, 0, len(plan.HardDrive.Elements()))
	diags := plan.HardDrive.ElementsAs(ctx, &planVDIUUIDs, false)
	if diags.HasError() {
		resp.Diagnostics.AddError(
			"Unable to get VDI UUIDs in plan data hard drive attributes",
			"Unable to get VDI UUIDs in plan data hard drive attributes",
		)
		return
	}

	stateVDIUUIDs := make([]string, 0, len(state.HardDrive.Elements()))
	diags = state.HardDrive.ElementsAs(ctx, &stateVDIUUIDs, false)
	if diags.HasError() {
		resp.Diagnostics.AddError(
			"Unable to get VDI UUIDs in state data hard drive attributes",
			"Unable to get VDI UUIDs in state data hard drive attributes",
		)
		return
	}

	err = updateVBDs(planVDIUUIDs, stateVDIUUIDs, vmRef, r.session)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update VBDs",
			err.Error(),
		)
		return
	}

	// Save updated plan into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *vmResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state vmResourceModel
	// Read Terraform prior state state into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// delete resource
	vmRef, err := xenapi.VM.GetByUUID(r.session, state.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get VM ref",
			err.Error(),
		)
		return
	}

	err = xenapi.VM.Destroy(r.session, vmRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to destroy VM",
			err.Error(),
		)
		return
	}
}

func (r *vmResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
