package xenserver

import (
	"errors"

	"xenapi"
)

func getBlobUUIDsMap(session *xenapi.Session, oldMap map[string]xenapi.BlobRef) (map[string]string, error) {
	// map[string]BlobRef to map[string]string
	newMap := make(map[string]string)
	for key, ref := range oldMap {
		uuid, err := getUUIDFromBlobRef(session, ref)
		if err != nil {
			return newMap, err
		}
		newMap[key] = uuid
	}
	return newMap, nil
}

func getUUIDFromBlobRef(session *xenapi.Session, ref xenapi.BlobRef) (string, error) {
	if string(ref) != "" && string(ref) != "OpaqueRef:NULL" {
		uuid, err := xenapi.Blob.GetUUID(session, ref)
		if err != nil {
			return uuid, errors.New("unable to get blob UUID. " + err.Error())
		}
		return uuid, nil
	}
	return "", nil
}

func getBondUUIDs(session *xenapi.Session, refs []xenapi.BondRef) ([]string, error) {
	uuids := []string{}
	for _, ref := range refs {
		uuid, err := getUUIDFromBondRef(session, ref)
		if err != nil {
			return uuids, err
		}
		if uuid != "" {
			uuids = append(uuids, uuid)
		}
	}
	return uuids, nil
}

func getUUIDFromBondRef(session *xenapi.Session, ref xenapi.BondRef) (string, error) {
	if string(ref) != "" && string(ref) != "OpaqueRef:NULL" {
		uuid, err := xenapi.Bond.GetUUID(session, ref)
		if err != nil {
			return uuid, errors.New("unable to get bond UUID. " + err.Error())
		}
		return uuid, nil
	}
	return "", nil
}

func getConsoleUUIDs(session *xenapi.Session, refs []xenapi.ConsoleRef) ([]string, error) {
	uuids := []string{}
	for _, ref := range refs {
		uuid, err := getUUIDFromConsoleRef(session, ref)
		if err != nil {
			return uuids, err
		}
		if uuid != "" {
			uuids = append(uuids, uuid)
		}
	}
	return uuids, nil
}

func getUUIDFromConsoleRef(session *xenapi.Session, ref xenapi.ConsoleRef) (string, error) {
	if string(ref) != "" && string(ref) != "OpaqueRef:NULL" {
		uuid, err := xenapi.Console.GetUUID(session, ref)
		if err != nil {
			return uuid, errors.New("unable to get console UUID. " + err.Error())
		}
		return uuid, nil
	}
	return "", nil
}

func getCrashdumpUUIDs(session *xenapi.Session, refs []xenapi.CrashdumpRef) ([]string, error) {
	uuids := []string{}
	for _, ref := range refs {
		uuid, err := getUUIDFromCrashdumpRef(session, ref)
		if err != nil {
			return uuids, err
		}
		if uuid != "" {
			uuids = append(uuids, uuid)
		}
	}
	return uuids, nil
}

func getUUIDFromCrashdumpRef(session *xenapi.Session, ref xenapi.CrashdumpRef) (string, error) {
	if string(ref) != "" && string(ref) != "OpaqueRef:NULL" {
		uuid, err := xenapi.Crashdump.GetUUID(session, ref)
		if err != nil {
			return uuid, errors.New("unable to get crash dump UUID. " + err.Error())
		}
		return uuid, nil
	}
	return "", nil
}

func getUUIDFromDRTaskRef(session *xenapi.Session, ref xenapi.DRTaskRef) (string, error) {
	if string(ref) != "" && string(ref) != "OpaqueRef:NULL" {
		uuid, err := xenapi.DRTask.GetUUID(session, ref)
		if err != nil {
			return uuid, errors.New("unable to get DR task UUID. " + err.Error())
		}
		return uuid, nil
	}
	return "", nil
}

func getUUIDFromHostRef(session *xenapi.Session, ref xenapi.HostRef) (string, error) {
	if string(ref) != "" && string(ref) != "OpaqueRef:NULL" {
		uuid, err := xenapi.Host.GetUUID(session, ref)
		if err != nil {
			return uuid, errors.New("unable to get host UUID. " + err.Error())
		}
		return uuid, nil
	}
	return "", nil
}

func getUUIDFromNetworkRef(session *xenapi.Session, ref xenapi.NetworkRef) (string, error) {
	if string(ref) != "" && string(ref) != "OpaqueRef:NULL" {
		uuid, err := xenapi.Network.GetUUID(session, ref)
		if err != nil {
			return uuid, errors.New("unable to get network UUID. " + err.Error())
		}
		return uuid, nil
	}
	return "", nil
}

func getNetworkSriovUUIDs(session *xenapi.Session, refs []xenapi.NetworkSriovRef) ([]string, error) {
	uuids := []string{}
	for _, ref := range refs {
		uuid, err := getUUIDFromNetworkSriovRef(session, ref)
		if err != nil {
			return uuids, err
		}
		if uuid != "" {
			uuids = append(uuids, uuid)
		}
	}
	return uuids, nil
}

func getUUIDFromNetworkSriovRef(session *xenapi.Session, ref xenapi.NetworkSriovRef) (string, error) {
	if string(ref) != "" && string(ref) != "OpaqueRef:NULL" {
		uuid, err := xenapi.NetworkSriov.GetUUID(session, ref)
		if err != nil {
			return uuid, errors.New("unable to get network sr-iov UUID. " + err.Error())
		}
		return uuid, nil
	}
	return "", nil
}

func getPBDUUIDs(session *xenapi.Session, refs []xenapi.PBDRef) ([]string, error) {
	uuids := []string{}
	for _, ref := range refs {
		uuid, err := getUUIDFromPBDRef(session, ref)
		if err != nil {
			return uuids, err
		}
		if uuid != "" {
			uuids = append(uuids, uuid)
		}
	}
	return uuids, nil
}

func getUUIDFromPBDRef(session *xenapi.Session, ref xenapi.PBDRef) (string, error) {
	if string(ref) != "" && string(ref) != "OpaqueRef:NULL" {
		uuid, err := xenapi.PBD.GetUUID(session, ref)
		if err != nil {
			return uuid, errors.New("unable to get PBD UUID. " + err.Error())
		}
		return uuid, nil
	}
	return "", nil
}

func getPCIUUIDs(session *xenapi.Session, refs []xenapi.PCIRef) ([]string, error) {
	uuids := []string{}
	for _, ref := range refs {
		uuid, err := getUUIDFromPCIRef(session, ref)
		if err != nil {
			return uuids, err
		}
		if uuid != "" {
			uuids = append(uuids, uuid)
		}
	}
	return uuids, nil
}

func getUUIDFromPCIRef(session *xenapi.Session, ref xenapi.PCIRef) (string, error) {
	if string(ref) != "" && string(ref) != "OpaqueRef:NULL" {
		uuid, err := xenapi.PCI.GetUUID(session, ref)
		if err != nil {
			return uuid, errors.New("unable to get PCI UUID. " + err.Error())
		}
		return uuid, nil
	}
	return "", nil
}

func getPIFUUIDs(session *xenapi.Session, refs []xenapi.PIFRef) ([]string, error) {
	uuids := []string{}
	for _, ref := range refs {
		uuid, err := getUUIDFromPIFRef(session, ref)
		if err != nil {
			return uuids, err
		}
		if uuid != "" {
			uuids = append(uuids, uuid)
		}
	}
	return uuids, nil
}

func getUUIDFromPIFRef(session *xenapi.Session, ref xenapi.PIFRef) (string, error) {
	if string(ref) != "" && string(ref) != "OpaqueRef:NULL" {
		uuid, err := xenapi.PIF.GetUUID(session, ref)
		if err != nil {
			return uuid, errors.New("unable to get PIF UUID. " + err.Error())
		}
		return uuid, nil
	}
	return "", nil
}

func getUUIDFromSRRef(session *xenapi.Session, ref xenapi.SRRef) (string, error) {
	if string(ref) != "" && string(ref) != "OpaqueRef:NULL" {
		uuid, err := xenapi.SR.GetUUID(session, ref)
		if err != nil {
			return uuid, errors.New("unable to get SR UUID. " + err.Error())
		}
		return uuid, nil
	}
	return "", nil
}

func getTunnelUUIDs(session *xenapi.Session, refs []xenapi.TunnelRef) ([]string, error) {
	uuids := []string{}
	for _, ref := range refs {
		uuid, err := getUUIDFromTunnelRef(session, ref)
		if err != nil {
			return uuids, err
		}
		if uuid != "" {
			uuids = append(uuids, uuid)
		}
	}
	return uuids, nil
}

func getUUIDFromTunnelRef(session *xenapi.Session, ref xenapi.TunnelRef) (string, error) {
	if string(ref) != "" && string(ref) != "OpaqueRef:NULL" {
		uuid, err := xenapi.Tunnel.GetUUID(session, ref)
		if err != nil {
			return uuid, errors.New("unable to get tunnel UUID. " + err.Error())
		}
		return uuid, nil
	}
	return "", nil
}

func getVBDUUIDs(session *xenapi.Session, refs []xenapi.VBDRef) ([]string, error) {
	uuids := []string{}
	for _, ref := range refs {
		uuid, err := getUUIDFromVBDRef(session, ref)
		if err != nil {
			return uuids, err
		}
		if uuid != "" {
			uuids = append(uuids, uuid)
		}
	}
	return uuids, nil
}

func getUUIDFromVBDRef(session *xenapi.Session, ref xenapi.VBDRef) (string, error) {
	if string(ref) != "" && string(ref) != "OpaqueRef:NULL" {
		uuid, err := xenapi.VBD.GetUUID(session, ref)
		if err != nil {
			return uuid, errors.New("unable to get VBD UUID. " + err.Error())
		}
		return uuid, nil
	}
	return "", nil
}

func getVDIUUIDs(session *xenapi.Session, refs []xenapi.VDIRef) ([]string, error) {
	uuids := []string{}
	for _, ref := range refs {
		uuid, err := getUUIDFromVDIRef(session, ref)
		if err != nil {
			return uuids, err
		}
		if uuid != "" {
			uuids = append(uuids, uuid)
		}
	}
	return uuids, nil
}

func getUUIDFromVDIRef(session *xenapi.Session, ref xenapi.VDIRef) (string, error) {
	if string(ref) != "" && string(ref) != "OpaqueRef:NULL" {
		uuid, err := xenapi.VDI.GetUUID(session, ref)
		if err != nil {
			return uuid, errors.New("unable to get VDI UUID. " + err.Error())
		}
		return uuid, nil
	}
	return "", nil
}

func getVGPUUUIDs(session *xenapi.Session, refs []xenapi.VGPURef) ([]string, error) {
	uuids := []string{}
	for _, ref := range refs {
		uuid, err := getUUIDFromVGPURef(session, ref)
		if err != nil {
			return uuids, err
		}
		if uuid != "" {
			uuids = append(uuids, uuid)
		}
	}
	return uuids, nil
}

func getUUIDFromVGPURef(session *xenapi.Session, ref xenapi.VGPURef) (string, error) {
	if string(ref) != "" && string(ref) != "OpaqueRef:NULL" {
		uuid, err := xenapi.VGPU.GetUUID(session, ref)
		if err != nil {
			return uuid, errors.New("unable to get vGPU UUID. " + err.Error())
		}
		return uuid, nil
	}
	return "", nil
}

func getVIFUUIDsMap(session *xenapi.Session, oldMap map[xenapi.VIFRef]string) (map[string]string, error) {
	// map[VIFRef]string to map[string]string
	newMap := make(map[string]string)
	for ref, value := range oldMap {
		uuid, err := getUUIDFromVIFRef(session, ref)
		if err != nil {
			return newMap, err
		}
		newMap[uuid] = value
	}
	return newMap, nil
}

func getVIFUUIDs(session *xenapi.Session, refs []xenapi.VIFRef) ([]string, error) {
	uuids := []string{}
	for _, ref := range refs {
		uuid, err := getUUIDFromVIFRef(session, ref)
		if err != nil {
			return uuids, err
		}
		if uuid != "" {
			uuids = append(uuids, uuid)
		}
	}
	return uuids, nil
}

func getUUIDFromVIFRef(session *xenapi.Session, ref xenapi.VIFRef) (string, error) {
	if string(ref) != "" && string(ref) != "OpaqueRef:NULL" {
		uuid, err := xenapi.VIF.GetUUID(session, ref)
		if err != nil {
			return uuid, errors.New("unable to get VIF UUID. " + err.Error())
		}
		return uuid, nil
	}
	return "", nil
}

func getVLANUUIDs(session *xenapi.Session, refs []xenapi.VLANRef) ([]string, error) {
	uuids := []string{}
	for _, ref := range refs {
		uuid, err := getUUIDFromVLANRef(session, ref)
		if err != nil {
			return uuids, err
		}
		if uuid != "" {
			uuids = append(uuids, uuid)
		}
	}
	return uuids, nil
}

func getUUIDFromVLANRef(session *xenapi.Session, ref xenapi.VLANRef) (string, error) {
	if string(ref) != "" && string(ref) != "OpaqueRef:NULL" {
		uuid, err := xenapi.VLAN.GetUUID(session, ref)
		if err != nil {
			return uuid, errors.New("unable to get vlan UUID. " + err.Error())
		}
		return uuid, nil
	}
	return "", nil
}

func getVMUUIDs(session *xenapi.Session, refs []xenapi.VMRef, excludeRef xenapi.VMRef) ([]string, error) {
	uuids := []string{}
	for _, ref := range refs {
		if ref != excludeRef {
			uuid, err := getUUIDFromVMRef(session, ref)
			if err != nil {
				return uuids, err
			}
			if uuid != "" {
				uuids = append(uuids, uuid)
			}
		}
	}
	return uuids, nil
}

func getUUIDFromVMRef(session *xenapi.Session, ref xenapi.VMRef) (string, error) {
	if string(ref) != "" && string(ref) != "OpaqueRef:NULL" {
		uuid, err := xenapi.VM.GetUUID(session, ref)
		if err != nil {
			return uuid, errors.New("unable to get VM UUID. " + err.Error())
		}
		return uuid, nil
	}
	return "", nil
}

func getUUIDFromVMApplianceRef(session *xenapi.Session, ref xenapi.VMApplianceRef) (string, error) {
	if string(ref) != "" && string(ref) != "OpaqueRef:NULL" {
		uuid, err := xenapi.VMAppliance.GetUUID(session, ref)
		if err != nil {
			return uuid, errors.New("unable to get VM appliance UUID. " + err.Error())
		}
		return uuid, nil
	}
	return "", nil
}

func getVMGroupUUIDs(session *xenapi.Session, refs []xenapi.VMGroupRef) ([]string, error) {
	uuids := []string{}
	for _, ref := range refs {
		uuid, err := getUUIDFromVMGroupRef(session, ref)
		if err != nil {
			return uuids, err
		}
		if uuid != "" {
			uuids = append(uuids, uuid)
		}
	}
	return uuids, nil
}

func getUUIDFromVMGroupRef(session *xenapi.Session, ref xenapi.VMGroupRef) (string, error) {
	if string(ref) != "" && string(ref) != "OpaqueRef:NULL" {
		uuid, err := xenapi.VMGroup.GetUUID(session, ref)
		if err != nil {
			return uuid, errors.New("unable to get VM group UUID. " + err.Error())
		}
		return uuid, nil
	}
	return "", nil
}

func getUUIDFromVMPPRef(session *xenapi.Session, ref xenapi.VMPPRef) (string, error) {
	if string(ref) != "" && string(ref) != "OpaqueRef:NULL" {
		uuid, err := xenapi.VMPP.GetUUID(session, ref)
		if err != nil {
			return uuid, errors.New("unable to get VMPP UUID. " + err.Error())
		}
		return uuid, nil
	}
	return "", nil
}

func getUUIDFromVMSSRef(session *xenapi.Session, ref xenapi.VMSSRef) (string, error) {
	if string(ref) != "" && string(ref) != "OpaqueRef:NULL" {
		uuid, err := xenapi.VMSS.GetUUID(session, ref)
		if err != nil {
			return uuid, errors.New("unable to get VMSS UUID. " + err.Error())
		}
		return uuid, nil
	}
	return "", nil
}

func getUUIDFromVMMetricsRef(session *xenapi.Session, ref xenapi.VMMetricsRef) (string, error) {
	if string(ref) != "" && string(ref) != "OpaqueRef:NULL" {
		uuid, err := xenapi.VMMetrics.GetUUID(session, ref)
		if err != nil {
			return uuid, errors.New("unable to get VM metrics UUID. " + err.Error())
		}
		return uuid, nil
	}
	return "", nil
}

func getUUIDFromVMGuestMetricsRef(session *xenapi.Session, ref xenapi.VMGuestMetricsRef) (string, error) {
	if string(ref) != "" && string(ref) != "OpaqueRef:NULL" {
		uuid, err := xenapi.VMGuestMetrics.GetUUID(session, ref)
		if err != nil {
			return uuid, errors.New("unable to get VM guest metrics UUID. " + err.Error())
		}
		return uuid, nil
	}
	return "", nil
}

func getVTPMUUIDs(session *xenapi.Session, refs []xenapi.VTPMRef) ([]string, error) {
	uuids := []string{}
	for _, ref := range refs {
		uuid, err := getUUIDFromVTPMRef(session, ref)
		if err != nil {
			return uuids, err
		}
		if uuid != "" {
			uuids = append(uuids, uuid)
		}
	}
	return uuids, nil
}

func getUUIDFromVTPMRef(session *xenapi.Session, ref xenapi.VTPMRef) (string, error) {
	if string(ref) != "" && string(ref) != "OpaqueRef:NULL" {
		uuid, err := xenapi.VTPM.GetUUID(session, ref)
		if err != nil {
			return uuid, errors.New("unable to get VTPM UUID. " + err.Error())
		}
		return uuid, nil
	}
	return "", nil
}

func getVUSBUUIDs(session *xenapi.Session, refs []xenapi.VUSBRef) ([]string, error) {
	uuids := []string{}
	for _, ref := range refs {
		uuid, err := getUUIDFromVUSBRef(session, ref)
		if err != nil {
			return uuids, err
		}
		if uuid != "" {
			uuids = append(uuids, uuid)
		}
	}
	return uuids, nil
}

func getUUIDFromVUSBRef(session *xenapi.Session, ref xenapi.VUSBRef) (string, error) {
	if string(ref) != "" && string(ref) != "OpaqueRef:NULL" {
		uuid, err := xenapi.VUSB.GetUUID(session, ref)
		if err != nil {
			return uuid, errors.New("unable to get VUSB UUID. " + err.Error())
		}
		return uuid, nil
	}
	return "", nil
}
