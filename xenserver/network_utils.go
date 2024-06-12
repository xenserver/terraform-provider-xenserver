package xenserver

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"xenapi"
)

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
