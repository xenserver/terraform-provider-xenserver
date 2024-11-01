data "xenserver_pif" "pif_eth1" {
  device = "eth1"
}

# Update single PIF configuration
resource "xenserver_pif_configure" "pif_update" {
  uuid = data.xenserver_pif.pif_eth1.data_items[0].uuid
  disallow_unplug = true
  interface = {
    mode = "Static"
    ip = "192.0.2.1"
    netmask = "255.255.255.0"
  }
}

# Update multiple PIFs configuration
locals {
  pif_data = tomap({for element in data.xenserver_pif.pif_eth1.data_items: element.uuid => element})
}

resource "xenserver_pif_configure" "pif_update" {
  for_each = local.pif_data
  uuid     = each.key
  interface = {
    mode = "DHCP"
  }
}