// Example of resource

package xenserver

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"xenapi"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &VMResource{}
	_ resource.ResourceWithConfigure   = &VMResource{}
	_ resource.ResourceWithImportState = &VMResource{}
)

func NewVMResource() resource.Resource {
	return &VMResource{}
}

// VMResource defines the resource implementation.
type VMResource struct {
	session *xenapi.Session
}

// VMResourceModel describes the resource data model.
type VMResourceModel struct {
	NameLabel    types.String `tfsdk:"name_label"`
	TemplateName types.String `tfsdk:"template_name"`
	OtherConfig  types.Map    `tfsdk:"other_config"`
	Snapshots    types.List   `tfsdk:"snapshots"`
	UUID         types.String `tfsdk:"id"`
}

// Set the resource name
func (r *VMResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm"
}

// Set the defined datamodel of the resource
func (r *VMResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "VM resource",
		Attributes: map[string]schema.Attribute{
			// required
			"name_label": schema.StringAttribute{
				MarkdownDescription: "The name of the virtual machine",
				Required:            true,
			},
			"template_name": schema.StringAttribute{
				MarkdownDescription: "The template name of the virtual machine which cloned from",
				Required:            true,
				// The resource will be removed and created again if the value of the attribute changes
				// PlanModifiers: []planmodifier.String{
				// 	stringplanmodifier.RequiresReplace(),
				// },
			},
			// optional
			"other_config": schema.MapAttribute{
				MarkdownDescription: "The other config of the virtual machine",
				Optional:            true,
				ElementType:         types.StringType,
			},
			// read only
			"snapshots": schema.ListAttribute{
				MarkdownDescription: "The all snapshots of the virtual machine",
				// If Required and Optional are both false, Computed must be true, and the attribute will be considered "read only"
				Computed:    true,
				ElementType: types.StringType,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "UUID of the virtual machine",
				Computed:            true,
				// attributes which are not configurable and that should not show updates from the existing state value
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Set the parameter of the resource, pass value from provider
func (r *VMResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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
	r.session = session
}

// Read data from Plan, create resource, get data from new source, set to State
// terraform plan/apply
func (r *VMResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VMResourceModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// create new reource
	tflog.Debug(ctx, "Get a template")
	templateRef, err := GetFirstTemplate(r.session, data.TemplateName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error get template Ref",
			"Could not find a template Ref, unexpected error: "+err.Error(),
		)
		return
	}
	tflog.Debug(ctx, "Clone vm from a template")
	vmRef, err := xenapi.VM.Clone(r.session, templateRef, data.NameLabel.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error clone vm from template",
			"Could not clone vm, unexpected error: "+err.Error(),
		)
		return
	}

	// Set some configure field
	otherConfig, err := GetVMOtherConfig(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error on other config",
			"Unexpected error: "+err.Error(),
		)
		return
	}
	err = xenapi.VM.SetOtherConfig(r.session, vmRef, otherConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error set other config",
			"Could not set other config, unexpected error: "+err.Error(),
		)
		return
	}

	// Overwrite data with refreshed resource state
	vmRecord, err := xenapi.VM.GetRecord(r.session, vmRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error get vm record",
			"Could not get vm record, unexpected error: "+err.Error(),
		)
		return
	}
	// Set all computed values
	data.UUID = types.StringValue(vmRecord.UUID)
	err = UpdateVMResourceModelComputed(ctx, vmRecord, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error update data",
			err.Error(),
		)
		return
	}

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a vm resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read data from State, retrieve the resource's information, update to State
// terraform import
func (r *VMResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VMResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Overwrite data with refreshed resource state
	vmRef, err := xenapi.VM.GetByUUID(r.session, data.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error get vm ref",
			"Could not get vm ref, unexpected error: "+err.Error(),
		)
		return
	}
	vmRecord, err := xenapi.VM.GetRecord(r.session, vmRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error get vm record",
			"Could not get vm record, unexpected error: "+err.Error(),
		)
		return
	}
	err = UpdateVMResourceModel(ctx, vmRecord, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error update data",
			err.Error(),
		)
		return
	}
	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read data from Plan, update resource configuration, Set to State
// terraform plan/apply (+2)
func (r *VMResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data VMResourceModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var dataState VMResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &dataState)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if data.TemplateName != dataState.TemplateName {
		resp.Diagnostics.AddError(
			"Error change template name",
			"The template name doesn't expected to be updated",
		)
		return
	}

	// Get existing vm record
	vmRef, err := xenapi.VM.GetByUUID(r.session, data.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error get vm ref",
			"Could not get vm ref, unexpected error: "+err.Error(),
		)
		return
	}
	// Update existing vm resource with new plan
	err = xenapi.VM.SetNameLabel(r.session, vmRef, data.NameLabel.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error set name label",
			"Could not set name label, unexpected error: "+err.Error(),
		)
		return
	}
	otherConfig, err := GetVMOtherConfig(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error on other config",
			"Unexpected error: "+err.Error(),
		)
		return
	}
	err = xenapi.VM.SetOtherConfig(r.session, vmRef, otherConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error set other config",
			"Could not set other config, unexpected error: "+err.Error(),
		)
		return
	}

	// Overwrite computed data with refreshed resource state
	vmRecord, err := xenapi.VM.GetRecord(r.session, vmRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error get vm record",
			"Could not get vm record, unexpected error: "+err.Error(),
		)
		return
	}
	err = UpdateVMResourceModelComputed(ctx, vmRecord, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error update data",
			err.Error(),
		)
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read data from State, delete resource
// terraform destroy
func (r *VMResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VMResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// delete resource
	vmRef, err := xenapi.VM.GetByUUID(r.session, data.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error get vm ref",
			"Could not get vm ref, unexpected error: "+err.Error(),
		)
		return
	}
	err = xenapi.VM.Destroy(r.session, vmRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error destroy vm",
			"Could not destroy vm, unexpected error: "+err.Error(),
		)
		return
	}
}

// Import existing resource with id, call Read()
// terraform import
func (r *VMResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
