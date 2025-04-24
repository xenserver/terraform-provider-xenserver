# Create a VDI with a new SR
resource "xenserver_sr_nfs" "nfs" {
  name_label       = "Test NFS SR"
  name_description = "A test NFS storage repository"
  version          = "3"
  storage_location = "192.0.2.1:/server/path"
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

# Create a VDI with an exist SR
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

resource "xenserver_vdi" "import_vdi" {
  name_label   = "Import VDI"
  sr_uuid      = data.xenserver_sr.sr.data_items[0].uuid
  raw_vdi_path = "/tmp/livecd.ubuntu-cpc.azure.vhd"
}
