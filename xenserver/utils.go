package xenserver

import (
	"context"
	"errors"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"xenapi"
)

func GetFirstTemplate(session *xenapi.Session, templateName string) (xenapi.VMRef, error) {
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
	return "", errors.New("no VM template found")
}

// Get VMResourceModel OtherConfig base on data
func GetVMOtherConfig(ctx context.Context, data VMResourceModel) (map[string]string, error) {
	otherConfig := make(map[string]string)
	if !data.OtherConfig.IsNull() {
		diags := data.OtherConfig.ElementsAs(ctx, &otherConfig, false)
		if diags.HasError() {
			return nil, errors.New("error accessing vm other_config")
		}
	}
	otherConfig["template_name"] = data.TemplateName.ValueString()
	return otherConfig, nil
}

// Update VMResourceModel base on new vmRecord, except uuid
func UpdateVMResourceModel(ctx context.Context, vmRecord xenapi.VMRecord, data *VMResourceModel) error {
	data.NameLabel = types.StringValue(vmRecord.NameLabel)
	data.TemplateName = types.StringValue(vmRecord.OtherConfig["template_name"])
	var diags diag.Diagnostics
	delete(vmRecord.OtherConfig, "template_name")
	data.OtherConfig, diags = types.MapValueFrom(ctx, types.StringType, vmRecord.OtherConfig)
	if diags.HasError() {
		return errors.New("error update data for vm other_config")
	}
	err := UpdateVMResourceModelComputed(ctx, vmRecord, data)
	if err != nil {
		return err
	}
	return nil
}

// Update VMResourceModel computed field base on new vmRecord, except uuid
func UpdateVMResourceModelComputed(ctx context.Context, vmRecord xenapi.VMRecord, data *VMResourceModel) error {
	var diags diag.Diagnostics
	data.Snapshots, diags = types.ListValueFrom(ctx, types.StringType, vmRecord.Snapshots)
	if diags.HasError() {
		return errors.New("error update data for vm snaphots")
	}
	return nil
}
