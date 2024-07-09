// Create a VDI with a new SR
resource "xenserver_sr_nfs" "nfs" {
  name_label       = "Test NFS SR"
  name_description = "A test NFS storage repository"
  version          = "3"
  storage_location = "10.70.58.9:/xenrtnfs"
}

resource "xenserver_vdi" "vdi" {
  name_label       = "Test VDI"
  name_description = "A test VDI on NFS SR"
  sr_uuid          = xenserver_sr_nfs.nfs.uuid
  virtual_size     = 1 * 1024 * 1024 * 1024
  sharable         = true
  other_config = {
    "flag" = "1"
  }
}

output "vdi_out" {
  value = xenserver_vdi.vdi
}

// Create a VDI with an exist SR
data "xenserver_sr" "sr" {
  name_label = "Local storage"
}

resource "xenserver_vdi" "vdi" {
  name_label       = "Test VDI"
  name_description = "A test VDI on Local storage"
  sr_uuid          = data.xenserver_sr.sr.data_items[0].uuid
  virtual_size     = 1 * 1024 * 1024 * 1024
  read_only        = true
  type             = "system"
}

output "vdi_out" {
  value = xenserver_vdi.vdi
}
