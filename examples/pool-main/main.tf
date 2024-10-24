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
}

data "xenserver_sr" "sr" {
  name_label = "Local storage"
}

data "xenserver_host" "host" {}

data "xenserver_pif" "pif" {
  device = "eth0"
}

data "xenserver_pif" "pif1" {
  device = "eth3"
}

locals {
  pif1_data = tomap({for element in data.xenserver_pif.pif1.data_items: element.uuid => element})
}

resource "xenserver_pif_configure" "pif_update" {
  for_each = local.pif1_data
  uuid     = each.key
  interface = {
    mode = "DHCP"
  }
}

resource "xenserver_pool" "pool" {
  name_label   = "pool-1"
  default_sr = data.xenserver_sr.sr.data_items[0].uuid
  management_network = data.xenserver_pif.pif.data_items[0].network
}