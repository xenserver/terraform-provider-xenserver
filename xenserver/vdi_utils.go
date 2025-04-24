package xenserver

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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

type vdiResourceModel struct {
	NameLabel       types.String `tfsdk:"name_label"`
	NameDescription types.String `tfsdk:"name_description"`
	SR              types.String `tfsdk:"sr_uuid"`
	VirtualSize     types.Int64  `tfsdk:"virtual_size"`
	RawVdiPath      types.String `tfsdk:"raw_vdi_path"`
	Type            types.String `tfsdk:"type"`
	Sharable        types.Bool   `tfsdk:"sharable"`
	ReadOnly        types.Bool   `tfsdk:"read_only"`
	OtherConfig     types.Map    `tfsdk:"other_config"`
	UUID            types.String `tfsdk:"uuid"`
	ID              types.String `tfsdk:"id"`
}

var vdiResourceModelAttrTypes = map[string]attr.Type{
	"name_label":       types.StringType,
	"name_description": types.StringType,
	"sr_uuid":          types.StringType,
	"virtual_size":     types.Int64Type,
	"raw_vdi_path":     types.StringType,
	"type":             types.StringType,
	"sharable":         types.BoolType,
	"read_only":        types.BoolType,
	"other_config":     types.MapType{ElemType: types.StringType},
	"uuid":             types.StringType,
	"id":               types.StringType,
}

func vdiSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name_label": schema.StringAttribute{
			MarkdownDescription: "The name of the virtual disk image.",
			Required:            true,
		},
		"name_description": schema.StringAttribute{
			MarkdownDescription: "The description of the virtual disk image, default to be `\"\"`.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"sr_uuid": schema.StringAttribute{
			MarkdownDescription: "The UUID of the storage repository used." +
				"\n\n-> **Note:** `sr_uuid` is not allowed to be updated.",
			Required: true,
		},
		"virtual_size": schema.Int64Attribute{
			MarkdownDescription: "The size of virtual disk image (in bytes)." +
				"\n\n-> **Note:**\n\n" +
				" 1. `virtual_size` is required if `raw_vdi_path` is not set." +
				" 2. `virtual_size` is not allowed to be updated.",
			Optional: true,
			Computed: true,
		},
		"raw_vdi_path": schema.StringAttribute{
			Description: "The file path to the raw disk image (VDI), compatible with \"Raw\", \"VHD\" formats." +
				"\n\n-> **Note:**\n\n" +
				" 1. `raw_vdi_path` is required if `virtual_size` is not set." +
				" 2. `raw_vdi_path` is not allowed to be updated." +
				" 3. If `raw_vdi_path` is set, `virtual_size` will be ignored." +
				" 4. If `raw_vdi_path` is set, `type` will be `user`, `sharable` and `read_only` will be `false`.",
			Optional: true,
		},
		"type": schema.StringAttribute{
			MarkdownDescription: "The type of the virtual disk image, default to be `\"user\"`." +
				"\n\n-> **Note:** `type` is not allowed to be updated.",
			Optional: true,
			Computed: true,
			Default:  stringdefault.StaticString("user"),
		},
		"sharable": schema.BoolAttribute{
			MarkdownDescription: "True if this disk may be shared, default to be `false`." +
				"\n\n-> **Note:** `sharable` is not allowed to be updated.",
			Optional: true,
			Computed: true,
			Default:  booldefault.StaticBool(false),
		},
		"read_only": schema.BoolAttribute{
			MarkdownDescription: "True if this SR is (capable of being) shared between multiple hosts, default to be `false`." +
				"\n\n-> **Note:** `read_only` is not allowed to be updated.",
			Optional: true,
			Computed: true,
			Default:  booldefault.StaticBool(false),
		},
		"other_config": schema.MapAttribute{
			MarkdownDescription: "The additional configuration of the virtual disk image, default to be `{}`.",
			Optional:            true,
			Computed:            true,
			Default:             mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
			ElementType:         types.StringType,
		},
		"uuid": schema.StringAttribute{
			MarkdownDescription: "The UUID of the virtual disk image.",
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"id": schema.StringAttribute{
			MarkdownDescription: "The test ID of the virtual disk image.",
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
	}
}

func getVDICreateParams(ctx context.Context, session *xenapi.Session, data vdiResourceModel) (xenapi.VDIRecord, error) {
	var record xenapi.VDIRecord
	record.NameLabel = data.NameLabel.ValueString()
	record.NameDescription = data.NameDescription.ValueString()
	srRef, err := xenapi.SR.GetByUUID(session, data.SR.ValueString())
	if err != nil {
		return record, errors.New(err.Error())
	}
	record.SR = srRef

	if data.RawVdiPath.IsNull() {
		record.Type = xenapi.VdiType(data.Type.ValueString())
		record.VirtualSize = int(data.VirtualSize.ValueInt64())
		record.Sharable = data.Sharable.ValueBool()
		record.ReadOnly = data.ReadOnly.ValueBool()
	} else {
		record.Type = xenapi.VdiType("user")
		record.Sharable = false
		record.ReadOnly = false
	}

	diags := data.OtherConfig.ElementsAs(ctx, &record.OtherConfig, false)
	if diags.HasError() {
		return record, errors.New("unable to access VDI other config")
	}

	return record, nil
}

func updateVDIResourceModel(ctx context.Context, session *xenapi.Session, record xenapi.VDIRecord, data *vdiResourceModel) error {
	data.NameLabel = types.StringValue(record.NameLabel)
	srUUID, err := getUUIDFromSRRef(session, record.SR)
	if err != nil {
		return err
	}
	data.SR = types.StringValue(srUUID)
	return updateVDIResourceModelComputed(ctx, record, data)
}

func updateVDIResourceModelComputed(ctx context.Context, record xenapi.VDIRecord, data *vdiResourceModel) error {
	data.UUID = types.StringValue(record.UUID)
	data.ID = types.StringValue(record.UUID)
	data.NameDescription = types.StringValue(record.NameDescription)
	data.Type = types.StringValue(string(record.Type))
	data.Sharable = types.BoolValue(record.Sharable)
	data.ReadOnly = types.BoolValue(record.ReadOnly)
	data.VirtualSize = types.Int64Value(int64(record.VirtualSize))
	var diags diag.Diagnostics
	// Remove key content_id that is created when importing a VDI in record.OtherConfig
	// delete() here to avoid TF state inconsistent.
	delete(record.OtherConfig, "content_id")
	data.OtherConfig, diags = types.MapValueFrom(ctx, types.StringType, record.OtherConfig)
	if diags.HasError() {
		return errors.New("unable to access VDI other config")
	}

	return nil
}

func vdiResourceModelUpdateCheck(data vdiResourceModel, dataState vdiResourceModel) error {
	if data.SR != dataState.SR {
		return errors.New(`"sr_uuid" doesn't expected to be updated`)
	}
	if data.VirtualSize != dataState.VirtualSize {
		return errors.New(`"virtual_size" doesn't expected to be updated`)
	}
	if data.Type != dataState.Type {
		return errors.New(`"type" doesn't expected to be updated`)
	}
	if data.Sharable != dataState.Sharable {
		return errors.New(`"sharable" doesn't expected to be updated`)
	}
	if data.ReadOnly != dataState.ReadOnly {
		return errors.New(`"read_only" doesn't expected to be updated`)
	}
	return nil
}

func vdiResourceModelUpdate(ctx context.Context, session *xenapi.Session, ref xenapi.VDIRef, data vdiResourceModel) error {
	err := xenapi.VDI.SetNameLabel(session, ref, data.NameLabel.ValueString())
	if err != nil {
		return errors.New(err.Error())
	}
	err = xenapi.VDI.SetNameDescription(session, ref, data.NameDescription.ValueString())
	if err != nil {
		return errors.New(err.Error())
	}
	otherConfig := make(map[string]string)
	diags := data.OtherConfig.ElementsAs(ctx, &otherConfig, false)
	if diags.HasError() {
		return errors.New("unable to access VDI other config")
	}
	err = xenapi.VDI.SetOtherConfig(session, ref, otherConfig)
	if err != nil {
		return errors.New(err.Error())
	}
	return nil
}

func cleanupVDIResource(ctx context.Context, session *xenapi.Session, ref xenapi.VDIRef) error {
	err := xenapi.VDI.Destroy(session, ref)
	if err != nil {
		// if error message VDI_IN_USE, retry 10 times
		if strings.Contains(err.Error(), "VDI_IN_USE") {
			for range 10 {
				tflog.Warn(ctx, "VDI is in use, retrying to destroy VDI...")
				time.Sleep(5 * time.Second)
				err = xenapi.VDI.Destroy(session, ref)
				if err == nil {
					return nil
				}
			}
		}
		return errors.New("failed to destroy VDI: " + err.Error())
	}
	return nil
}

// VHDFooter represents the footer structure of a VHD file
// According to "Virtual Hard Disk Format Spec_10_18_06.doc"
type VHDFooter struct {
	Cookie             [8]byte   // "conectix" string
	Features           uint32    // Features bit field
	FileFormatVersion  uint32    // Major/minor version of the format
	DataOffset         uint64    // Offset to the next structure
	TimeStamp          uint32    // Creation time
	CreatorApplication [4]byte   // Creator application
	CreatorVersion     uint32    // Version of creator application
	CreatorHostOS      uint32    // Creator host OS
	OriginalSize       uint64    // Size of the virtual disk
	CurrentSize        uint64    // Current size of the virtual disk
	DiskGeometry       uint32    // Disk geometry
	DiskType           uint32    // Disk type
	Checksum           uint32    // Checksum
	UniqueID           [16]byte  // Unique ID
	SavedState         uint8     // Saved state
	Reserved           [427]byte // Reserved
}

// IsVHDFile checks if a file is a valid VHD file by examining its footer
// See Virtual Hard Disk Format Spec_10_18_06.doc
func IsVHDFile(filePath string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get the file size
	fileInfo, err := file.Stat()
	if err != nil {
		return false, fmt.Errorf("failed to get file stats: %w", err)
	}

	// The VHD footer is 512 bytes and is located at the end of the file
	const footerSize = 512

	// Check if the file is large enough to have a footer
	if fileInfo.Size() < footerSize {
		return false, nil // Not an error, just not a VHD file
	}

	// Seek to the beginning of the footer (512 bytes from the end)
	_, err = file.Seek(-footerSize, io.SeekEnd)
	if err != nil {
		return false, fmt.Errorf("failed to seek to footer: %w", err)
	}

	// Read just the cookie to verify
	cookie := make([]byte, 8)
	if _, err := io.ReadFull(file, cookie); err != nil {
		return false, fmt.Errorf("failed to read cookie: %w", err)
	}

	// Check if the cookie matches "conectix"
	expectedCookie := []byte("conectix")
	return string(cookie) == string(expectedCookie), nil
}

// Retrieves the original size of a VHD file
func GetVHDOriginalSize(filePath string) (uint64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open VHD file: %w", err)
	}
	defer file.Close()

	const footerSize = 512
	_, err = file.Seek(-footerSize, io.SeekEnd)
	if err != nil {
		return 0, fmt.Errorf("failed to seek to VHD footer: %w", err)
	}

	// Read the footer
	footer := VHDFooter{}
	err = binary.Read(file, binary.BigEndian, &footer)
	if err != nil {
		return 0, fmt.Errorf("failed to read VHD footer: %w", err)
	}

	return footer.OriginalSize, nil
}

func importRawVdiTask(ctx context.Context, session *xenapi.Session, coordinatorConf *coordinatorConf, sessionRef xenapi.SessionRef, vdiRef xenapi.VDIRef, filePath string, fileSize int64, format string) error {
	// Create the import task
	vdiUUID, err := xenapi.VDI.GetUUID(session, vdiRef)
	if err != nil {
		return errors.New("failed to get VDI UUID: " + err.Error())
	}
	taskName := "HTTP_actions.put_import_raw_vdi"
	taskDresciption := "import disk " + filePath + " to VDI " + vdiUUID
	importTask, err := xenapi.Task.Create(session, taskName, taskDresciption)
	if err != nil {
		return errors.New("failed to create import task: " + err.Error())
	}

	// Open file for streaming
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Configure HTTP client with appropriate timeouts and TLS settings
	// #nosec G402 - InsecureSkipVerify is required for self-signed certificates
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // Equivalent to curl's --insecure flag
		},
		// Set other transport options for performance
		MaxIdleConns:          10,
		MaxIdleConnsPerHost:   5,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Minute, // Long timeout for large files
	}

	// Create a new PUT request
	fullURL := fmt.Sprintf("%s/import_raw_vdi?session_id=%s&vdi=%s&task_id=%s&format=%s", coordinatorConf.Host, sessionRef, vdiRef, importTask, format)
	tflog.Debug(ctx, "Creating HTTP request to upload VDI to: "+fullURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, fullURL, file)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.ContentLength = fileSize // Important for large files
	req.Header.Set("Content-Type", "application/octet-stream")
	tflog.Debug(ctx, fmt.Sprintf("Uploading file %s (%d bytes) to %s", filePath, fileSize, fullURL))

	// Send the request
	startTime := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close() // Important to prevent memory leaks

	// Log upload statistics
	uploadDuration := time.Since(startTime)
	uploadSpeed := float64(fileSize) / uploadDuration.Seconds() / 1024 / 1024 // MB/s
	tflog.Debug(ctx, fmt.Sprintf("Upload completed in %v (%.2f MB/s)", uploadDuration, uploadSpeed))

	// Check response status
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to upload file: %s, status code: %d, response: %s", filePath, resp.StatusCode, string(respBody))
	}

	// Continue with your existing code for monitoring the task...
	// Monitor task status
	importStatus, err := xenapi.Task.GetStatus(session, importTask)
	if err != nil {
		return errors.New("unable to get task status: " + err.Error())
	}

	// Wait for task completion - remove the unnecessary Sleep
	timeout := 60 * 60
	for importStatus == "pending" {
		time.Sleep(5 * time.Second) // Check every second
		importStatus, err = xenapi.Task.GetStatus(session, importTask)
		if err != nil {
			return errors.New("unable to get task status: " + err.Error())
		}

		progress, err := xenapi.Task.GetProgress(session, importTask)
		if err != nil {
			return errors.New("unable to get task progress: " + err.Error())
		}
		tflog.Debug(ctx, fmt.Sprintf("Task progress: %.2f", progress))

		timeout--
		if timeout <= 0 {
			if err := xenapi.Task.Cancel(session, importTask); err != nil {
				tflog.Warn(ctx, "Failed to cancel task: "+err.Error())
			}
			return errors.New("import task timed out: the server took too long to process the import")
		}
	}

	// Check task success
	if importStatus != "success" {
		errorInfo, err := xenapi.Task.GetErrorInfo(session, importTask)
		if err != nil {
			return errors.New("task failed but couldn't get error info: " + err.Error())
		}
		return fmt.Errorf("import task failed: %s", errorInfo)
	}

	// Cleanup task
	if err := xenapi.Task.Destroy(session, importTask); err != nil {
		tflog.Warn(ctx, "Failed to destroy task: "+err.Error())
	}

	tflog.Debug(ctx, "VDI import completed successfully")
	return nil
}
