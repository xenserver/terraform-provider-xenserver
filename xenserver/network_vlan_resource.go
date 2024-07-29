package xenserver

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"xenapi"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &vlanResource{}
	_ resource.ResourceWithConfigure   = &vlanResource{}
	_ resource.ResourceWithImportState = &vlanResource{}
)

func NewVlanResource() resource.Resource {
	return &vlanResource{}
}

// vlanResource defines the resource implementation.
type vlanResource struct {
	session *xenapi.Session
}

func (r *vlanResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_vlan"
}

func (r *vlanResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Provides an external network resource. A network that passes traffic over one of your VLANs.",
		Attributes: map[string]schema.Attribute{
			"name_label": schema.StringAttribute{
				MarkdownDescription: "The name of the network.",
				Required:            true,
			},
			"name_description": schema.StringAttribute{
				MarkdownDescription: "The description of the network, default to be `\"\"`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"mtu": schema.Int32Attribute{
				MarkdownDescription: "The MTU of the network, default to be `1500`. The minimum value this attribute can be set is `0`.",
				Optional:            true,
				Computed:            true,
				Default:             int32default.StaticInt32(1500),
				Validators: []validator.Int32{
					int32validator.AtLeast(0),
				},
			},
			"managed": schema.BoolAttribute{
				MarkdownDescription: "True if the bridge is managed by [XAPI](https://github.com/xapi-project/xen-api), default to be `true`." +
					"\n\n-> **Note:** `managed` is not allowed to be updated.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"other_config": schema.MapAttribute{
				MarkdownDescription: "The additional configuration of the network, default to be `{}`.",
				Optional:            true,
				Computed:            true,
				Default:             mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
				ElementType:         types.StringType,
			},
			"vlan_tag": schema.Int32Attribute{
				MarkdownDescription: "The VLAN tag of the network." +
					"\n\n-> **Note:** `vlan_tag` is not allowed to be updated.",
				Required: true,
			},
			"nic": schema.StringAttribute{
				MarkdownDescription: "The NIC used by the network, for example, `\"NIC 0\"`, `\"Bond 0+1\"`, `\"NIC-SR-IOV 0\"`." + "<br />" +
					"The NIC on target XenServer environment can be found by the `xenserver_nic` data-source." +
					"\n\n-> **Note:** `nic` is not allowed to be updated.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^NIC|^Bond|^NIC-SR-IOV`),
						`must start with "NIC", "Bond" or "NIC-SR-IOV", eg. "NIC 0", "Bond 0+1", "NIC-SR-IOV 0"`,
					),
				},
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the network.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The test ID of the network.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *vlanResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *vlanResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data vlanResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating Network...")
	networkRecord, err := getNetworkCreateParams(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get network create params",
			err.Error(),
		)
		return
	}
	networkRef, err := xenapi.Network.Create(r.session, networkRecord)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create network",
			err.Error(),
		)
		return
	}
	networkRecord, err = xenapi.Network.GetRecord(r.session, networkRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get network record",
			err.Error(),
		)
		err = cleanupVlanResource(r.session, networkRef)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error cleaning up network resource",
				err.Error(),
			)
		}
		return
	}
	err = updateVlanResourceModelComputed(ctx, networkRecord, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update the computed fields of vlanResourceModel",
			err.Error(),
		)
		err = cleanupVlanResource(r.session, networkRef)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error cleaning up network resource",
				err.Error(),
			)
		}
		return
	}

	tflog.Debug(ctx, "Creating Vlan...")
	params, err := getVlanCreateParams(r.session, data, networkRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get vlan create params",
			err.Error(),
		)
		err = cleanupVlanResource(r.session, networkRef)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error cleaning up network resource",
				err.Error(),
			)
		}
		return
	}
	_, err = xenapi.Pool.CreateVLANFromPIF(r.session, params.PifRef, params.NetworkRef, params.Tag)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create vlan",
			err.Error(),
		)
		err = cleanupVlanResource(r.session, networkRef)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error cleaning up network resource",
				err.Error(),
			)
		}
		return
	}

	tflog.Debug(ctx, "External Network created")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vlanResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data vlanResourceModel
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
	err = updateVlanResourceModel(ctx, r.session, networkRecord, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update the fields of vlanResourceModel",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vlanResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state vlanResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Checking if configuration changes are allowed
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := vlanResourceModelUpdateCheck(plan, state)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error update xenserver_network_vlan configuration",
			err.Error(),
		)
		return
	}

	// Update the resource with new configuration
	networkRef, err := xenapi.Network.GetByUUID(r.session, plan.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get network ref",
			err.Error(),
		)
		return
	}
	err = vlanResourceModelUpdate(ctx, r.session, networkRef, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update network_vlan resource",
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
	err = updateVlanResourceModelComputed(ctx, networkRecord, &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update the computed fields of vlanResourceModel",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *vlanResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data vlanResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	networkRef, err := xenapi.Network.GetByUUID(r.session, data.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get network ref",
			err.Error(),
		)
		return
	}
	err = cleanupVlanResource(r.session, networkRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete network resource",
			err.Error(),
		)
		return
	}
}

func (r *vlanResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}
