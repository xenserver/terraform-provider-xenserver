resource "xenserver_network_vlan" "vlan" {
  name_label       = "Test external network"
  name_description = "test description"
  managed          = true
  mtu              = 1600
  vlan_tag         = 1
  nic              = "NIC 0"
  other_config = {
    "flag" = "1"
  }
}

output "vlan_out" {
  value = xenserver_network_vlan.vlan.uuid
}
