package xenserver

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"xenapi"
)

// srDataSourceModel describes the data source data model.
type networkDataSourceModel struct {
	NameLabel types.String        `tfsdk:"name_label"`
	UUID      types.String        `tfsdk:"uuid"`
	DataItems []networkRecordData `tfsdk:"data_items"`
}

type networkRecordData struct {
	UUID               types.String `tfsdk:"uuid"`
	NameLabel          types.String `tfsdk:"name_label"`
	NameDescription    types.String `tfsdk:"name_description"`
	AllowedOperations  types.List   `tfsdk:"allowed_operations"`
	CurrentOperations  types.Map    `tfsdk:"current_operations"`
	VIFs               types.List   `tfsdk:"vifs"`
	PIFs               types.List   `tfsdk:"pifs"`
	MTU                types.Int64  `tfsdk:"mtu"`
	OtherConfig        types.Map    `tfsdk:"other_config"`
	Bridge             types.String `tfsdk:"bridge"`
	Managed            types.Bool   `tfsdk:"managed"`
	Blobs              types.Map    `tfsdk:"blobs"`
	Tags               types.List   `tfsdk:"tags"`
	DefaultLockingMode types.String `tfsdk:"default_locking_mode"`
	AssignedIps        types.Map    `tfsdk:"assigned_ips"`
	Purpose            types.List   `tfsdk:"purpose"`
}

func updateNetworkRecordData(ctx context.Context, record xenapi.NetworkRecord, data *networkRecordData) error {
	data.UUID = types.StringValue(record.UUID)
	data.NameLabel = types.StringValue(record.NameLabel)
	data.NameDescription = types.StringValue(record.NameDescription)
	var diags diag.Diagnostics
	data.AllowedOperations, diags = types.ListValueFrom(ctx, types.StringType, record.AllowedOperations)
	if diags.HasError() {
		return errors.New("unable to read network allowed operations")
	}
	data.CurrentOperations, diags = types.MapValueFrom(ctx, types.StringType, record.CurrentOperations)
	if diags.HasError() {
		return errors.New("unable to read network current operation")
	}
	data.VIFs, diags = types.ListValueFrom(ctx, types.StringType, record.VIFs)
	if diags.HasError() {
		return errors.New("unable to read network VIFs")
	}
	data.PIFs, diags = types.ListValueFrom(ctx, types.StringType, record.PIFs)
	if diags.HasError() {
		return errors.New("unable to read network PIFs")
	}
	data.MTU = types.Int64Value(int64(record.MTU))
	data.OtherConfig, diags = types.MapValueFrom(ctx, types.StringType, record.OtherConfig)
	if diags.HasError() {
		return errors.New("unable to read network other config")
	}
	data.Bridge = types.StringValue(record.Bridge)
	data.Managed = types.BoolValue(record.Managed)
	data.Blobs, diags = types.MapValueFrom(ctx, types.StringType, record.Blobs)
	if diags.HasError() {
		return errors.New("unable to read network blobs")
	}
	data.Tags, diags = types.ListValueFrom(ctx, types.StringType, record.Tags)
	if diags.HasError() {
		return errors.New("unable to read network tags")
	}
	data.DefaultLockingMode = types.StringValue(string(record.DefaultLockingMode))
	data.AssignedIps, diags = types.MapValueFrom(ctx, types.StringType, record.AssignedIps)
	if diags.HasError() {
		return errors.New("unable to read network assigned_ips")
	}
	data.Purpose, diags = types.ListValueFrom(ctx, types.StringType, record.Purpose)
	if diags.HasError() {
		return errors.New("unable to read network purpose")
	}

	return nil
}

// NetworkResourceModel describes the resource data model.
type networkResourceModel struct {
	NameLabel       types.String `tfsdk:"name_label"`
	NameDescription types.String `tfsdk:"name_description"`
	MTU             types.Int64  `tfsdk:"mtu"`
	Managed         types.Bool   `tfsdk:"managed"`
	OtherConfig     types.Map    `tfsdk:"other_config"`
	UUID            types.String `tfsdk:"id"`
}

// Update NetworkResourceModel base on new NetworkRecord
func updateNetworkResourceModel(ctx context.Context, networkRecord xenapi.NetworkRecord, data *networkResourceModel) error {
	data.NameLabel = types.StringValue(networkRecord.NameLabel)

	err := updateNetworkResourceModelComputed(ctx, networkRecord, data)
	if err != nil {
		return err
	}
	return nil
}

// Update NetworkResourceModel computed field base on new NetworkRecord
func updateNetworkResourceModelComputed(ctx context.Context, networkRecord xenapi.NetworkRecord, data *networkResourceModel) error {
	data.UUID = types.StringValue(networkRecord.UUID)
	data.NameDescription = types.StringValue(networkRecord.NameDescription)
	data.MTU = types.Int64Value(int64(networkRecord.MTU))
	data.Managed = types.BoolValue(networkRecord.Managed)

	otherConfig, diags := types.MapValueFrom(ctx, types.StringType, networkRecord.OtherConfig)
	data.OtherConfig = otherConfig
	if diags.HasError() {
		return errors.New("unable to update data for network other_config")
	}
	return nil
}

// update fields of network resource by xen-api sdk
func updateNetworkFields(ctx context.Context, session *xenapi.Session, networkRef xenapi.NetworkRef, data networkResourceModel) error {
	err := xenapi.Network.SetNameLabel(session, networkRef, data.NameLabel.ValueString())
	if err != nil {
		return errors.New("unable to update network name_label")
	}

	err = xenapi.Network.SetNameDescription(session, networkRef, data.NameDescription.ValueString())
	if err != nil {
		return errors.New("unable to update network name_description")
	}

	err = xenapi.Network.SetMTU(session, networkRef, int(data.MTU.ValueInt64()))
	if err != nil {
		return errors.New("unable to update network mtu")
	}

	otherConfig := make(map[string]string, len(data.OtherConfig.Elements()))
	diags := data.OtherConfig.ElementsAs(ctx, &otherConfig, false)
	if diags.HasError() {
		return errors.New("unable to update network other_config")
	}

	err = xenapi.Network.SetOtherConfig(session, networkRef, otherConfig)
	if err != nil {
		return errors.New("unable to update network other_config")
	}
	return nil
}
