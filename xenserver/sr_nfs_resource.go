package xenserver

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"xenapi"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &nfsResource{}
	_ resource.ResourceWithConfigure   = &nfsResource{}
	_ resource.ResourceWithImportState = &nfsResource{}
)

func NewNFSResource() resource.Resource {
	return &nfsResource{}
}

// nfsResource defines the resource implementation.
type nfsResource struct {
	session *xenapi.Session
}

func (r *nfsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sr_nfs"
}

func (r *nfsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "NFS SR resource",
		Attributes: map[string]schema.Attribute{
			"name_label": schema.StringAttribute{
				MarkdownDescription: "The name of the NFS storage repository",
				Required:            true,
			},
			"name_description": schema.StringAttribute{
				MarkdownDescription: `The human-readable description of the NFS storage repository, default to be ""`,
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"storage_location": schema.StringAttribute{
				MarkdownDescription: `The server and server patch of the NFS storage repository, for example: "10.70.58.9:/xenrtnfs"`,
				Required:            true,
			},
			"version": schema.StringAttribute{
				MarkdownDescription: `The version of NFS storage repository, for example: "3"`,
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("3", "4"),
				},
			},
			"advanced_options": schema.StringAttribute{
				MarkdownDescription: `The advanced options of the NFS storage repository, default to be ""`,
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the NFS storage repository",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The test id of the NFS storage repository",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Set the parameter of the resource, pass value from provider
func (r *nfsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *nfsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data nfsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating NFS SR...")
	params, err := getNFSCreateParams(r.session, data)
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
			"Unable to get SR or PBD record",
			err.Error(),
		)
		err = cleanupSRResource(r.session, srRef)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error cleaning up SR resource",
				err.Error(),
			)
		}
		return
	}
	err = updateNFSResourceModelComputed(srRecord, pbdRecord, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update the computed fields of NFSResourceModel",
			err.Error(),
		)
		err = cleanupSRResource(r.session, srRef)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error cleaning up SR resource",
				err.Error(),
			)
		}
		return
	}
	tflog.Debug(ctx, "NFS SR created")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read data from State, retrieve the resource's information, update to State
// terraform import
func (r *nfsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data nfsResourceModel
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
	err = updateNFSResourceModel(srRecord, pbdRecord, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update the fields of NFSResourceModel",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *nfsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state nfsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Checking if configuration changes are allowed
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := nfsResourceModelUpdateCheck(plan, state)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error update xenserver_sr_nfs configuration",
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
	err = nfsResourceModelUpdate(r.session, srRef, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update NFS SR resource",
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
	err = updateNFSResourceModelComputed(srRecord, pbdRecord, &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update the computed fields of NFSResourceModel",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *nfsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data nfsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	srRef, err := xenapi.SR.GetByUUID(r.session, data.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get SR ref",
			err.Error(),
		)
		return
	}
	err = cleanupSRResource(r.session, srRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete NFS SR",
			err.Error(),
		)
		return
	}
}

func (r *nfsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}
