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
	_ resource.Resource                = &smbResource{}
	_ resource.ResourceWithConfigure   = &smbResource{}
	_ resource.ResourceWithImportState = &smbResource{}
)

func NewSMBResource() resource.Resource {
	return &smbResource{}
}

// smbResource defines the resource implementation.
type smbResource struct {
	session *xenapi.Session
}

func (r *smbResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sr_smb"
}

func (r *smbResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Provides an SMB storage repository resource.",
		Attributes: map[string]schema.Attribute{
			"name_label": schema.StringAttribute{
				MarkdownDescription: "The name of the SMB storage repository.",
				Required:            true,
			},
			"name_description": schema.StringAttribute{
				MarkdownDescription: "The description of the SMB storage repository, default to be `\"\"`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of the SMB storage repository, default to be `\"smb\"`." + "<br />" +
					"Can be set as `\"smb\"` or `\"iso\"`." +
					"\n\n-> **Note:** `type` is not allowed to be updated.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("smb"),
				Validators: []validator.String{
					stringvalidator.OneOf("smb", "iso"),
				},
			},
			"storage_location": schema.StringAttribute{
				MarkdownDescription: "The server and server path of the SMB storage repository." + "<br />" +
					"Follow the format `\"\\\\\\\\server\\\\path\"`." +
					"\n\n-> **Note:** `storage_location` is not allowed to be updated.",
				Required: true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "The username of the SMB storage repository. Used when creating the SR.",
				Optional:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "The password of the SMB storage repository. Used when creating the SR." +
					"\n\n-> **Note:** This password will be stored in terraform state file, follow document [Sensitive values in state](https://developer.hashicorp.com/terraform/tutorials/configuration-language/sensitive-variables#sensitive-values-in-state) to protect your sensitive data.",
				Optional:  true,
				Sensitive: true,
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the SMB storage repository.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The test ID of the SMB storage repository.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Set the parameter of the resource, pass value from provider
func (r *smbResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *smbResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data smbResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating SMB SR...")
	params, err := getSMBCreateParams(r.session, data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get SR create params",
			err.Error(),
		)
		return
	}
	srRef, err := createSRResource(r.session, params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create SR",
			err.Error(),
		)
		return
	}
	srRecord, _, err := getSRRecordAndPBDRecord(r.session, srRef)
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
	err = updateSMBResourceModelComputed(srRecord, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update the computed fields of SMBResourceModel",
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
	tflog.Debug(ctx, "SMB SR created")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read data from State, retrieve the resource's information, update to State
// terraform import
func (r *smbResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data smbResourceModel
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
	err = updateSMBResourceModel(srRecord, pbdRecord, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update the fields of SMBResourceModel",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *smbResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state smbResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Checking if configuration changes are allowed
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := smbResourceModelUpdateCheck(plan, state)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error update xenserver_sr_smb configuration",
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
	err = smbResourceModelUpdate(r.session, srRef, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update SMB SR resource",
			err.Error(),
		)
		return
	}
	srRecord, _, err := getSRRecordAndPBDRecord(r.session, srRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get SR or PBDrecord",
			err.Error(),
		)
		return
	}
	err = updateSMBResourceModelComputed(srRecord, &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update the computed fields of SMBResourceModel",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *smbResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data smbResourceModel
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
			"Unable to delete SMB SR",
			err.Error(),
		)
		return
	}
}

func (r *smbResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}
