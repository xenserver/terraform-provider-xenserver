package xenserver

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
	coordinatorConf *coordinatorConf
	session         *xenapi.Session
	sessionRef      xenapi.SessionRef
}

func (r *vdiResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vdi"
}

func (r *vdiResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Provides a virtual disk image resource.",
		Attributes:          vdiSchema(),
	}
}

// Set the parameter of the resource, pass value from provider
func (r *vdiResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.coordinatorConf = &providerData.coordinatorConf
	r.sessionRef = providerData.sessionRef
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

	var fileInfo os.FileInfo
	diskFormat := "raw"
	if !data.RawVdiPath.IsNull() {
		filePath := data.RawVdiPath.ValueString()
		tflog.Debug(ctx, "Creating VDI with file path: "+filePath)
		fileInfo, err = os.Stat(filePath)
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
				"The provided path is a directory, not a file: "+filePath,
			)
			return
		}

		if fileInfo.Size() == 0 {
			resp.Diagnostics.AddError(
				"Empty file",
				"The provided file is empty: "+filePath,
			)
			return
		}

		virtualSize := int(fileInfo.Size())
		// Check if file is a VHD and try to get its original size
		isVHD, err := IsVHDFile(filePath)
		if err != nil {
			tflog.Debug(ctx, fmt.Sprintf("Error checking if file is VHD: %s", err))
		} else if isVHD {
			if vhdSize, vhdErr := GetVHDOriginalSize(filePath); vhdErr == nil {
				tflog.Info(ctx, fmt.Sprintf("Detected VHD file with original size: %d bytes", vhdSize))
				virtualSize = int(vhdSize)
				diskFormat = "vhd"
			} else {
				resp.Diagnostics.AddError(
					"Unable to get VHD original size",
					fmt.Sprintf("Failed to get VHD original size: %s", vhdErr),
				)
				return
			}
		}

		record.VirtualSize = virtualSize
	}

	vdiRef, err := xenapi.VDI.Create(r.session, record)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create VDI",
			err.Error(),
		)
		return
	}

	if !data.RawVdiPath.IsNull() {
		err = importRawVdiTask(ctx, r.session, r.coordinatorConf, r.sessionRef, vdiRef, data.RawVdiPath.ValueString(), fileInfo.Size(), diskFormat)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to import VDI",
				fmt.Sprintf("Error importing VDI: %s", err),
			)

			err = cleanupVDIResource(ctx, r.session, vdiRef)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error cleaning up VDI resource",
					err.Error(),
				)
			}
			return
		}
	}

	vdiRecord, err := xenapi.VDI.GetRecord(r.session, vdiRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get VDI record",
			err.Error(),
		)
		err = cleanupVDIResource(ctx, r.session, vdiRef)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error cleaning up VDI resource",
				err.Error(),
			)
		}
		return
	}

	err = updateVDIResourceModelComputed(ctx, vdiRecord, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update the computed fields of VDIResourceModel",
			err.Error(),
		)
		err = cleanupVDIResource(ctx, r.session, vdiRef)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error cleaning up VDI resource",
				err.Error(),
			)
		}
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
	var plan, state vdiResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Checking if configuration changes are allowed
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := vdiResourceModelUpdateCheck(plan, state)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error update xenserver_vdi configuration",
			err.Error(),
		)
		return
	}

	// Update the resource with new configuration
	vdiRef, err := xenapi.VDI.GetByUUID(r.session, plan.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get VDI ref",
			err.Error(),
		)
		return
	}
	err = vdiResourceModelUpdate(ctx, r.session, vdiRef, plan)
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
	err = updateVDIResourceModelComputed(ctx, vdiRecord, &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update the computed fields of VDIResourceModel",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
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
	err = cleanupVDIResource(ctx, r.session, vdiRef)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete VDI resource",
			err.Error(),
		)
		return
	}
}

func (r *vdiResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}
