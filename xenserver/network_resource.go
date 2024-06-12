package xenserver

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"xenapi"
)

// networkResource defines the resource implementation.
type networkResource struct {
	session *xenapi.Session
}

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &networkResource{}
	_ resource.ResourceWithConfigure   = &networkResource{}
	_ resource.ResourceWithImportState = &networkResource{}
)

// This is a helper function to simplify the provider implementation.
func NewNetworkResource() resource.Resource {
	return &networkResource{}
}

func (r *networkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network"
}

func NetworkSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			MarkdownDescription: "The UUID of the virtual network on xenserver",
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		// required
		"name_label": schema.StringAttribute{
			MarkdownDescription: "The name of the virtual network",
			Required:            true,
		},
		"name_description": schema.StringAttribute{
			MarkdownDescription: "The description of the virtual network, default to be empty string",
			Optional:            true,
			Computed:            true, // Required to use Default
			Default:             stringdefault.StaticString(""),
		},
		"mtu": schema.Int64Attribute{
			MarkdownDescription: "MTU in octets, default to be 1500",
			Optional:            true,
			Computed:            true, // Required to use Default
			Default:             int64default.StaticInt64(1500),
		},
		"managed": schema.BoolAttribute{
			MarkdownDescription: "True if the bridge is managed by xapi, default to be true",
			Optional:            true,
			Computed:            true, // Required to use Default
			Default:             booldefault.StaticBool(true),
		},
		"other_config": schema.MapAttribute{
			MarkdownDescription: "The additional configuration of the virtual network",
			Optional:            true,
			Computed:            true, // Required to use Default
			ElementType:         types.StringType,
			Default:             mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
		},
	}
}

func (r *networkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Network Resource",
		Attributes:          NetworkSchema(),
	}
}

func (r *networkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *networkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data networkResourceModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// create new resource
	otherConfig := make(map[string]string, len(data.OtherConfig.Elements()))
	diags := data.OtherConfig.ElementsAs(ctx, &otherConfig, false)
	if diags.HasError() {
		resp.Diagnostics.AddError(
			"Unable to get other_config in plan data",
			"Unable to get other_config in plan data",
		)
		return
	}

	networkRecord := xenapi.NetworkRecord{
		NameLabel:       data.NameLabel.ValueString(),
		NameDescription: data.NameDescription.ValueString(),
		MTU:             int(data.MTU.ValueInt64()),
		Managed:         data.Managed.ValueBool(),
		OtherConfig:     otherConfig,
	}

	networkRef, err := xenapi.Network.Create(r.session, networkRecord)
	if err != nil {
		// failed to create network
		resp.Diagnostics.AddError(
			"Unable to create network",
			err.Error(),
		)
		return
	}

	// Overwrite data with refreshed resource state
	record, err := xenapi.Network.GetRecord(r.session, networkRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get network record",
			err.Error(),
		)
		return
	}

	err = updateNetworkResourceModelComputed(ctx, record, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update the computed fields of NetworkResourceModel",
			err.Error(),
		)
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *networkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data networkResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Overwrite data with refreshed resource state
	networkRef, err := xenapi.Network.GetByUUID(r.session, data.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get network ref",
			err.Error(),
		)
		return
	}

	networkRecord, err := xenapi.Network.GetRecord(r.session, networkRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get network record",
			err.Error(),
		)
		return
	}

	err = updateNetworkResourceModel(ctx, networkRecord, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update the fields of NetworkResourceModel",
			err.Error(),
		)
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *networkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var dataPlan networkResourceModel
	// Read Terraform plan dataPlan into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &dataPlan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Checking if configuration changes are allowed
	var dataState networkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &dataState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if dataPlan.Managed != dataState.Managed {
		resp.Diagnostics.AddError(
			"Error updating managed field of network resource",
			"Managed field is immutable",
		)
		return
	}

	// Get existing network record
	networkRef, err := xenapi.Network.GetByUUID(r.session, dataPlan.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get network ref",
			err.Error(),
		)
		return
	}

	// Update existing network resource with new plan
	err = updateNetworkFields(ctx, r.session, networkRef, dataPlan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update network fields",
			err.Error(),
		)
		return
	}

	// Overwrite dataPlan with refreshed resource state
	record, err := xenapi.Network.GetRecord(r.session, networkRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get network record",
			err.Error(),
		)
		return
	}

	err = updateNetworkResourceModelComputed(ctx, record, &dataPlan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update the computed fields of network resource model",
			err.Error(),
		)
		return
	}

	// Save updated dataPlan into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &dataPlan)...)
}

func (r *networkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data networkResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// delete resource
	networkRef, err := xenapi.Network.GetByUUID(r.session, data.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get network ref",
			err.Error(),
		)
		return
	}

	err = xenapi.Network.Destroy(r.session, networkRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to destroy network",
			err.Error(),
		)
		return
	}
}

func (r *networkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
