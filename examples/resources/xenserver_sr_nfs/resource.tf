resource "xenserver_sr_nfs" "nfs_test" {
  name_label       = "NFS virtual disk storage"
  name_description = "A test NFS storage repository"
  version          = "3"
  storage_location = "10.70.58.9:/xenrtnfs"
}

output "nfs_test_out" {
  value = xenserver_sr_nfs.nfs_test
}