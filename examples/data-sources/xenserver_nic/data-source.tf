data "xenserver_nic" "nic" {
  network_type = "vlan"
}

output "nic_output" {
  value = data.xenserver_nic.nic.data_items
}