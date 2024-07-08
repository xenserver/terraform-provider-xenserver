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

resource "xenserver_vm" "vm" {
  name_label    = "A test virtual-machine"
  template_name = "Windows 11"
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
}

output "vm_out" {
  value = xenserver_vm.vm
}
