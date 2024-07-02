package xenserver

import (
	"context"
	"errors"
	"xenapi"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func createVBD(vdiUUID string, vmRef xenapi.VMRef, session *xenapi.Session) (xenapi.VBDRef, error) {
	vdiRef, err := xenapi.VDI.GetByUUID(session, vdiUUID)
	if err != nil {
		return "", errors.New("unable to get VDI ref, vdi UUID: " + vdiUUID)
	}

	userDevices, _ := xenapi.VM.GetAllowedVBDDevices(session, vmRef)
	if len(userDevices) == 0 {
		return "", errors.New("No available devices to attach to vm " + string(vmRef))
	}

	vbdRecord := xenapi.VBDRecord{
		VM:         vmRef,
		VDI:        vdiRef,
		Type:       "Disk",
		Mode:       "RW",
		Bootable:   false,
		Empty:      false,
		Userdevice: userDevices[0],
	}

	vbdRef, err := xenapi.VBD.Create(session, vbdRecord)
	if err != nil {
		return "", errors.New("unable to create VBD, vdi UUID: " + vdiUUID)
	}

	// plug VBDs if VM is running
	vmPowerState, err := xenapi.VM.GetPowerState(session, vmRef)
	if err != nil {
		return "", errors.New("unable to get VM power state, vm ref: " + string(vmRef))
	}

	if vmPowerState == xenapi.VMPowerStateRunning {
		err = xenapi.VBD.Plug(session, vbdRef)
		if err != nil {
			return "", errors.New("unable to plug VBD, vdi UUID: " + vdiUUID)
		}
	}

	return vbdRef, nil
}

func createVBDs(ctx context.Context, vdiUUIDs types.List, vmRef xenapi.VMRef, session *xenapi.Session) ([]xenapi.VBDRef, error) {
	elements := make([]string, 0, len(vdiUUIDs.Elements()))
	diags := vdiUUIDs.ElementsAs(ctx, &elements, false)
	if diags.HasError() {
		return nil, errors.New("unable to get VDI UUIDs in plan data hard drive attributes")
	}

	var vbdRefs []xenapi.VBDRef
	for _, vdiUUID := range elements {
		vbdRef, err := createVBD(vdiUUID, vmRef, session)
		if err != nil {
			return nil, err
		}
		vbdRefs = append(vbdRefs, vbdRef)
	}
	return vbdRefs, nil
}

func updateVBDs(planVDIUUIDs []string, stateVDIUUIDs []string, vmRef xenapi.VMRef, session *xenapi.Session) error {
	var err error

	planVDIsMap := make(map[string]bool, len(planVDIUUIDs))
	for _, vdiUUID := range planVDIUUIDs {
		planVDIsMap[vdiUUID] = true
	}

	stateVDIsMap := make(map[string]bool, len(stateVDIUUIDs))
	for _, vdiUUID := range stateVDIUUIDs {
		stateVDIsMap[vdiUUID] = true
	}

	// Create VBDs that are in plan but not in state
	for vdiUUID := range planVDIsMap {
		if _, ok := stateVDIsMap[vdiUUID]; !ok {
			_, err = createVBD(vdiUUID, vmRef, session)
			if err != nil {
				return err
			}
		}
	}

	// Destroy VBDs that are not in plan
	for vdiUUID := range stateVDIsMap {
		if _, ok := planVDIsMap[vdiUUID]; !ok {
			err = removeVBDbyVDIUUID(session, vdiUUID)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func removeVBDbyVDIUUID(session *xenapi.Session, vdiUUID string) error {
	VDIRef, err := xenapi.VDI.GetByUUID(session, vdiUUID)
	if err != nil {
		return errors.New("unable to get VDI ref, vdi UUID: " + vdiUUID)
	}

	VBDRefs, err := xenapi.VDI.GetVBDs(session, VDIRef)
	if err != nil {
		return errors.New("unable to get VBDs for VDI, vdi UUID: " + vdiUUID)
	}

	for _, vbdRef := range VBDRefs {
		err = xenapi.VBD.Destroy(session, vbdRef)
		if err != nil {
			return errors.New("unable to destroy VBD, vbd ref: " + string(vbdRef))
		}
	}
	return nil
}
