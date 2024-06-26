locals {
  env_vars = { for tuple in regexall("export\\s*(\\S*)\\s*=\\s*(\\S*)\\s*", file("../../.env")) : tuple[0] => tuple[1] }
}

terraform {
  required_providers {
    xenserver = {
      source = "xenserver/xenserver"
    }
  }
}

provider "xenserver" {
  host     = local.env_vars["XENSERVER_HOST"]
  username = local.env_vars["XENSERVER_USERNAME"]
  password = local.env_vars["XENSERVER_PASSWORD"]
}

data "xenserver_vm" "vm_data" {
}

output "vm_data_out" {
  value = data.xenserver_vm.vm_data.data_items
}

data "xenserver_pif" "pif" {
  device     = "eth0"
  management = true
}

output "pif_data_out" {
  value = data.xenserver_pif.pif.data_items
}

resource "xenserver_vm" "vm" {
  name_label    = "Test CentOS VM"
  template_name = "CentOS 7"
  other_config = {
    flag = "1"
  }
}

output "vm_out" {
  value = xenserver_vm.vm
}

data "xenserver_sr" "sr" {
  name_label = "Local storage"
}

output "local_storage_output" {
  value = data.xenserver_sr.sr.data_items
}

resource "xenserver_sr" "local" {
  name_label = "Test Local SR"
  type       = "dummy"
  shared     = false
}

output "sr_local_out" {
  value = xenserver_sr.local
}

resource "xenserver_sr" "nfs" {
  name_label = "Test NFS SR"
  type       = "nfs"
  device_config = {
    serverpath = local.env_vars["NFS_SERVER_PATH"]
    server     = local.env_vars["NFS_SERVER"]
    nfsversion = "3"
  }
}

output "sr_nfs_out" {
  value = xenserver_sr.nfs
}

resource "xenserver_sr_nfs" "nfs_test" {
  name_label       = "NFS virtual disk storage"
  name_description = "A test NFS storage repository"
  version          = "3"
  storage_location = format("%s:%s", local.env_vars["NFS_SERVER"], local.env_vars["NFS_SERVER_PATH"])
}

output "nfs_test_out" {
  value = xenserver_sr_nfs.nfs_test
}

resource "xenserver_network" "network" {
  name_label       = "Network Object on Pool"
  name_description = "VM and Host objects that are attached to the Network object"
  mtu              = 1500
  // managed = true // This can be set on create and can't be updated
  other_config = {}
}

output "network_out" {
  value = xenserver_network.network
}

resource "xenserver_vdi" "vdi1" {
  name_label       = "Test VDI1"
  name_description = "A test VDI on NFS SR"
  sr_uuid          = xenserver_sr_nfs.nfs_test.id
  virtual_size     = 1 * 1024 * 1024 * 1024
  sharable         = true
  other_config = {
    "flag" = "1"
  }
}

output "vdi_out1" {
  value = xenserver_vdi.vdi1
}

resource "xenserver_vdi" "vdi2" {
  name_label       = "Test VDI2"
  name_description = "A test VDI on Local storage"
  sr_uuid          = data.xenserver_sr.sr.data_items[0].uuid
  virtual_size     = 1 * 1024 * 1024 * 1024
  read_only        = true
  type             = "system"
}

output "vdi_out2" {
  value = xenserver_vdi.vdi2
}

data "xenserver_network" "network" {
  name_label = "Pool-wide network associated with eth0"
}

output "network_output" {
  value = data.xenserver_network.network.data_items
}

data "xenserver_nic" "nic" {
  network_type = "vlan"
}

output "nic_output" {
  value = data.xenserver_nic.nic.data_items
}