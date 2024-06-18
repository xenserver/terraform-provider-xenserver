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

data "xenserver_pif" "pif_data" {
  device     = "eth0"
  management = true
}

output "pif_data_out" {
  value = data.xenserver_pif.pif_data
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
  value = xenserver_network.network.id
}
