package xenserver

import (
	"context"
	"errors"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"xenapi"
)

// vmResourceModel describes the resource data model.
type vmResourceModel struct {
	NameLabel    types.String `tfsdk:"name_label"`
	TemplateName types.String `tfsdk:"template_name"`
	OtherConfig  types.Map    `tfsdk:"other_config"`
	Snapshots    types.List   `tfsdk:"snapshots"`
	UUID         types.String `tfsdk:"id"`
}

func getFirstTemplate(session *xenapi.Session, templateName string) (xenapi.VMRef, error) {
	records, err := xenapi.VM.GetAllRecords(session)
	if err != nil {
		return "", errors.New(err.Error())
	}
	// Get the first VM template ref
	for ref, record := range records {
		if record.IsATemplate && strings.Contains(record.NameLabel, templateName) {
			return ref, nil
		}
	}
	return "", errors.New("unable to find VM template ref")
}

// Get vmResourceModel OtherConfig base on data
func getVMOtherConfig(ctx context.Context, data vmResourceModel) (map[string]string, error) {
	otherConfig := make(map[string]string)
	if !data.OtherConfig.IsNull() {
		diags := data.OtherConfig.ElementsAs(ctx, &otherConfig, false)
		if diags.HasError() {
			return nil, errors.New("unable to read VM other config")
		}
	}
	otherConfig["template_name"] = data.TemplateName.ValueString()
	return otherConfig, nil
}

// Update vmResourceModel base on new vmRecord, except uuid
func updateVMResourceModel(ctx context.Context, vmRecord xenapi.VMRecord, data *vmResourceModel) error {
	data.NameLabel = types.StringValue(vmRecord.NameLabel)
	data.TemplateName = types.StringValue(vmRecord.OtherConfig["template_name"])
	var diags diag.Diagnostics
	delete(vmRecord.OtherConfig, "template_name")
	data.OtherConfig, diags = types.MapValueFrom(ctx, types.StringType, vmRecord.OtherConfig)
	if diags.HasError() {
		return errors.New("unable to read VM other config")
	}
	err := updateVMResourceModelComputed(ctx, vmRecord, data)
	if err != nil {
		return err
	}
	return nil
}

// Update vmResourceModel computed field base on new vmRecord, except uuid
func updateVMResourceModelComputed(ctx context.Context, vmRecord xenapi.VMRecord, data *vmResourceModel) error {
	var diags diag.Diagnostics
	data.Snapshots, diags = types.ListValueFrom(ctx, types.StringType, vmRecord.Snapshots)
	if diags.HasError() {
		return errors.New("unable to read VM snapshots")
	}
	return nil
}
