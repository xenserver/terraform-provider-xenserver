package xenserver

import (
	"context"
	"errors"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"xenapi"
)

// srDataSourceModel describes the data source data model.
type srDataSourceModel struct {
	NameLabel types.String   `tfsdk:"name_label"`
	UUID      types.String   `tfsdk:"uuid"`
	DataItems []srRecordData `tfsdk:"data_items"`
}

type srRecordData struct {
	UUID                types.String `tfsdk:"uuid"`
	NameLabel           types.String `tfsdk:"name_label"`
	NameDescription     types.String `tfsdk:"name_description"`
	AllowedOperations   types.List   `tfsdk:"allowed_operations"`
	CurrentOperations   types.Map    `tfsdk:"current_operations"`
	VDIs                types.List   `tfsdk:"vdis"`
	PBDs                types.List   `tfsdk:"pbds"`
	VirtualAllocation   types.Int64  `tfsdk:"virtual_allocation"`
	PhysicalUtilisation types.Int64  `tfsdk:"physical_utilisation"`
	PhysicalSize        types.Int64  `tfsdk:"physical_size"`
	Type                types.String `tfsdk:"type"`
	ContentType         types.String `tfsdk:"content_type"`
	Shared              types.Bool   `tfsdk:"shared"`
	OtherConfig         types.Map    `tfsdk:"other_config"`
	Tags                types.List   `tfsdk:"tags"`
	SmConfig            types.Map    `tfsdk:"sm_config"`
	Blobs               types.Map    `tfsdk:"blobs"`
	LocalCacheEnabled   types.Bool   `tfsdk:"local_cache_enabled"`
	IntroducedBy        types.String `tfsdk:"introduced_by"`
	Clustered           types.Bool   `tfsdk:"clustered"`
	IsToolsSr           types.Bool   `tfsdk:"is_tools_sr"`
}

func updateSRRecordData(ctx context.Context, session *xenapi.Session, record xenapi.SRRecord, data *srRecordData) error {
	data.UUID = types.StringValue(record.UUID)
	data.NameLabel = types.StringValue(record.NameLabel)
	data.NameDescription = types.StringValue(record.NameDescription)
	var diags diag.Diagnostics
	data.AllowedOperations, diags = types.ListValueFrom(ctx, types.StringType, record.AllowedOperations)
	if diags.HasError() {
		return errors.New("unable to read SR allowed operations")
	}
	data.CurrentOperations, diags = types.MapValueFrom(ctx, types.StringType, record.CurrentOperations)
	if diags.HasError() {
		return errors.New("unable to read SR current operation")
	}
	vdiUUIDs, err := getVDIUUIDs(session, record.VDIs)
	if err != nil {
		return err
	}
	data.VDIs, diags = types.ListValueFrom(ctx, types.StringType, vdiUUIDs)
	if diags.HasError() {
		return errors.New("unable to read SR VDIs")
	}
	pbdUUIDs, err := getPBDUUIDs(session, record.PBDs)
	if err != nil {
		return err
	}
	data.PBDs, diags = types.ListValueFrom(ctx, types.StringType, pbdUUIDs)
	if diags.HasError() {
		return errors.New("unable to read SR PBDs")
	}
	data.VirtualAllocation = types.Int64Value(int64(record.VirtualAllocation))
	data.PhysicalUtilisation = types.Int64Value(int64(record.PhysicalUtilisation))
	data.PhysicalSize = types.Int64Value(int64(record.PhysicalSize))
	data.Type = types.StringValue(record.Type)
	data.ContentType = types.StringValue(record.ContentType)
	data.Shared = types.BoolValue(record.Shared)
	data.OtherConfig, diags = types.MapValueFrom(ctx, types.StringType, record.OtherConfig)
	if diags.HasError() {
		return errors.New("unable to read SR other config")
	}
	data.Tags, diags = types.ListValueFrom(ctx, types.StringType, record.Tags)
	if diags.HasError() {
		return errors.New("unable to read SR tags")
	}
	data.SmConfig, diags = types.MapValueFrom(ctx, types.StringType, record.SmConfig)
	if diags.HasError() {
		return errors.New("unable to read SR SM config")
	}
	blobs, err := getBlobUUIDsMap(session, record.Blobs)
	if err != nil {
		return err
	}
	data.Blobs, diags = types.MapValueFrom(ctx, types.StringType, blobs)
	if diags.HasError() {
		return errors.New("unable to read SR blobs")
	}
	data.LocalCacheEnabled = types.BoolValue(record.LocalCacheEnabled)
	introducedBy, err := getUUIDFromDRTaskRef(session, record.IntroducedBy)
	if err != nil {
		return err
	}
	data.IntroducedBy = types.StringValue(introducedBy)
	data.Clustered = types.BoolValue(record.Clustered)
	data.IsToolsSr = types.BoolValue(record.IsToolsSr)
	return nil
}

type srCreateParams struct {
	Host            xenapi.HostRef
	DeviceConfig    map[string]string
	PhysicalSize    int
	NameLabel       string
	NameDescription string
	TypeKey         string
	ContentType     string
	Shared          bool
	SmConfig        map[string]string
}

// srResourceModel describes the resource data model.
type srResourceModel struct {
	NameLabel       types.String `tfsdk:"name_label"`
	NameDescription types.String `tfsdk:"name_description"`
	Type            types.String `tfsdk:"type"`
	ContentType     types.String `tfsdk:"content_type"`
	Shared          types.Bool   `tfsdk:"shared"`
	SmConfig        types.Map    `tfsdk:"sm_config"`
	DeviceConfig    types.Map    `tfsdk:"device_config"`
	Host            types.String `tfsdk:"host"`
	UUID            types.String `tfsdk:"uuid"`
	ID              types.String `tfsdk:"id"`
}

func getSRCreateParams(ctx context.Context, session *xenapi.Session, data srResourceModel) (srCreateParams, error) {
	var params srCreateParams
	params.NameLabel = data.NameLabel.ValueString()
	params.NameDescription = data.NameDescription.ValueString()
	params.TypeKey = data.Type.ValueString()
	params.ContentType = data.ContentType.ValueString()
	params.Shared = data.Shared.ValueBool()
	diags := data.DeviceConfig.ElementsAs(ctx, &params.DeviceConfig, false)
	if diags.HasError() {
		return params, errors.New("unable to access SR device config data")
	}
	diags = data.SmConfig.ElementsAs(ctx, &params.SmConfig, false)
	if diags.HasError() {
		return params, errors.New("unable to access SR SM config data")
	}
	coordinatorRef, _, err := getCoordinatorRef(session)
	if err != nil {
		return params, err
	}
	params.Host = coordinatorRef
	if !data.Host.IsUnknown() {
		hostRef, err := xenapi.Host.GetByUUID(session, data.Host.ValueString())
		if err != nil {
			return params, errors.New(err.Error())
		}
		if params.Shared && hostRef != params.Host {
			return params, errors.New("shared SR can only created with coordinator host")
		}
		params.Host = hostRef
	}

	return params, nil
}

func getSRRecordAndPBDRecord(session *xenapi.Session, srRef xenapi.SRRef) (xenapi.SRRecord, xenapi.PBDRecord, error) {
	srRecord, err := xenapi.SR.GetRecord(session, srRef)
	if err != nil {
		return xenapi.SRRecord{}, xenapi.PBDRecord{}, errors.New(err.Error())
	}
	pbdRecord, err := xenapi.PBD.GetRecord(session, srRecord.PBDs[0])
	if err != nil {
		return xenapi.SRRecord{}, xenapi.PBDRecord{}, errors.New(err.Error())
	}
	return srRecord, pbdRecord, nil
}

func updateSRResourceModel(ctx context.Context, session *xenapi.Session, srRecord xenapi.SRRecord, pbdRecord xenapi.PBDRecord, data *srResourceModel) error {
	data.NameLabel = types.StringValue(srRecord.NameLabel)

	return updateSRResourceModelComputed(ctx, session, srRecord, pbdRecord, data)
}

func updateSRResourceModelComputed(ctx context.Context, session *xenapi.Session, srRecord xenapi.SRRecord, pbdRecord xenapi.PBDRecord, data *srResourceModel) error {
	data.UUID = types.StringValue(srRecord.UUID)
	data.ID = types.StringValue(srRecord.UUID)
	data.NameDescription = types.StringValue(srRecord.NameDescription)
	data.Type = types.StringValue(srRecord.Type)
	data.ContentType = types.StringValue(srRecord.ContentType)
	data.Shared = types.BoolValue(srRecord.Shared)
	var diags diag.Diagnostics
	data.SmConfig, diags = types.MapValueFrom(ctx, types.StringType, srRecord.SmConfig)
	if diags.HasError() {
		return errors.New("unable to access SR SM config")
	}
	hostRef, _, err := getCoordinatorRef(session)
	if err != nil {
		return err
	}
	if !srRecord.Shared {
		hostRef = pbdRecord.Host
	}
	hostUUID, err := getUUIDFromHostRef(session, hostRef)
	if err != nil {
		return err
	}
	data.Host = types.StringValue(hostUUID)
	data.DeviceConfig, diags = types.MapValueFrom(ctx, types.StringType, pbdRecord.DeviceConfig)
	if diags.HasError() {
		return errors.New("unable to access PBD device config")
	}

	return nil
}

func srResourceModelUpdateCheck(data srResourceModel, dataState srResourceModel) error {
	if data.Shared != dataState.Shared {
		return errors.New(`"shared" doesn't expected to be updated`)
	}
	if !data.Host.IsUnknown() && data.Host != dataState.Host {
		return errors.New(`"host" doesn't expected to be updated`)
	}
	if !reflect.DeepEqual(data.DeviceConfig, dataState.DeviceConfig) {
		return errors.New(`"device_config" doesn't expected to be updated`)
	}
	if data.Type != dataState.Type {
		return errors.New(`"type" doesn't expected to be updated`)
	}
	if data.ContentType != dataState.ContentType {
		return errors.New(`"content_type" doesn't expected to be updated`)
	}
	return nil
}

func srResourceModelUpdate(ctx context.Context, session *xenapi.Session, ref xenapi.SRRef, data srResourceModel) error {
	err := xenapi.SR.SetNameLabel(session, ref, data.NameLabel.ValueString())
	if err != nil {
		return errors.New(err.Error())
	}
	err = xenapi.SR.SetNameDescription(session, ref, data.NameDescription.ValueString())
	if err != nil {
		return errors.New(err.Error())
	}
	smConfig := make(map[string]string)
	diags := data.SmConfig.ElementsAs(ctx, &smConfig, false)
	if diags.HasError() {
		return errors.New("unable to access SR SM config data")
	}
	err = xenapi.SR.SetSmConfig(session, ref, smConfig)
	if err != nil {
		return errors.New(err.Error())
	}
	return nil
}

func unplugPBDs(session *xenapi.Session, pbdRefs []xenapi.PBDRef) error {
	if len(pbdRefs) == 0 {
		return nil
	}

	var allPBDRefsToNonCoordinator []xenapi.PBDRef
	var allPBDRefsToCoordinator []xenapi.PBDRef

	coordinatorRef, _, err := getCoordinatorRef(session)
	if err != nil {
		return err
	}
	// Need to run Unplug for the coordinator last
	for _, pbdRef := range pbdRefs {
		pbdRecord, err := xenapi.PBD.GetRecord(session, pbdRef)
		if err != nil {
			return errors.New(err.Error())
		}
		if pbdRecord.CurrentlyAttached {
			if string(pbdRecord.Host) != "OpaqueRef:NULL" && pbdRecord.Host == coordinatorRef {
				allPBDRefsToCoordinator = append(allPBDRefsToCoordinator, pbdRef)
			} else {
				allPBDRefsToNonCoordinator = append(allPBDRefsToNonCoordinator, pbdRef)
			}
		}
	}

	var allPBDRefs []xenapi.PBDRef
	allPBDRefs = append(allPBDRefs, allPBDRefsToNonCoordinator...)
	allPBDRefs = append(allPBDRefs, allPBDRefsToCoordinator...)
	for _, pbdRef := range allPBDRefs {
		err = xenapi.PBD.Unplug(session, pbdRef)
		if err != nil {
			return errors.New(err.Error())
		}
	}

	return nil
}

func cleanupSRResource(session *xenapi.Session, ref xenapi.SRRef) error {
	pbdRefs, err := xenapi.SR.GetPBDs(session, ref)
	if err != nil {
		return errors.New(err.Error())
	}
	err = unplugPBDs(session, pbdRefs)
	if err != nil {
		return err
	}
	err = xenapi.SR.Forget(session, ref)
	if err != nil {
		return errors.New(err.Error())
	}
	return nil
}

func createSRResource(session *xenapi.Session, params srCreateParams) (xenapi.SRRef, error) {
	var srRef xenapi.SRRef
	// Create secret for password
	var secretRef xenapi.SecretRef
	keys := []string{"cifspassword", "password", "chappassword"}
	if params.DeviceConfig != nil {
		for _, key := range keys {
			value, exists := params.DeviceConfig[key]
			if exists {
				delete(params.DeviceConfig, key)
				secretRecord := xenapi.SecretRecord{Value: value}
				secretRef, err := xenapi.Secret.Create(session, secretRecord)
				if err != nil {
					return srRef, errors.New(err.Error())
				}
				secretUUID, err := xenapi.Secret.GetUUID(session, secretRef)
				if err != nil {
					return srRef, errors.New(err.Error())
				}
				params.DeviceConfig[key+"_secret"] = secretUUID
				break
			}
		}
	}
	// Create SR
	srRef, err := xenapi.SR.Create(session, params.Host, params.DeviceConfig, params.PhysicalSize, params.NameLabel, params.NameDescription, params.TypeKey, params.ContentType, params.Shared, params.SmConfig)
	if err != nil {
		errDestroy := xenapi.Secret.Destroy(session, secretRef)
		if errDestroy != nil {
			return srRef, errors.New(err.Error() + "\n" + errDestroy.Error())
		}
		return srRef, errors.New(err.Error())
	}
	// Checking that SR.Create actually succeeded
	pbdRefs, err := xenapi.SR.GetPBDs(session, srRef)
	if err != nil {
		return srRef, errors.New(err.Error())
	}
	for _, pbdRef := range pbdRefs {
		currentlyAttached, err := xenapi.PBD.GetCurrentlyAttached(session, pbdRef)
		if err != nil {
			return srRef, errors.New(err.Error())
		}
		if !currentlyAttached {
			err = xenapi.PBD.Plug(session, pbdRef)
			if err != nil {
				return srRef, errors.New(err.Error())
			}
		}
	}
	otherConfig, err := xenapi.SR.GetOtherConfig(session, srRef)
	if err != nil {
		return srRef, errors.New(err.Error())
	}
	otherConfig["auto-scan"] = "false"
	if params.ContentType == "iso" {
		otherConfig["auto-scan"] = "true"
	}
	err = xenapi.SR.SetOtherConfig(session, srRef, otherConfig)
	if err != nil {
		return srRef, errors.New(err.Error())
	}
	return srRef, nil
}

type nfsResourceModel struct {
	NameLabel       types.String `tfsdk:"name_label"`
	NameDescription types.String `tfsdk:"name_description"`
	Type            types.String `tfsdk:"type"`
	StorageLocation types.String `tfsdk:"storage_location"`
	Version         types.String `tfsdk:"version"`
	AdvancedOptions types.String `tfsdk:"advanced_options"`
	UUID            types.String `tfsdk:"uuid"`
	ID              types.String `tfsdk:"id"`
}

func getNFSCreateParams(session *xenapi.Session, data nfsResourceModel) (srCreateParams, error) {
	var params srCreateParams
	coordinatorRef, _, err := getCoordinatorRef(session)
	if err != nil {
		return params, err
	}
	params.Host = coordinatorRef
	params.TypeKey = data.Type.ValueString()
	deviceConfig := make(map[string]string)
	storageLocation := strings.Split(data.StorageLocation.ValueString(), ":")
	if params.TypeKey == "iso" {
		params.ContentType = "iso"
		deviceConfig["location"] = strings.TrimSpace(data.StorageLocation.ValueString())
		deviceConfig["type"] = "nfs_iso"
	} else {
		deviceConfig["server"] = strings.TrimSpace(storageLocation[0])
		deviceConfig["serverpath"] = strings.TrimSpace(strings.Join(storageLocation[1:], ":"))
	}
	deviceConfig["options"] = data.AdvancedOptions.ValueString()
	deviceConfig["nfsversion"] = data.Version.ValueString()
	params.DeviceConfig = deviceConfig
	params.NameLabel = data.NameLabel.ValueString()
	params.NameDescription = data.NameDescription.ValueString()
	params.Shared = true
	params.SmConfig = make(map[string]string)

	return params, nil
}

func updateNFSResourceModel(srRecord xenapi.SRRecord, pbdRecord xenapi.PBDRecord, data *nfsResourceModel) error {
	data.NameLabel = types.StringValue(srRecord.NameLabel)
	if srRecord.Type == "iso" {
		location, ok := pbdRecord.DeviceConfig["location"]
		if !ok {
			return errors.New(`unable to find "location" in PBD device config`)
		}
		data.StorageLocation = types.StringValue(location)
	} else {
		server, ok := pbdRecord.DeviceConfig["server"]
		if !ok {
			return errors.New(`unable to find "server" in PBD device config`)
		}
		serverPath, ok := pbdRecord.DeviceConfig["serverpath"]
		if !ok {
			return errors.New(`unable to find "serverpath" in PBD device config`)
		}
		data.StorageLocation = types.StringValue(server + ":" + serverPath)
	}
	nfsVersion, ok := pbdRecord.DeviceConfig["nfsversion"]
	if !ok {
		return errors.New(`unable to find "nfsversion" in PBD device config`)
	}
	data.Version = types.StringValue(nfsVersion)
	err := updateNFSResourceModelComputed(srRecord, pbdRecord, data)

	return err
}

func updateNFSResourceModelComputed(srRecord xenapi.SRRecord, pbdRecord xenapi.PBDRecord, data *nfsResourceModel) error {
	data.UUID = types.StringValue(srRecord.UUID)
	data.ID = types.StringValue(srRecord.UUID)
	data.NameDescription = types.StringValue(srRecord.NameDescription)
	data.Type = types.StringValue(srRecord.Type)
	advancedOptions, ok := pbdRecord.DeviceConfig["options"]
	if !ok {
		data.AdvancedOptions = types.StringValue("")
	}
	data.AdvancedOptions = types.StringValue(advancedOptions)

	return nil
}

func nfsResourceModelUpdateCheck(data nfsResourceModel, dataState nfsResourceModel) error {
	if data.Type != dataState.Type {
		return errors.New(`"type" doesn't expected to be updated`)
	}
	if strings.TrimSpace(data.StorageLocation.ValueString()) != strings.TrimSpace(dataState.StorageLocation.ValueString()) {
		return errors.New(`"storage_location" doesn't expected to be updated`)
	}
	if data.Version != dataState.Version {
		return errors.New(`"version" doesn't expected to be updated`)
	}
	if data.AdvancedOptions != dataState.AdvancedOptions {
		return errors.New(`"advanced_options" doesn't expected to be updated`)
	}
	return nil
}

func nfsResourceModelUpdate(session *xenapi.Session, ref xenapi.SRRef, data nfsResourceModel) error {
	err := xenapi.SR.SetNameLabel(session, ref, data.NameLabel.ValueString())
	if err != nil {
		return errors.New(err.Error())
	}
	err = xenapi.SR.SetNameDescription(session, ref, data.NameDescription.ValueString())
	if err != nil {
		return errors.New(err.Error())
	}

	return nil
}

type smbResourceModel struct {
	NameLabel       types.String `tfsdk:"name_label"`
	NameDescription types.String `tfsdk:"name_description"`
	Type            types.String `tfsdk:"type"`
	StorageLocation types.String `tfsdk:"storage_location"`
	Username        types.String `tfsdk:"username"`
	Password        types.String `tfsdk:"password"`
	UUID            types.String `tfsdk:"uuid"`
	ID              types.String `tfsdk:"id"`
}

func getSMBCreateParams(session *xenapi.Session, data smbResourceModel) (srCreateParams, error) {
	var params srCreateParams
	coordinatorRef, _, err := getCoordinatorRef(session)
	if err != nil {
		return params, err
	}
	params.Host = coordinatorRef
	deviceConfig := make(map[string]string)
	username := strings.TrimSpace(data.Username.ValueString())
	password := strings.TrimSpace(data.Password.ValueString())
	storageLocation := strings.Split(strings.TrimSpace(data.StorageLocation.ValueString()), ":")
	params.TypeKey = data.Type.ValueString()
	if params.TypeKey == "iso" {
		params.ContentType = "iso"
		deviceConfig["location"] = strings.ReplaceAll(storageLocation[0], "\\", "/")
		bits := strings.Split(deviceConfig["location"], "/")
		if len(bits) > 4 {
			deviceConfig["location"] = "//" + bits[2] + "/" + bits[3]
			deviceConfig["iso_path"] = "/" + strings.Join(bits[4:], "/")
		}
		deviceConfig["type"] = "cifs"
		if username != "" {
			deviceConfig["username"] = username
		}
		if password != "" {
			deviceConfig["cifspassword"] = password
		}
	} else {
		deviceConfig["server"] = storageLocation[0]
		if len(storageLocation) > 1 {
			deviceConfig["serverpath"] = storageLocation[1]
		}
		if username != "" {
			deviceConfig["username"] = username
		}
		if password != "" {
			deviceConfig["password"] = password
		}
	}
	params.DeviceConfig = deviceConfig
	params.NameLabel = data.NameLabel.ValueString()
	params.NameDescription = data.NameDescription.ValueString()
	params.Shared = true
	params.SmConfig = make(map[string]string)

	return params, nil
}

func updateSMBResourceModel(srRecord xenapi.SRRecord, pbdRecord xenapi.PBDRecord, data *smbResourceModel) error {
	data.NameLabel = types.StringValue(srRecord.NameLabel)
	if srRecord.Type == "iso" {
		location, ok := pbdRecord.DeviceConfig["location"]
		if !ok {
			return errors.New(`unable to find "location" in PBD device config`)
		}
		isoPath, ok := pbdRecord.DeviceConfig["iso_path"]
		if ok && isoPath != "" {
			location += isoPath
		}
		location = strings.ReplaceAll(location, "/", "\\")
		data.StorageLocation = types.StringValue(location)
	} else {
		server, ok := pbdRecord.DeviceConfig["server"]
		if !ok {
			return errors.New(`unable to find "server" in PBD device config`)
		}
		data.StorageLocation = types.StringValue(server)
		serverPath, ok := pbdRecord.DeviceConfig["serverpath"]
		if ok && serverPath != "" {
			data.StorageLocation = types.StringValue(server + ":" + serverPath)
		}
	}
	err := updateSMBResourceModelComputed(srRecord, data)

	return err
}

func updateSMBResourceModelComputed(srRecord xenapi.SRRecord, data *smbResourceModel) error {
	data.UUID = types.StringValue(srRecord.UUID)
	data.ID = types.StringValue(srRecord.UUID)
	data.NameDescription = types.StringValue(srRecord.NameDescription)
	data.Type = types.StringValue(srRecord.Type)

	return nil
}

func smbResourceModelUpdateCheck(data smbResourceModel, dataState smbResourceModel) error {
	if data.Type != dataState.Type {
		return errors.New(`"type" doesn't expected to be updated`)
	}
	if strings.TrimSpace(data.StorageLocation.ValueString()) != strings.TrimSpace(dataState.StorageLocation.ValueString()) {
		return errors.New(`"storage_location" doesn't expected to be updated`)
	}
	return nil
}

func smbResourceModelUpdate(session *xenapi.Session, ref xenapi.SRRef, data smbResourceModel) error {
	err := xenapi.SR.SetNameLabel(session, ref, data.NameLabel.ValueString())
	if err != nil {
		return errors.New(err.Error())
	}
	err = xenapi.SR.SetNameDescription(session, ref, data.NameDescription.ValueString())
	if err != nil {
		return errors.New(err.Error())
	}

	return nil
}
