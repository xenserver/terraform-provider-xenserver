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
  server_cert_path = local.env_vars["XENSERVER_SERVER_CERT_PATH"]
}

# get the existing supporter hosts
data "xenserver_host" "supporter" {
  is_coordinator = false
}

# join a new supporter to the pool
resource "xenserver_pool" "pool" {
  name_label   = "pool"
  join_supporters = [
    {
      host = local.env_vars["SUPPORTER_HOST"]
      username = local.env_vars["SUPPORTER_USERNAME"]
      password = local.env_vars["SUPPORTER_PASSWORD"]
      server_cert_path = local.env_vars["SUPPORTER_SERVER_CERT_PATH"]
    }
  ]
  eject_supporters = [ data.xenserver_host.supporter.data_items[0].uuid ]
}


# resource "xenserver_sr_nfs" "nfs" {
#   name_label       = "NFS shared storage"
#   name_description = "A test NFS storage repository"
#   version          = "3"
#   storage_location = format("%s:%s", local.env_vars["NFS_SERVER"], local.env_vars["NFS_SERVER_PATH"])
# }

# data "xenserver_pif" "pif" {
#   device = "eth0"
# }

# data "xenserver_pif" "pif1" {
#   device = "eth3"
# }

# locals {
#   pif1_data = tomap({for element in data.xenserver_pif.pif1.data_items: element.uuid => element})
# }

# resource "xenserver_pif_configure" "pif_update" {
#   for_each = local.pif1_data
#   uuid     = each.key
#   interface = {
#     mode = "DHCP"
#   }
# }

# resource "xenserver_pool" "pool_management" {
#   name_label   = "pool"
#   default_sr = data.xenserver_sr_nfs.nfs.data_items[0].uuid
#   management_network = data.xenserver_pif.pif.data_items[0].uuid
# }