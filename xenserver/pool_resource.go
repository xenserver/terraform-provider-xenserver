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
	session         *xenapi.Session
	coordinatorConf *coordinatorConf
}

func (r *poolResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pool"
}

func (r *poolResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "This provides a pool resource." + "\n\n-> **Note:** During the execution of `terraform destroy` for this particular resource, all of the hosts that are part of the pool will be separated and converted into standalone hosts.",
		Attributes:          PoolSchema(),
	}
}

// Set the parameter of the resource, pass value from provider
func (r *poolResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(*xsProvider)
	if !ok {
		resp.Diagnostics.AddError(
			"Failed to get Provider Data in PoolResource",
			fmt.Sprintf("Expected *xenserver.xsProvider, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.session = providerData.session
	r.coordinatorConf = &providerData.coordinatorConf
}

func (r *poolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "---> Create Pool resource")
	var plan poolResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	poolParams := getPoolParams(plan)

	poolRef, err := getPoolRef(r.session)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get pool ref",
			err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "----> Start Pool join")
	err = poolJoin(ctx, r.session, r.coordinatorConf, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to join pool in Create stage",
			err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "----> Start Pool eject")
	err = poolEject(ctx, r.session, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to eject pool in Create stage",
			err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "----> Start Pool setting")
	err = setPool(r.session, poolRef, poolParams)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to set pool in Create stage",
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
			"Unable to update the computed fields of PoolResourceModel in Create stage",
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
			"Unable to update the computed fields of PoolResourceModel in Read stage",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *poolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "---> Update Pool resource")
	var plan, state poolResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	poolParams := getPoolParams(plan)

	poolRef, err := getPoolRef(r.session)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get pool ref",
			err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "----> Start Pool join")
	err = poolJoin(ctx, r.session, r.coordinatorConf, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to join pool in Update stage",
			err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "----> Start Pool eject")
	err = poolEject(ctx, r.session, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to eject pool in Update stage",
			err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "----> Start Pool setting")
	err = setPool(r.session, poolRef, poolParams)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to set pool in Update stage",
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
			"Unable to update the computed fields of PoolResourceModel in Update stage",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *poolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "---> Delete Pool resource")
	var state poolResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	poolRef, err := xenapi.Pool.GetByUUID(r.session, state.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to get pool ref", err.Error())
		return
	}

	tflog.Debug(ctx, "----> Clean pool resource")
	err = cleanupPoolResource(r.session, poolRef)
	if err != nil {
		resp.Diagnostics.AddError("Unable to cleanup pool resource", err.Error())
		return
	}

	tflog.Debug(ctx, "---> Pool deleted")
}

func (r *poolResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}
