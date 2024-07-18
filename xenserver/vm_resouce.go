package xenserver

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
	tflog.Debug(ctx, "---> Create VM resource")
	var plan vmResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// create new resource
	templateRef, err := getFirstTemplate(r.session, plan.TemplateName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get template Ref",
			err.Error(),
		)
		return
	}
	tflog.Debug(ctx, "Clone VM from a template")
	vmRef, err := xenapi.VM.Clone(r.session, templateRef, plan.NameLabel.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to clone VM from template",
			err.Error(),
		)
		return
	}

	err = setVMResourceModel(ctx, r.session, vmRef, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to set VM resource model",
			err.Error(),
		)

		err = cleanupVMResource(r.session, vmRef)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to destroy VM",
				err.Error(),
			)
		}

		return
	}

	// Overwrite data with refreshed resource state
	vmRecord, err := xenapi.VM.GetRecord(r.session, vmRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get VM record",
			err.Error(),
		)

		err = cleanupVMResource(r.session, vmRef)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to destroy VM",
				err.Error(),
			)
		}
		return
	}

	err = updateVMResourceModelComputed(ctx, r.session, vmRecord, &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update VM resource model state",
			err.Error(),
		)

		err = cleanupVMResource(r.session, vmRef)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to destroy VM",
				err.Error(),
			)
		}

		return
	}

	// Save plan into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *vmResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "---> Read VM resource")
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
	tflog.Debug(ctx, "---> Update VM resource")
	var plan, state vmResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.TemplateName != state.TemplateName {
		resp.Diagnostics.AddError(
			"The template name doesn't expected to be updated",
			"plan.TemplateName: "+plan.TemplateName.ValueString()+"  state.TemplateName: "+state.TemplateName.ValueString(),
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

	err = vmResourceModelUpdate(ctx, r.session, vmRef, plan, state)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update VM",
			err.Error(),
		)
		return
	}

	// Overwrite computed data with refreshed resource state
	vmRecord, err := xenapi.VM.GetRecord(r.session, vmRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get VM record",
			err.Error(),
		)
		return
	}

	err = updateVMResourceModelComputed(ctx, r.session, vmRecord, &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update VM resource model state",
			err.Error(),
		)
		return
	}

	// Save updated plan into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *vmResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "---> Delete VM resource")
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

	err = cleanupVMResource(r.session, vmRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to destroy VM",
			err.Error(),
		)
		return
	}
}

func (r *vmResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}
