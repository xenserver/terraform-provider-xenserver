package xenserver

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"xenapi"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource              = &importRawVdiResource{}
	_ resource.ResourceWithConfigure = &importRawVdiResource{}
)

func NewImportRawVdiResource() resource.Resource {
	return &importRawVdiResource{}
}

// importRawVdiResource defines the resource implementation.
type importRawVdiResource struct {
	coordinatorConf coordinatorConf
	session         *xenapi.Session
	sessionRef      xenapi.SessionRef
}

func (r *importRawVdiResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_import_raw_vdi"
}

func (r *importRawVdiResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "import Raw/VHD vdi resource.",
		Attributes:          importRawVdiSchema(),
	}
}

// Set the parameter of the resource, pass value from provider
func (r *importRawVdiResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	providerData, ok := req.ProviderData.(*xsProvider)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *xenserver.xsProvider, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.session = providerData.session
	r.sessionRef = providerData.sessionRef
	r.coordinatorConf = providerData.coordinatorConf
}

func (r *importRawVdiResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data importRawVdiResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.RawVdiPath.IsUnknown() || data.RawVdiPath.IsNull() {
		resp.Diagnostics.AddError(
			"Missing Required Attribute",
			"The `raw_vdi_path` attribute is required, but was not set.",
		)
		return
	}

	tflog.Debug(ctx, "Creating VDI with file path: "+data.RawVdiPath.ValueString())
	fileInfo, err := os.Stat(data.RawVdiPath.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get file",
			fmt.Sprintf("Failed to get file: %s", err),
		)
		return
	}

	if fileInfo.IsDir() {
		resp.Diagnostics.AddError(
			"Invalid file path",
			"The provided path is a directory, not a file: "+data.RawVdiPath.ValueString(),
		)
		return
	}

	if fileInfo.Size() == 0 {
		resp.Diagnostics.AddError(
			"Empty file",
			"The provided file is empty: "+data.RawVdiPath.ValueString(),
		)
		return
	}

	defaultSr, err := getDefaultSR(r.session)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get pool",
			fmt.Sprintf("Error getting pool: %s", err),
		)
		return
	}

	vdiRef, err := createVDI(r.session, fileInfo.Name(), int(fileInfo.Size()), defaultSr)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create VDI",
			"Error creating VDI: "+err.Error(),
		)
		return
	}

	vdiUUID, err := xenapi.VDI.GetUUID(r.session, vdiRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get VDI UUID",
			fmt.Sprintf("Error getting VDI UUID: %s", err),
		)
		return
	}

	err = r.importRawVdiTask(ctx, vdiRef, data.RawVdiPath.ValueString(), fileInfo.Size())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to import VDI",
			fmt.Sprintf("Error importing VDI: %s", err),
		)

		err = removeVDI(r.session, vdiRef)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to remove VDI",
				fmt.Sprintf("Error removing VDI: %s", err),
			)
			tflog.Debug(ctx, "Failed to destroy VDI for UUID: "+vdiUUID)
			return
		}

		return
	}

	data.ID = types.StringValue(vdiUUID)
	data.UUID = types.StringValue(vdiUUID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *importRawVdiResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data importRawVdiResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Simply pass through the state since we don't need to track any changes
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *importRawVdiResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"This resource only supports importing VDIs. To modify, you need to create a new resource.",
	)
}

func (r *importRawVdiResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data importRawVdiResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vdiRef, err := xenapi.VDI.GetByUUID(r.session, data.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get VDI ref",
			fmt.Sprintf("Error getting VDI ref: %s", err),
		)
		return
	}

	err = removeVDI(r.session, vdiRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to remove VDI",
			fmt.Sprintf("Error removing VDI: %s", err),
		)
		tflog.Debug(ctx, "Failed to destroy VDI for UUID: "+data.UUID.ValueString())
		return
	}
}
