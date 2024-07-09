// snapshot from an exist running VM 
data "xenserver_vm" "vm_data" {
  name_label = "Test VM"
}

resource "xenserver_snapshot" "snapshot" {
  name_label  = "A test snapshot 1"
  vm_uuid     = data.xenserver_vm.vm_data.data_items[0].uuid
  with_memory = true
}

// snapshot from a new VM
data "xenserver_sr" "sr" {
  name_label = "Local storage"
}

resource "xenserver_vdi" "vdi1" {
  name_label   = "local-storage-vdi-1"
  sr_uuid      = data.xenserver_sr.sr.data_items[0].uuid
  virtual_size = 100 * 1024 * 1024 * 1024
}

resource "xenserver_vm" "vm" {
  name_label    = "A test virtual-machine"
  template_name = "Windows 11"
  hard_drive = [
    {
      vdi_uuid = xenserver_vdi.vdi1.id,
      bootable = true,
      mode     = "RW"
    },
  ]
}

resource "xenserver_snapshot" "snapshot" {
  name_label = "A test snapshot 2"
  vm_uuid    = xenserver_vm.vm.id
}
