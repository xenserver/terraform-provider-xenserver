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
	if len(joinSupporters) == 0 {
		tflog.Debug(ctx, "No host to join.")
		return nil
	}
	ejectSupporters := make([]string, 0, len(plan.EjectSupporters.Elements()))
	diags = plan.EjectSupporters.ElementsAs(ctx, &ejectSupporters, false)
	if diags.HasError() {
		return errors.New("unable to access eject supporters in config data")
	}
	// if coordinator host has scheme, remove it
	coordinatorIP := regexp.MustCompile(`^https?://`).ReplaceAllString(coordinatorConf.Host, "")
	supportersHosts := []string{}
	for _, supporter := range joinSupporters {
		// check if the supporter is duplicated in 'join_supporters', skip if it is
		if slices.Contains(supportersHosts, supporter.Host.ValueString()) {
			tflog.Debug(ctx, "Skip duplicate supporter in 'join_supporters'")
			continue
		}
		supportersHosts = append(supportersHosts, supporter.Host.ValueString())

		supporterSession, _, err := loginServer(supporter.Host.ValueString(), supporter.Username.ValueString(), supporter.Password.ValueString())
		if err != nil {
			if strings.Contains(err.Error(), "HOST_IS_SLAVE") {
				// check if the supporter in current pool
				re := regexp.MustCompile(`data \[([^']*)\]`)
				matches := re.FindStringSubmatch(err.Error())
				if len(matches) > 1 && matches[1] == coordinatorIP {
					tflog.Debug(ctx, "Host "+supporter.Host.ValueString()+" is already in this pool, continue")
					continue
				} else {
					return errors.New("unable to join supporter host " + supporter.Host.ValueString() + ", it's not a standalone host")
				}
			}
			return errors.New("login supporter host " + supporter.Host.ValueString() + "failed. " + err.Error())
		}

		hostRefs, err := xenapi.Host.GetAll(supporterSession)
		if err != nil {
			return errors.New("unable to get the supporter host refs. " + err.Error())
		}
		// check if the supporter is a pool with more than 1 host, return error if it is
		if len(hostRefs) > 1 {
			return errors.New("unable to join supporter host " + supporter.Host.ValueString() + ", it's not a standalone host")
		}
		supporterRef := hostRefs[0]
		supporterUUID, err := getUUIDFromHostRef(supporterSession, supporterRef)
		if err != nil {
			return errors.New(err.Error() + ". \n\nsupporter host is: " + supporter.Host.ValueString())
		}

		// check if the host is in eject_supporters, return error if it is
		if slices.Contains(ejectSupporters, supporterUUID) {
			return errors.New("host " + supporter.Host.ValueString() + " with uuid " + supporterUUID + " is in eject_supporters, can't join the pool")
		}

		err = xenapi.Pool.Join(supporterSession, coordinatorIP, coordinatorConf.Username, coordinatorConf.Password)
		if err != nil {
			return errors.New(err.Error() + ". \n\nPool join failed with host uuid: " + supporterUUID)
		}
		joinedSupporterUUIDs = append(joinedSupporterUUIDs, supporterUUID)
	}

	return waitAllSupportersLive(ctx, coordinatorSession, joinedSupporterUUIDs)
}

func waitAllSupportersLive(ctx context.Context, session *xenapi.Session, supporterUUIDs []string) error {
	tflog.Debug(ctx, "---> Waiting for all supporters to join the pool...")
	operation := func() error {
		for _, supporterUUID := range supporterUUIDs {
			hostRef, err := xenapi.Host.GetByUUID(session, supporterUUID)
			if err != nil {
				return errors.New("unable to get host ref by UUID " + supporterUUID + "!\n" + err.Error())
			}
			hostEnabled, err := xenapi.Host.GetEnabled(session, hostRef)
			if err != nil {
				return errors.New("unable to get host enabled status. " + err.Error())
			}
			if hostEnabled {
				tflog.Debug(ctx, "Host "+supporterUUID+" is enabled")
				continue
			} else {
				tflog.Debug(ctx, "Host "+supporterUUID+" is disabled, retrying...")
				return errors.New("host " + supporterUUID + " is disabled")
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
	tflog.Debug(ctx, "---> All supporters success join the pool.")

	return nil
}

func poolEject(ctx context.Context, session *xenapi.Session, plan poolResourceModel) error {
	ejectSupporters := make([]string, 0, len(plan.EjectSupporters.Elements()))
	diags := plan.EjectSupporters.ElementsAs(ctx, &ejectSupporters, false)
	if diags.HasError() {
		return errors.New("unable to access eject supporters in config data")
	}
	if len(ejectSupporters) == 0 {
		tflog.Debug(ctx, "No host to eject.")
		return nil
	}
	// get all the hosts current in pool
	beforeEjectHostRefs, err := xenapi.Host.GetAll(session)
	if err != nil {
		return errors.New("unable to get the origin host refs in pool. " + err.Error())
	}
	beforeEjectHosts := make(map[string]xenapi.HostRef)
	for _, ref := range beforeEjectHostRefs {
		uuid, err := getUUIDFromHostRef(session, ref)
		if err != nil {
			return err
		}
		beforeEjectHosts[uuid] = ref
	}

	for _, hostUUID := range ejectSupporters {
		tflog.Debug(ctx, "Ejecting pool with host: "+hostUUID)
		// check if the supporter is not in the pool, skip if it is
		hostRef, ok := beforeEjectHosts[hostUUID]
		if !ok {
			tflog.Debug(ctx, "Skip eject as supporter is not in pool")
			continue
		}
		err := xenapi.Pool.Eject(session, hostRef)
		if err != nil {
			return errors.New("unable to eject pool with host UUID " + hostUUID + "!\n" + err.Error())
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
		return coordinatorRef, coordinatorUUID, errors.New("unable to get pool master. " + err.Error())
	}
	coordinatorUUID, err = getUUIDFromHostRef(session, coordinatorRef)
	if err != nil {
		return coordinatorRef, coordinatorUUID, err
	}
	return coordinatorRef, coordinatorUUID, nil
}

func getPoolRef(session *xenapi.Session) (xenapi.PoolRef, error) {
	poolRefs, err := xenapi.Pool.GetAll(session)
	if err != nil {
		return "", errors.New("unable to get pool refs. " + err.Error())
	}

	return poolRefs[0], nil
}

func cleanupPoolResource(session *xenapi.Session, poolRef xenapi.PoolRef) error {
	err := xenapi.Pool.SetNameLabel(session, poolRef, "")
	if err != nil {
		return errors.New("unable to set pool name_label. " + err.Error())
	}

	coordinatorRef, _, err := getCoordinatorRef(session)
	if err != nil {
		return errors.New(err.Error())
	}

	// eject supporters
	hostRefs, err := xenapi.Host.GetAll(session)
	if err != nil {
		return errors.New("unable to get host all refs. " + err.Error())
	}

	for _, hostRef := range hostRefs {
		isCoordinator := hostRef == coordinatorRef
		if isCoordinator {
			continue
		}

		err = xenapi.Pool.Eject(session, hostRef)
		if err != nil {
			return errors.New("Pool eject failed when clean up. " + err.Error())
		}
	}

	return nil
}

func setPool(session *xenapi.Session, poolRef xenapi.PoolRef, poolParams poolParams) error {
	err := xenapi.Pool.SetNameLabel(session, poolRef, poolParams.NameLabel)
	if err != nil {
		return errors.New("unable to set pool name_label. " + err.Error())
	}

	err = xenapi.Pool.SetNameDescription(session, poolRef, poolParams.NameDescription)
	if err != nil {
		return errors.New("unable to set pool name_description. " + err.Error())
	}

	if poolParams.DefaultSRUUID != "" {
		srRef, err := xenapi.SR.GetByUUID(session, poolParams.DefaultSRUUID)
		if err != nil {
			return errors.New("unable to get SR by UUID " + poolParams.DefaultSRUUID + "!\n" + err.Error())
		}

		// Check if the SR is non-shared, return error if it is
		shared, err := xenapi.SR.GetShared(session, srRef)
		if err != nil {
			return errors.New("unable to get SR shared status. " + err.Error())
		}

		if !shared {
			return errors.New("SR with uuid " + poolParams.DefaultSRUUID + " is non-shared SR")
		}

		err = xenapi.Pool.SetDefaultSR(session, poolRef, srRef)
		if err != nil {
			return errors.New("unable to set pool default_SR. " + err.Error())
		}
	}

	if poolParams.ManagementNetworkUUID != "" {
		networkRef, err := xenapi.Network.GetByUUID(session, poolParams.ManagementNetworkUUID)
		if err != nil {
			return errors.New("unable to get network by UUID " + poolParams.ManagementNetworkUUID + "!\n" + err.Error())
		}

		err = xenapi.Pool.ManagementReconfigure(session, networkRef)
		if err != nil {
			return errors.New("unable to reconfigure pool management network " + poolParams.ManagementNetworkUUID + "!\n" + err.Error())
		}

		// wait for toolstack restart
		time.Sleep(60 * time.Second)
	}

	return nil
}

func getManagementNetworkUUID(session *xenapi.Session, coordinatorRef xenapi.HostRef) (string, error) {
	pifRefs, err := xenapi.Host.GetPIFs(session, coordinatorRef)
	if err != nil {
		return "", errors.New("unable to get host PIFs. " + err.Error())
	}

	for _, pifRef := range pifRefs {
		isManagement, err := xenapi.PIF.GetManagement(session, pifRef)
		if err != nil {
			return "", errors.New("unable to get PIF management. " + err.Error())
		}

		if isManagement {
			networkRef, err := xenapi.PIF.GetNetwork(session, pifRef)
			if err != nil {
				return "", errors.New("unable to get PIF network. " + err.Error())
			}

			networkRecord, err := xenapi.Network.GetRecord(session, networkRef)
			if err != nil {
				return "", errors.New("unable to get network record. " + err.Error())
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
		srUUID, err := getUUIDFromSRRef(session, record.DefaultSR)
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
