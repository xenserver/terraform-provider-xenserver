package xenserver

import (
	"errors"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"xenapi"
)

type poolResourceModel struct {
	NameLabel             types.String             `tfsdk:"name_label"`
	NameDescription       types.String             `tfsdk:"name_description"`
	DefaultSRUUID         types.String             `tfsdk:"default_sr"`
	ManagementNetworkUUID types.String             `tfsdk:"management_network"`
	Supporters            []supporterResourceModel `tfsdk:"supporters"`
	UUID                  types.String             `tfsdk:"uuid"`
	ID                    types.String             `tfsdk:"id"`
}

type supporterResourceModel struct {
	Host     types.String `tfsdk:"host"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	UUID     types.String `tfsdk:"uuid"`
}

type poolParams struct {
	NameLabel             string
	NameDescription       string
	DefaultSRUUID         string
	ManagementNetworkUUID string
	Supporters            []supporterParams
}

type supporterParams struct {
	Host     string
	Username string
	Password string
}

func PoolSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name_label": schema.StringAttribute{
			MarkdownDescription: "The name of the pool.",
			Required:            true,
		},
		"name_description": schema.StringAttribute{
			MarkdownDescription: "The description of the pool, default to be `\"\"`.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"default_sr": schema.StringAttribute{
			MarkdownDescription: "The default SR UUID of the pool.",
			Required:            true,
		},
		"management_network": schema.StringAttribute{
			MarkdownDescription: "The management network UUID of the pool.",
			Optional:            true,
			Computed:            true,
		},
		"supporters": schema.SetNestedAttribute{
			MarkdownDescription: "The set of pool supporters which will join the pool.",
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"host": schema.StringAttribute{
						MarkdownDescription: "The address of the host.",
						Optional:            true,
					},
					"username": schema.StringAttribute{
						MarkdownDescription: "The user name of the host.",
						Optional:            true,
					},
					"password": schema.StringAttribute{
						MarkdownDescription: "The password of the host.",
						Optional:            true,
						Sensitive:           true,
					},
					"uuid": schema.StringAttribute{
						MarkdownDescription: "The UUID of the host.",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			Optional: true,
		},
		"uuid": schema.StringAttribute{
			MarkdownDescription: "The UUID of the pool.",
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"id": schema.StringAttribute{
			MarkdownDescription: "The test ID of the pool.",
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
	}
}

func getPoolParams(plan poolResourceModel) (poolParams, error) {
	var params poolParams
	params.NameLabel = plan.NameLabel.ValueString()
	params.NameDescription = plan.NameDescription.ValueString()
	params.DefaultSRUUID = plan.DefaultSRUUID.ValueString()
	if !plan.ManagementNetworkUUID.IsUnknown() {
		params.ManagementNetworkUUID = plan.ManagementNetworkUUID.ValueString()
	}

	for _, host := range plan.Supporters {
		hostParams, err := getSupporterParams(host)
		if err != nil {
			return params, err
		}
		params.Supporters = append(params.Supporters, hostParams)
	}
	return params, nil
}

func getSupporterParams(plan supporterResourceModel) (supporterParams, error) {
	var params supporterParams
	if plan.Host.IsUnknown() || plan.Username.IsUnknown() || plan.Password.IsUnknown() {
		return params, errors.New("host url, username, and password	required when pool join")
	}

	params.Host = plan.Host.ValueString()
	params.Username = plan.Username.ValueString()
	params.Password = plan.Password.ValueString()

	return params, nil
}

func poolJoin(providerConfig *providerModel, poolParams poolParams) error {
	for _, supporter := range poolParams.Supporters {
		supporterSession, err := loginServer(supporter.Host, supporter.Username, supporter.Password)
		if err != nil {
			return err
		}

		err = xenapi.Pool.Join(supporterSession, providerConfig.Host.ValueString(), providerConfig.Username.ValueString(), providerConfig.Password.ValueString())
		if err != nil {
			return errors.New(err.Error())
		}
	}
	return nil
}

func getPoolRef(session *xenapi.Session) (xenapi.PoolRef, error) {
	poolRefs, err := xenapi.Pool.GetAll(session)
	if err != nil {
		return "", errors.New(err.Error())
	}

	return poolRefs[0], nil
}

func poolResourceModelUpdate(session *xenapi.Session, poolRef xenapi.PoolRef, plan poolResourceModel) error {
	err := xenapi.Pool.SetNameLabel(session, poolRef, plan.NameLabel.ValueString())
	if err != nil {
		return errors.New(err.Error())
	}

	err = xenapi.Pool.SetNameDescription(session, poolRef, plan.NameDescription.ValueString())
	if err != nil {
		return errors.New(err.Error())
	}

	srRef, err := xenapi.SR.GetByUUID(session, plan.DefaultSRUUID.ValueString())
	if err != nil {
		return errors.New(err.Error())
	}

	err = xenapi.Pool.SetDefaultSR(session, poolRef, srRef)
	if err != nil {
		return errors.New(err.Error())
	}

	if !plan.ManagementNetworkUUID.IsUnknown() {
		networkRef, err := xenapi.Network.GetByUUID(session, plan.ManagementNetworkUUID.ValueString())
		if err != nil {
			return errors.New(err.Error())
		}

		err = xenapi.Pool.ManagementReconfigure(session, networkRef)
		if err != nil {
			return errors.New(err.Error())
		}
		// wait for toolstack restart
		time.Sleep(60 * time.Second)
	}

	return nil
}

func cleanupPoolResource(session *xenapi.Session, poolRef xenapi.PoolRef) error {
	err := xenapi.Pool.SetNameLabel(session, poolRef, "")
	if err != nil {
		return errors.New(err.Error())
	}
	return nil
}

func setPool(session *xenapi.Session, poolRef xenapi.PoolRef, poolParams poolParams) error {
	err := xenapi.Pool.SetNameLabel(session, poolRef, poolParams.NameLabel)
	if err != nil {
		return errors.New(err.Error())
	}

	err = xenapi.Pool.SetNameDescription(session, poolRef, poolParams.NameDescription)
	if err != nil {
		return errors.New(err.Error())
	}

	srRef, err := xenapi.SR.GetByUUID(session, poolParams.DefaultSRUUID)
	if err != nil {
		return errors.New(err.Error())
	}

	err = xenapi.Pool.SetDefaultSR(session, poolRef, srRef)
	if err != nil {
		return errors.New(err.Error())
	}

	if poolParams.ManagementNetworkUUID != "" {
		networkRef, err := xenapi.Network.GetByUUID(session, poolParams.ManagementNetworkUUID)
		if err != nil {
			return errors.New(err.Error() + ", uuid: " + poolParams.ManagementNetworkUUID)
		}

		err = xenapi.Pool.ManagementReconfigure(session, networkRef)
		if err != nil {
			return errors.New(err.Error() + ", uuid: " + poolParams.ManagementNetworkUUID)
		}
		// wait for toolstack restart
		time.Sleep(60 * time.Second)
	}

	return nil
}

func getManagementNetworkUUID(session *xenapi.Session, coordinatorRef xenapi.HostRef) (string, error) {
	pifRefs, err := xenapi.Host.GetPIFs(session, coordinatorRef)
	if err != nil {
		return "", errors.New(err.Error())
	}

	for _, pifRef := range pifRefs {
		isManagement, err := xenapi.PIF.GetManagement(session, pifRef)
		if err != nil {
			return "", errors.New(err.Error())
		}

		if isManagement {
			networkRef, err := xenapi.PIF.GetNetwork(session, pifRef)
			if err != nil {
				return "", errors.New(err.Error())
			}

			networkRecord, err := xenapi.Network.GetRecord(session, networkRef)
			if err != nil {
				return "", errors.New(err.Error())
			}

			return networkRecord.UUID, nil
		}
	}
	return "", errors.New("no management network found")
}

func updatePoolResourceModel(session *xenapi.Session, record xenapi.PoolRecord, data *poolResourceModel) error {
	data.NameLabel = types.StringValue(record.NameLabel)
	data.NameDescription = types.StringValue(record.NameDescription)
	srUUID, err := xenapi.SR.GetUUID(session, record.DefaultSR)
	if err != nil {
		return errors.New(err.Error())
	}
	data.DefaultSRUUID = types.StringValue(srUUID)
	return updatePoolResourceModelComputed(session, record, data)
}

func updatePoolResourceModelComputed(session *xenapi.Session, record xenapi.PoolRecord, data *poolResourceModel) error {
	data.UUID = types.StringValue(record.UUID)
	data.ID = types.StringValue(record.UUID)

	networkUUID, err := getManagementNetworkUUID(session, record.Master)
	if err != nil {
		return err
	}

	data.ManagementNetworkUUID = types.StringValue(networkUUID)

	return nil
}
