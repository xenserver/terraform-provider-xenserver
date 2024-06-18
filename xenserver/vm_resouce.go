// Example of resource

package xenserver

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"xenapi"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &vmResource{}
	_ resource.ResourceWithConfigure   = &vmResource{}
	_ resource.ResourceWithImportState = &vmResource{}
)

func NewVMResource() resource.Resource {
	return &vmResource{}
}

// vmResource defines the resource implementation.
type vmResource struct {
	session *xenapi.Session
}

// Set the resource name
func (r *vmResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm"
}

// Set the defined data model of the resource
func (r *vmResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
				Computed:            true,
				ElementType:         types.StringType,
				Default:             mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
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
func (r *vmResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Read data from Plan, create resource, get data from new source, set to State
// terraform plan/apply
func (r *vmResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data vmResourceModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// create new resource
	tflog.Debug(ctx, "Get a template")
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

	// Overwrite data with refreshed resource state
	vmRecord, err := xenapi.VM.GetRecord(r.session, vmRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get VM record",
			err.Error(),
		)
		return
	}
	// Set all computed values
	data.UUID = types.StringValue(vmRecord.UUID)
	err = updateVMResourceModelComputed(ctx, vmRecord, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update VM resource model computed fields",
			err.Error(),
		)
		return
	}

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "VM created")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read data from State, retrieve the resource's information, update to State
// terraform import
func (r *vmResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data vmResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Overwrite data with refreshed resource state
	vmRef, err := xenapi.VM.GetByUUID(r.session, data.UUID.ValueString())
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
	err = updateVMResourceModel(ctx, vmRecord, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update VM resource model data",
			err.Error(),
		)
		return
	}
	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read data from Plan, update resource configuration, Set to State
// terraform plan/apply (+2)
func (r *vmResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data vmResourceModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var dataState vmResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &dataState)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if data.TemplateName != dataState.TemplateName {
		resp.Diagnostics.AddError(
			"Unable to change template name",
			"The template name doesn't expected to be updated",
		)
		return
	}

	// Get existing vm record
	vmRef, err := xenapi.VM.GetByUUID(r.session, data.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get VM ref",
			err.Error(),
		)
		return
	}
	// Update existing vm resource with new plan
	err = xenapi.VM.SetNameLabel(r.session, vmRef, data.NameLabel.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to set VM name label",
			err.Error(),
		)
		return
	}
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

	// Overwrite computed data with refreshed resource state
	vmRecord, err := xenapi.VM.GetRecord(r.session, vmRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get VM record",
			err.Error(),
		)
		return
	}
	err = updateVMResourceModelComputed(ctx, vmRecord, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update VM resource model computed fields",
			err.Error(),
		)
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read data from State, delete resource
// terraform destroy
func (r *vmResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data vmResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// delete resource
	vmRef, err := xenapi.VM.GetByUUID(r.session, data.UUID.ValueString())
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

// Import existing resource with id, call Read()
// terraform import
func (r *vmResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
