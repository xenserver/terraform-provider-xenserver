package xenserver

import (
	"context"
	"errors"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"xenapi"
)

type poolResourceModel struct {
	NameLabel             types.String `tfsdk:"name_label"`
	NameDescription       types.String `tfsdk:"name_description"`
	DefaultSRUUID         types.String `tfsdk:"default_sr"`
	ManagementNetworkUUID types.String `tfsdk:"management_network"`
	JoinSupporters        types.Set    `tfsdk:"join_supporters"`
	EjectSupporters       types.Set    `tfsdk:"eject_supporters"`
	UUID                  types.String `tfsdk:"uuid"`
	ID                    types.String `tfsdk:"id"`
}

type joinSupporterResourceModel struct {
	Host     types.String `tfsdk:"host"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

type poolParams struct {
	NameLabel             string
	NameDescription       string
	DefaultSRUUID         string
	ManagementNetworkUUID string
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
			MarkdownDescription: "The default SR UUID of the pool. this SR should be shared SR.",
			Optional:            true,
			Computed:            true,
		},
		"management_network": schema.StringAttribute{
			MarkdownDescription: "The management network UUID of the pool." +
				"\n\n-> **Note:** " +
				"1. The management network would be reconfigured only when the management network UUID is provided.<br>" +
				"2. All of the hosts in the pool should have the same management network with network configuration, and you can set network configuration by resource `pif_configure`.<br>" +
				"3. It is not recommended to set the `management_network` with the `join_supporters` and `eject_supporters` attributes together.<br>",
			Optional: true,
			Computed: true,
		},
		"join_supporters": schema.SetNestedAttribute{
			MarkdownDescription: "The set of pool supporters which will join the pool." +
				"\n\n-> **Note:** 1. It would raise error if a supporter is in both join_supporters and eject_supporters.<br>" +
				"2. The join operation would be performed only when the host, username, and password are provided.<br>",
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
				},
			},
			Optional: true,
		},
		"eject_supporters": schema.SetAttribute{
			MarkdownDescription: "The set of pool supporters which will be ejected from the pool.",
			ElementType:         types.StringType,
			Optional:            true,
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

func getPoolParams(plan poolResourceModel) poolParams {
	var params poolParams
	params.NameLabel = plan.NameLabel.ValueString()
	params.NameDescription = plan.NameDescription.ValueString()
	params.DefaultSRUUID = plan.DefaultSRUUID.ValueString()
	if !plan.ManagementNetworkUUID.IsUnknown() {
		params.ManagementNetworkUUID = plan.ManagementNetworkUUID.ValueString()
	}

	return params
}

func poolJoin(ctx context.Context, coordinatorSession *xenapi.Session, coordinatorConf *coordinatorConf, plan poolResourceModel) error {
	joinedSupporterUUIDs := []string{}
	joinSupporters := make([]joinSupporterResourceModel, 0, len(plan.JoinSupporters.Elements()))
	diags := plan.JoinSupporters.ElementsAs(ctx, &joinSupporters, false)
	if diags.HasError() {
		return errors.New("unable to access join supporters in config data")
	}
	for _, supporter := range joinSupporters {
		supporterSession, err := loginServer(supporter.Host.ValueString(), supporter.Username.ValueString(), supporter.Password.ValueString())
		if err != nil {
			if strings.Contains(err.Error(), "HOST_IS_SLAVE") {
				tflog.Debug(ctx, "Host is already in the pool, continue")
				continue
			}
			return errors.New("Login Supporter Host Failed!\n" + err.Error() + ", host: " + supporter.Host.ValueString())
		}

		hostRefs, err := xenapi.Host.GetAll(supporterSession)
		if err != nil {
			return errors.New(err.Error())
		}

		if len(hostRefs) > 1 {
			return errors.New("Supporter host " + supporter.Host.ValueString() + " is not a standalone host")
		}

		supporterRef := hostRefs[0]

		// Check if the host is already in the pool, continue if it is
		beforeJoinHostRefs, err := xenapi.Host.GetAll(coordinatorSession)
		if err != nil {
			return errors.New(err.Error())
		}

		if slices.Contains(beforeJoinHostRefs, supporterRef) {
			continue
		}

		supporterUUID, err := xenapi.Host.GetUUID(supporterSession, supporterRef)
		if err != nil {
			return errors.New(err.Error() + ". \n\nunable to Get Host UUID with host: " + supporter.Host.ValueString())
		}

		ejectSupporters := make([]string, 0, len(plan.EjectSupporters.Elements()))
		diags := plan.EjectSupporters.ElementsAs(ctx, &ejectSupporters, false)
		if diags.HasError() {
			return errors.New("unable to access eject supporters in config data")
		}

		// Check if the host is in eject_supporters, return error if it is
		if slices.Contains(ejectSupporters, supporterUUID) {
			return errors.New("host " + supporter.Host.ValueString() + " with uuid " + supporterUUID + " is in eject_supporters, can't join the pool")
		}

		// if coordinator host has scheme, remove it
		coordinatorIP := regexp.MustCompile(`^https?://`).ReplaceAllString(coordinatorConf.Host, "")
		err = xenapi.Pool.Join(supporterSession, coordinatorIP, coordinatorConf.Username, coordinatorConf.Password)
		if err != nil {
			return errors.New(err.Error() + ". \n\nPool join failed with host uuid: " + supporterUUID)
		}

		joinedSupporterUUIDs = append(joinedSupporterUUIDs, supporterUUID)
	}

	return waitAllSupportersLive(ctx, coordinatorSession, joinedSupporterUUIDs)
}

func waitAllSupportersLive(ctx context.Context, session *xenapi.Session, supporterUUIDs []string) error {
	tflog.Debug(ctx, "Waiting for all supporters to join the pool...")
	operation := func() error {
		for _, supporterUUID := range supporterUUIDs {
			hostRef, err := xenapi.Host.GetByUUID(session, supporterUUID)
			if err != nil {
				return errors.New("unable to Get Host by UUID " + supporterUUID + "!\n" + err.Error())
			}

			hostMetricsRef, err := xenapi.Host.GetMetrics(session, hostRef)
			if err != nil {
				return errors.New("unable to Get Host Metrics with UUID " + supporterUUID + "!\n" + err.Error())
			}

			hostIsLive, err := xenapi.HostMetrics.GetLive(session, hostMetricsRef)
			if err != nil {
				return errors.New("unable to Get Host Live Status with UUID " + supporterUUID + "!\n" + err.Error())
			}

			if hostIsLive {
				tflog.Debug(ctx, "Host "+supporterUUID+" is live")
				continue
			} else {
				tflog.Debug(ctx, "Host "+supporterUUID+" is not live, retrying...")
				return errors.New("host " + supporterUUID + " is not live")
			}
		}
		return nil
	}

	b := backoff.NewExponentialBackOff()
	b.MaxInterval = 10 * time.Second
	b.MaxElapsedTime = 5 * time.Minute
	err := backoff.Retry(operation, b)
	if err != nil {
		return errors.New(err.Error())
	}

	return nil
}

func poolEject(ctx context.Context, session *xenapi.Session, plan poolResourceModel) error {
	ejectSupporters := make([]string, 0, len(plan.EjectSupporters.Elements()))
	diags := plan.EjectSupporters.ElementsAs(ctx, &ejectSupporters, false)
	if diags.HasError() {
		return errors.New("unable to access eject supporters in config data")
	}

	for _, hostUUID := range ejectSupporters {
		tflog.Debug(ctx, "Ejecting pool with host: "+hostUUID)

		operation := func() error {
			hostRef, err := xenapi.Host.GetByUUID(session, hostUUID)
			if err != nil {
				return errors.New(err.Error())
			}
			return xenapi.Pool.Eject(session, hostRef)
		}

		err := backoff.Retry(operation, backoff.NewExponentialBackOff())
		if err != nil {
			return errors.New(err.Error())
		}
	}

	return nil
}

func getCoordinatorRef(session *xenapi.Session) (xenapi.HostRef, string, error) {
	var coordinatorRef xenapi.HostRef
	var coordinatorUUID string
	poolRef, err := getPoolRef(session)
	if err != nil {
		return coordinatorRef, coordinatorUUID, errors.New(err.Error())
	}
	coordinatorRef, err = xenapi.Pool.GetMaster(session, poolRef)
	if err != nil {
		return coordinatorRef, coordinatorUUID, errors.New(err.Error())
	}
	coordinatorUUID, err = xenapi.Host.GetUUID(session, coordinatorRef)
	if err != nil {
		return coordinatorRef, coordinatorUUID, errors.New(err.Error())
	}
	return coordinatorRef, coordinatorUUID, nil
}

func getPoolRef(session *xenapi.Session) (xenapi.PoolRef, error) {
	poolRefs, err := xenapi.Pool.GetAll(session)
	if err != nil {
		return "", errors.New(err.Error())
	}

	return poolRefs[0], nil
}

func cleanupPoolResource(session *xenapi.Session, poolRef xenapi.PoolRef) error {
	err := xenapi.Pool.SetNameLabel(session, poolRef, "")
	if err != nil {
		return errors.New(err.Error())
	}

	// eject supporters
	coordinatorRef, _, err := getCoordinatorRef(session)
	if err != nil {
		return errors.New(err.Error())
	}

	// eject supporters
	hostRefs, err := xenapi.Host.GetAll(session)
	if err != nil {
		return errors.New(err.Error())
	}

	for _, hostRef := range hostRefs {
		isCoordinator := hostRef == coordinatorRef
		if isCoordinator {
			continue
		}

		operation := func() error {
			return xenapi.Pool.Eject(session, hostRef)
		}

		err = backoff.Retry(operation, backoff.NewExponentialBackOff())
		if err != nil {
			return errors.New(err.Error())
		}
	}

	return nil
}

func setPool(session *xenapi.Session, poolRef xenapi.PoolRef, poolParams poolParams) error {
	err := xenapi.Pool.SetNameLabel(session, poolRef, poolParams.NameLabel)
	if err != nil {
		return errors.New("unable to Set NameLabel!\n" + err.Error())
	}

	err = xenapi.Pool.SetNameDescription(session, poolRef, poolParams.NameDescription)
	if err != nil {
		return errors.New("unable to Set NameDescription!\n" + err.Error())
	}

	if poolParams.DefaultSRUUID != "" {
		srRef, err := xenapi.SR.GetByUUID(session, poolParams.DefaultSRUUID)
		if err != nil {
			return errors.New("unable to Get SR by UUID!\n" + err.Error() + ", uuid: " + poolParams.DefaultSRUUID)
		}

		// Check if the SR is non-shared, return error if it is
		shared, err := xenapi.SR.GetShared(session, srRef)
		if err != nil {
			return errors.New("unable to Get SR shared status!\n" + err.Error())
		}

		if !shared {
			return errors.New("SR with uuid " + poolParams.DefaultSRUUID + " is non-shared SR")
		}

		err = xenapi.Pool.SetDefaultSR(session, poolRef, srRef)
		if err != nil {
			return errors.New("unable to Set DefaultSR on the Pool!\n" + err.Error())
		}
	}

	if poolParams.ManagementNetworkUUID != "" {
		networkRef, err := xenapi.Network.GetByUUID(session, poolParams.ManagementNetworkUUID)
		if err != nil {
			return errors.New("unable to Get Network by UUID!\n" + err.Error() + ", uuid: " + poolParams.ManagementNetworkUUID)
		}

		err = xenapi.Pool.ManagementReconfigure(session, networkRef)
		if err != nil {
			return errors.New("unable to Reconfigure Management Network on the Pool!\n" + err.Error() + ", uuid: " + poolParams.ManagementNetworkUUID)
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
	return updatePoolResourceModelComputed(session, record, data)
}

func updatePoolResourceModelComputed(session *xenapi.Session, record xenapi.PoolRecord, data *poolResourceModel) error {
	data.UUID = types.StringValue(record.UUID)
	data.ID = types.StringValue(record.UUID)
	data.NameDescription = types.StringValue(record.NameDescription)

	data.DefaultSRUUID = types.StringValue("")
	if string(record.DefaultSR) != "OpaqueRef:NULL" {
		srUUID, err := xenapi.SR.GetUUID(session, record.DefaultSR)
		if err == nil {
			data.DefaultSRUUID = types.StringValue(srUUID)
		}
	}

	networkUUID, err := getManagementNetworkUUID(session, record.Master)
	if err != nil {
		return err
	}

	data.ManagementNetworkUUID = types.StringValue(networkUUID)

	return nil
}
