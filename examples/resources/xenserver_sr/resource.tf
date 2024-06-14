// A simple example of creating a local storage on XenServer
resource "xenserver_sr" "local" {
  name_label       = "Test Local SR"
  name_description = "A test local storage repository"
  type             = "dummy"
  shared           = false
  host             = "cbdad2c6-b181-4047-ba2a-b4914bdecdbd"
}

output "local_out" {
  value = xenserver_sr.local
}

// A simple example of creating a NFS SR on XenServer
resource "xenserver_sr" "nfs" {
  name_label   = "Test NFS SR"
  type         = "nfs"
  content_type = ""
  device_config = {
    server     = "10.70.58.9"
    serverpath = "/xenrtnfs"
    nfsversion = "3"
  }
  sm_config = {
    shared = "true"
  }
}

output "nfs_out" {
  value = xenserver_sr.nfs
}