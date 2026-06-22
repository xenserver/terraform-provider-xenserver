# Create a local storage
resource "xenserver_sr" "local" {
  name_label       = "Test Local SR"
  name_description = "A test local storage repository"
  type             = "dummy"
  shared           = false
  host             = "cbdad2c6-b181-4047-ba2a-b4914bdecdbd"
}

# Create a NFS SR
resource "xenserver_sr" "nfs" {
  name_label   = "Test NFS SR"
  type         = "nfs"
  content_type = ""
  shared       = true
  device_config = {
    server     = "1.1.1.1"
    serverpath = "/server/path"
    nfsversion = "3"
  }
  sm_config = {
    shared = "true"
  }
}
