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

data "xenserver_network" "network" {}

resource "xenserver_vdi" "vdi" {
  name_label   = "Import VDI"
  sr_uuid      = data.xenserver_sr.sr.data_items[0].uuid
  raw_vdi_path = "/tmp/livecd.ubuntu-cpc.azure.vhd"
}

resource "xenserver_vm" "linux_vm" {
  name_label       = "Linux VM"
  template_name    = "Ubuntu Jammy Jellyfish 22.04"
  static_mem_max   = 4 * 1024 * 1024 * 1024
  vcpus            = 4
  boot_mode        = "uefi"

  hard_drive = [
    {
      vdi_uuid = xenserver_vdi.vdi.uuid,
      bootable = true,
      mode     = "RW"
    },
  ]

  network_interface = [
    {
      network_uuid = data.xenserver_network.network.data_items[0].uuid,
      device       = "0"
    },
  ]
}