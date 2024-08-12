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

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &poolResource{}
	_ resource.ResourceWithConfigure   = &poolResource{}
	_ resource.ResourceWithImportState = &poolResource{}
)

func NewPoolResource() resource.Resource {
	return &poolResource{}
}

// poolResource defines the resource implementation.
type poolResource struct {
	session        *xenapi.Session
	providerConfig *providerModel
}

func (r *poolResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pool"
}

func (r *poolResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Provides a pool resource.",
		Attributes:          PoolSchema(),
	}
}

// Set the parameter of the resource, pass value from provider
func (r *poolResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *poolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan poolResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating pool...")
	poolParams, err := getPoolParams(plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get pool create params",
			err.Error(),
		)
		return
	}

	poolRef, err := getPoolRef(r.session)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get pool ref",
			err.Error(),
		)
		return
	}

	err = setPool(r.session, poolRef, poolParams)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to set pool",
			err.Error(),
		)

		err = cleanupPoolResource(r.session, poolRef)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to cleanup pool resource",
				err.Error(),
			)
		}

		return
	}

	err = poolJoin(r.providerConfig, poolParams)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to join pool",
			err.Error(),
		)
		return
	}

	poolRecord, err := xenapi.Pool.GetRecord(r.session, poolRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get pool record",
			err.Error(),
		)
		return
	}

	err = updatePoolResourceModelComputed(r.session, poolRecord, &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update the computed fields of PoolResourceModel",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *poolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state poolResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	poolRef, err := xenapi.Pool.GetByUUID(r.session, state.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get pool ref",
			err.Error(),
		)
		return
	}

	poolRecord, err := xenapi.Pool.GetRecord(r.session, poolRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get pool record",
			err.Error(),
		)
		return
	}

	err = updatePoolResourceModel(r.session, poolRecord, &state)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update the computed fields of PoolResourceModel",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *poolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state poolResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	poolRef, err := xenapi.Pool.GetByUUID(r.session, state.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get pool ref",
			err.Error(),
		)
		return
	}

	err = poolResourceModelUpdate(r.session, poolRef, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update pool resource model",
			err.Error(),
		)
		return
	}

	poolRecord, err := xenapi.Pool.GetRecord(r.session, poolRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get pool record",
			err.Error(),
		)
		return
	}

	err = updatePoolResourceModelComputed(r.session, poolRecord, &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update the computed fields of PoolResourceModel",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *poolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state poolResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	poolRef, err := xenapi.Pool.GetByUUID(r.session, state.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get pool ref",
			err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "Deleting pool...")
	err = cleanupPoolResource(r.session, poolRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to cleanup pool resource",
			err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "Pool deleted")
}

func (r *poolResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}
