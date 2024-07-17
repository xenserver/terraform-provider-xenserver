data "xenserver_sr" "sr" {
  name_label = "Local storage"
}

resource "xenserver_vdi" "vdi1" {
  name_label   = "local-storage-vdi-1"
  sr_uuid      = data.xenserver_sr.sr.data_items[0].uuid
  virtual_size = 100 * 1024 * 1024 * 1024
}
resource "xenserver_vdi" "vdi2" {
  name_label   = "local-storage-vdi-2"
  sr_uuid      = data.xenserver_sr.sr.data_items[0].uuid
  virtual_size = 100 * 1024 * 1024 * 1024
}

data "xenserver_network" "network" {}

resource "xenserver_vm" "vm" {
  name_label       = "A test virtual-machine"
  template_name    = "Windows 11"
  static_mem_max   = 4 * 1024 * 1024 * 1024
  vcpus            = 4
  cores_per_socket = 2

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
      network_uuid = data.xenserver_network.network.data_items[0].uuid,
      device       = "0"
    },
    {
      other_config = {
        ethtool-gso = "off"
      }
      mtu          = 1600
      device       = "0"
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