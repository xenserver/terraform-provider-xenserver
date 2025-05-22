package xenserver

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
	"xenapi"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type importRawVdiResourceModel struct {
	RawVdiPath types.String `tfsdk:"raw_vdi_path"`
	ID         types.String `tfsdk:"id"`
	UUID       types.String `tfsdk:"uuid"`
}

func importRawVdiSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"raw_vdi_path": schema.StringAttribute{
			Description: "The path to the raw VDI file.",
			Required:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
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

func (r *importRawVdiResource) importRawVdiTask(ctx context.Context, vdiRef xenapi.VDIRef, filePath string, fileSize int64) error {
	// Create the import task
	vdiUUID, err := xenapi.VDI.GetUUID(r.session, vdiRef)
	if err != nil {
		return errors.New("failed to get VDI UUID: " + err.Error())
	}
	taskName := "import " + vdiUUID
	importTask, err := xenapi.Task.Create(r.session, taskName, "")
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
	fullURL := fmt.Sprintf("%s/import_raw_vdi?session_id=%s&vdi=%s&task_id=%s", r.coordinatorConf.Host, r.sessionRef, vdiRef, importTask)
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
	importStatus, err := xenapi.Task.GetStatus(r.session, importTask)
	if err != nil {
		return errors.New("unable to get task status: " + err.Error())
	}

	// Wait for task completion - remove the unnecessary Sleep
	timeout := 60 * 60 // 10 minutes in seconds
	for importStatus == "pending" {
		time.Sleep(5 * time.Second) // Check every second
		importStatus, err = xenapi.Task.GetStatus(r.session, importTask)
		if err != nil {
			return errors.New("unable to get task status: " + err.Error())
		}

		progress, err := xenapi.Task.GetProgress(r.session, importTask)
		if err != nil {
			return errors.New("unable to get task progress: " + err.Error())
		}
		tflog.Debug(ctx, fmt.Sprintf("Task progress: %.2f", progress))

		timeout--
		if timeout <= 0 {
			if err := xenapi.Task.Cancel(r.session, importTask); err != nil {
				tflog.Warn(ctx, "Failed to cancel task: "+err.Error())
			}
			return errors.New("import task timed out: the server took too long to process the import")
		}
	}

	// Check task success
	if importStatus != "success" {
		errorInfo, err := xenapi.Task.GetErrorInfo(r.session, importTask)
		if err != nil {
			return errors.New("task failed but couldn't get error info: " + err.Error())
		}
		return fmt.Errorf("import task failed: %s", errorInfo)
	}

	// Cleanup task
	if err := xenapi.Task.Destroy(r.session, importTask); err != nil {
		tflog.Warn(ctx, "Failed to destroy task: "+err.Error())
	}

	tflog.Debug(ctx, "VDI import completed successfully")
	return nil
}
