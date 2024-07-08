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
	_ resource.Resource                = &srResource{}
	_ resource.ResourceWithConfigure   = &srResource{}
	_ resource.ResourceWithImportState = &srResource{}
)

func NewSRResource() resource.Resource {
	return &srResource{}
}

// srResource defines the resource implementation.
type srResource struct {
	session *xenapi.Session
}

func (r *srResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sr"
}

func (r *srResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "SR resource",
		Attributes: map[string]schema.Attribute{
			"name_label": schema.StringAttribute{
				MarkdownDescription: "The name of the storage repository",
				Required:            true,
			},
			"name_description": schema.StringAttribute{
				MarkdownDescription: `The human-readable description of the storage repository, default to be ""`,
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"type": schema.StringAttribute{
				MarkdownDescription: `The type of the storage repository, default to be "dummy"`,
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("dummy"),
			},
			"content_type": schema.StringAttribute{
				MarkdownDescription: `The type of the SR's content, if required (e.g. ISOs), default to be ""`,
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"shared": schema.BoolAttribute{
				MarkdownDescription: `True if this SR is (capable of being) shared between multiple hosts, default to be false`,
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"sm_config": schema.MapAttribute{
				MarkdownDescription: "The SM dependent data, default to be {}",
				Optional:            true,
				Computed:            true,
				Default:             mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
				ElementType:         types.StringType,
			},
			"device_config": schema.MapAttribute{
				MarkdownDescription: "The device config that will be passed to backend SR driver, default to be {}",
				Optional:            true,
				Computed:            true,
				Default:             mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
				ElementType:         types.StringType,
			},
			"host": schema.StringAttribute{
				MarkdownDescription: "The UUID of the host to create/make the SR on",
				Optional:            true,
				Computed:            true,
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the storage repository",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the storage repository",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Set the parameter of the resource, pass value from provider
func (r *srResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *srResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data srResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating SR ...")
	params, err := getSRCreateParams(ctx, r.session, data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get SR create params",
			err.Error(),
		)
		return
	}
	srRef, err := xenapi.SR.Create(r.session, params.Host, params.DeviceConfig, params.PhysicalSize, params.NameLabel, params.NameDescription, params.TypeKey, params.ContentType, params.Shared, params.SmConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create SR",
			err.Error(),
		)
		return
	}
	srRecord, pbdRecord, err := getSRRecordAndPBDRecord(r.session, srRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get SR or PBDrecord",
			err.Error(),
		)
		return
	}
	err = updateSRResourceModelComputed(ctx, r.session, srRecord, pbdRecord, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update the computed fields of SRResourceModel",
			err.Error(),
		)
		return
	}
	tflog.Debug(ctx, "SR created")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read data from State, retrieve the resource's information, update to State
// terraform import
func (r *srResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data srResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Overwrite data with refreshed resource state
	srRef, err := xenapi.SR.GetByUUID(r.session, data.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get SR ref",
			err.Error(),
		)
		return
	}
	srRecord, pbdRecord, err := getSRRecordAndPBDRecord(r.session, srRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get SR or PBDrecord",
			err.Error(),
		)
		return
	}
	err = updateSRResourceModel(ctx, r.session, srRecord, pbdRecord, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update the fields of SRResourceModel",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *srResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state srResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Checking if configuration changes are allowed
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := srResourceModelUpdateCheck(plan, state)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error update xenserver_sr configuration",
			err.Error(),
		)
		return
	}

	// Update the resource with new configuration
	srRef, err := xenapi.SR.GetByUUID(r.session, plan.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get SR ref",
			err.Error(),
		)
		return
	}
	err = srResourceModelUpdate(ctx, r.session, srRef, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update SR resource",
			err.Error(),
		)
		return
	}
	srRecord, pbdRecord, err := getSRRecordAndPBDRecord(r.session, srRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get SR or PBDrecord",
			err.Error(),
		)
		return
	}
	err = updateSRResourceModelComputed(ctx, r.session, srRecord, pbdRecord, &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update the computed fields of SRResourceModel",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *srResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data srResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := srDelete(r.session, data.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error delete SR",
			err.Error(),
		)
		return
	}
}

func (r *srResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}
