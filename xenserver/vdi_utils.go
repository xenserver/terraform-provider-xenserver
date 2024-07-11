package xenserver

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"xenapi"
)

type vdiResourceModel struct {
	NameLabel       types.String `tfsdk:"name_label"`
	NameDescription types.String `tfsdk:"name_description"`
	SR              types.String `tfsdk:"sr_uuid"`
	VirtualSize     types.Int64  `tfsdk:"virtual_size"`
	Type            types.String `tfsdk:"type"`
	Sharable        types.Bool   `tfsdk:"sharable"`
	ReadOnly        types.Bool   `tfsdk:"read_only"`
	OtherConfig     types.Map    `tfsdk:"other_config"`
	UUID            types.String `tfsdk:"uuid"`
	ID              types.String `tfsdk:"id"`
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
	record.VirtualSize = int(data.VirtualSize.ValueInt64())
	record.Type = xenapi.VdiType(data.Type.ValueString())
	record.Sharable = data.Sharable.ValueBool()
	record.ReadOnly = data.ReadOnly.ValueBool()

	diags := data.OtherConfig.ElementsAs(ctx, &record.OtherConfig, false)
	if diags.HasError() {
		return record, errors.New("unable to access VDI other config")
	}

	return record, nil
}

func updateVDIResourceModel(ctx context.Context, session *xenapi.Session, record xenapi.VDIRecord, data *vdiResourceModel) error {
	data.NameLabel = types.StringValue(record.NameLabel)
	srUUID, err := xenapi.SR.GetUUID(session, record.SR)
	if err != nil {
		return errors.New(err.Error())
	}
	data.SR = types.StringValue(srUUID)
	data.VirtualSize = types.Int64Value(int64(record.VirtualSize))

	return updateVDIResourceModelComputed(ctx, record, data)
}

func updateVDIResourceModelComputed(ctx context.Context, record xenapi.VDIRecord, data *vdiResourceModel) error {
	data.UUID = types.StringValue(record.UUID)
	data.ID = types.StringValue(record.UUID)
	data.NameDescription = types.StringValue(record.NameDescription)
	data.Type = types.StringValue(string(record.Type))
	data.Sharable = types.BoolValue(record.Sharable)
	data.ReadOnly = types.BoolValue(record.ReadOnly)
	var diags diag.Diagnostics
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

func cleanupVDIResource(session *xenapi.Session, ref xenapi.VDIRef) error {
	err := xenapi.VDI.Destroy(session, ref)
	if err != nil {
		return errors.New(err.Error())
	}
	return nil
}
