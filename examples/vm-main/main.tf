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

data "xenserver_sr" "sr" {
  name_label = "Local storage"
}

resource "xenserver_vdi" "vdi1" {
  name_label   = "local-storage-vdi-1"
  sr_uuid      = data.xenserver_sr.sr.data_items[0].uuid
  virtual_size = 30 * 1024 * 1024 * 1024
}
resource "xenserver_vdi" "vdi2" {
  name_label   = "local-storage-vdi-2"
  sr_uuid      = data.xenserver_sr.sr.data_items[0].uuid
  virtual_size = 30 * 1024 * 1024 * 1024
}

data "xenserver_network" "network" {}

resource "xenserver_vm" "vm" {
  name_label       = "Windows VM"
  template_name    = "Windows 11"
  static_mem_max   = 4 * 1024 * 1024 * 1024
  vcpus            = 4
  cores_per_socket = 2
  cdrom            = "win11-x64_uefi.iso"
  boot_mode        = "uefi_security"
  boot_order       = "cdn"

  hard_drive = [
    {
      vdi_uuid = xenserver_vdi.vdi1.uuid,
      bootable = true,
      mode     = "RW"
    },
    {
      vdi_uuid = xenserver_vdi.vdi2.uuid,
      bootable = false,
      mode     = "RO"
    },
  ]

  network_interface = [
    {
      device       = "0"
      network_uuid = data.xenserver_network.network.data_items[0].uuid,
    },
    {
      device = "1"
      other_config = {
        ethtool-gso = "off"
      }
      mac          = "11:22:33:44:55:66"
      network_uuid = data.xenserver_network.network.data_items[1].uuid,
    },
  ]

  other_config = {
    "tf_created" = "true"
  }
}

output "vm_out" {
  value = xenserver_vm.vm
}

resource "xenserver_snapshot" "snapshot" {
  name_label = "A test snapshot with disk"
  vm_uuid    = xenserver_vm.vm.uuid
}

output "snapshot_out" {
  value = xenserver_snapshot.snapshot
}
