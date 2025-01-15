package xenserver

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"xenapi"
)

// pifDataSourceModel describes the data source data model.
type hostDataSourceModel struct {
	NameLabel     types.String     `tfsdk:"name_label"`
	UUID          types.String     `tfsdk:"uuid"`
	Address       types.String     `tfsdk:"address"`
	IsCoordinator types.Bool       `tfsdk:"is_coordinator"`
	DataItems     []hostRecordData `tfsdk:"data_items"`
}

type hostRecordData struct {
	UUID            types.String `tfsdk:"uuid"`
	NameLabel       types.String `tfsdk:"name_label"`
	NameDescription types.String `tfsdk:"name_description"`
	Hostname        types.String `tfsdk:"hostname"`
	Address         types.String `tfsdk:"address"`
	ResidentVMs     types.List   `tfsdk:"resident_vms"`
}

func hostDataSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"uuid": schema.StringAttribute{
			MarkdownDescription: "The UUID of the host.",
			Computed:            true,
		},
		"name_label": schema.StringAttribute{
			MarkdownDescription: "The name of the host.",
			Computed:            true,
		},
		"name_description": schema.StringAttribute{
			MarkdownDescription: "The human-readable description of the host.",
			Computed:            true,
		},
		"hostname": schema.StringAttribute{
			MarkdownDescription: "The hostname of the host.",
			Computed:            true,
		},
		"address": schema.StringAttribute{
			MarkdownDescription: "The address by which this host can be contacted from any other host in the pool.",
			Computed:            true,
		},
		"resident_vms": schema.ListAttribute{
			MarkdownDescription: "The list of VMs(UUID) currently resident on host.",
			Computed:            true,
			ElementType:         types.StringType,
		},
	}
}

func updateHostRecordData(ctx context.Context, session *xenapi.Session, record xenapi.HostRecord, data *hostRecordData) error {
	tflog.Debug(ctx, "Found host data: "+record.NameLabel)
	data.UUID = types.StringValue(record.UUID)
	data.NameLabel = types.StringValue(record.NameLabel)
	data.NameDescription = types.StringValue(record.NameDescription)
	data.Hostname = types.StringValue(record.Hostname)
	data.Address = types.StringValue(record.Address)
	residentVMs, err := getVMUUIDs(session, record.ResidentVMs, record.ControlDomain)
	if err != nil {
		return err
	}
	var diags diag.Diagnostics
	data.ResidentVMs, diags = types.ListValueFrom(ctx, types.StringType, residentVMs)
	if diags.HasError() {
		return errors.New("unable to read Host resident VMs")
	}

	return nil
}
