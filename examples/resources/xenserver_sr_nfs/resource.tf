resource "xenserver_sr_nfs" "nfs_test" {
  name_label       = "NFS virtual disk storage"
  name_description = "A test NFS storage repository"
  version          = "3"
  storage_location = "server:/path"
}

resource "xenserver_sr_nfs" "nfs_iso_test" {
  name_label       = "NFS ISO library"
  name_description = "A test NFS ISO library"
  type             = "iso"
  version          = "4"
  storage_location = "server:/path"
}
