package xenserver

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"xenapi"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &vdiResource{}
	_ resource.ResourceWithConfigure   = &vdiResource{}
	_ resource.ResourceWithImportState = &vdiResource{}
)

func NewVDIResource() resource.Resource {
	return &vdiResource{}
}

// vdiResource defines the resource implementation.
type vdiResource struct {
	session *xenapi.Session
}

func (r *vdiResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vdi"
}

func (r *vdiResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "VDI resource",
		Attributes: map[string]schema.Attribute{
			"name_label": schema.StringAttribute{
				MarkdownDescription: "The name of the virtual disk image",
				Required:            true,
			},
			"name_description": schema.StringAttribute{
				MarkdownDescription: `The human-readable description of the virtual disk image, default to be ""`,
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"sr_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the storage repository",
				Required:            true,
			},
			"virtual_size": schema.Int64Attribute{
				MarkdownDescription: "The size of virtual disk image (in bytes)",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: `The type of the virtual disk image, default to be "user"`,
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("user"),
			},
			"sharable": schema.BoolAttribute{
				MarkdownDescription: `True if this disk may be shared, default to be false`,
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"read_only": schema.BoolAttribute{
				MarkdownDescription: `True if this SR is (capable of being) shared between multiple hosts, default to be false`,
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"other_config": schema.MapAttribute{
				MarkdownDescription: "The additional configuration, default to be {}",
				Optional:            true,
				Computed:            true,
				Default:             mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
				ElementType:         types.StringType,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The UUID of the virtual disk image",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Set the parameter of the resource, pass value from provider
func (r *vdiResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *vdiResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data vdiResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating VDI...")
	record, err := getVDICreateParams(ctx, r.session, data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get VDI create params",
			err.Error(),
		)
		return
	}
	vdiRef, err := xenapi.VDI.Create(r.session, record)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create VDI",
			err.Error(),
		)
		return
	}
	vdiRecord, err := xenapi.VDI.GetRecord(r.session, vdiRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get VDI record",
			err.Error(),
		)
		return
	}
	err = updateVDIResourceModelComputed(ctx, vdiRecord, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update the computed fields of VDIResourceModel",
			err.Error(),
		)
		return
	}
	tflog.Debug(ctx, "VDI created")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vdiResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data vdiResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Overwrite data with refreshed resource state
	vdiRef, err := xenapi.VDI.GetByUUID(r.session, data.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get VDI ref",
			err.Error(),
		)
		return
	}
	vdiRecord, err := xenapi.VDI.GetRecord(r.session, vdiRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get VDI record",
			err.Error(),
		)
		return
	}
	err = updateVDIResourceModel(ctx, r.session, vdiRecord, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update the fields of VDIResourceModel",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vdiResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data vdiResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Checking if configuration changes are allowed
	var dataState vdiResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &dataState)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := vdiResourceModelUpdateCheck(data, dataState)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error update xenserver_vdi configuration",
			err.Error(),
		)
		return
	}

	// Update the resource with new configuration
	vdiRef, err := xenapi.VDI.GetByUUID(r.session, data.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get VDI ref",
			err.Error(),
		)
		return
	}
	err = vdiResourceModelUpdate(ctx, r.session, vdiRef, data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update VDI resource",
			err.Error(),
		)
		return
	}
	vdiRecord, err := xenapi.VDI.GetRecord(r.session, vdiRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get VDI record",
			err.Error(),
		)
		return
	}
	err = updateVDIResourceModelComputed(ctx, vdiRecord, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update the computed fields of VDIResourceModel",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vdiResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data vdiResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vdiRef, err := xenapi.VDI.GetByUUID(r.session, data.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get VDI ref",
			err.Error(),
		)
		return
	}
	err = xenapi.VDI.Destroy(r.session, vdiRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to destroy VDI",
			err.Error(),
		)
		return
	}
}

func (r *vdiResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
