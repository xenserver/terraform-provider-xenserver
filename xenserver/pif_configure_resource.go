package xenserver

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"xenapi"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &pifConfigureResource{}
	_ resource.ResourceWithConfigure   = &pifConfigureResource{}
	_ resource.ResourceWithImportState = &pifConfigureResource{}
)

func NewPIFConfigureResource() resource.Resource {
	return &pifConfigureResource{}
}

// pifConfigureResource defines the resource implementation.
type pifConfigureResource struct {
	session *xenapi.Session
}

func (r *pifConfigureResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pif_configure"
}

func (r *pifConfigureResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Provides an PIF configure resource to update the exist PIF parameters.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the PIF.",
				Required:            true,
			},
			"disallow_unplug": schema.BoolAttribute{
				MarkdownDescription: "Set to `true` if you want to prevent this PIF from being unplugged.",
				Optional:            true,
			},
			"interface": schema.SingleNestedAttribute{
				MarkdownDescription: "The IP interface of the PIF. Currently only support IPv4.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"mode": schema.StringAttribute{
						MarkdownDescription: "The protocol define the primary address of this PIF, for example, `\"None\"`, `\"DHCP\"`, `\"Static\"`.",
						Required:            true,
						Validators: []validator.String{
							stringvalidator.OneOf("None", "DHCP", "Static"),
						},
					},
					"ip": schema.StringAttribute{
						MarkdownDescription: "The IP address.",
						Optional:            true,
					},
					"gateway": schema.StringAttribute{
						MarkdownDescription: "The IP gateway.",
						Optional:            true,
					},
					"netmask": schema.StringAttribute{
						MarkdownDescription: "The IP netmask.",
						Optional:            true,
					},
					"dns": schema.StringAttribute{
						MarkdownDescription: "Comma separated list of the IP addresses of the DNS servers to use.",
						Optional:            true,
					},
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The test ID of the PIF.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Set the parameter of the resource, pass value from provider
func (r *pifConfigureResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *pifConfigureResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data pifConfigureResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := pifConfigureResourceModelUpdate(ctx, r.session, data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update PIF configuration",
			err.Error(),
		)
		return
	}

	data.ID = data.UUID
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read data from State, retrieve the resource's information, update to State
// terraform import
func (r *pifConfigureResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data pifConfigureResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = data.UUID
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *pifConfigureResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan pifConfigureResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := pifConfigureResourceModelUpdate(ctx, r.session, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update PIF configuration",
			err.Error(),
		)
		return
	}

	plan.ID = plan.UUID
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pifConfigureResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Don't recover the PIF configuration when destroy resource")
}

func (r *pifConfigureResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}
