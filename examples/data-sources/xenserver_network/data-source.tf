data "xenserver_network" "network" {
  name_label = "Pool-wide network associated with eth0"
}

output "network_output" {
  value = data.xenserver_network.network.data_items
}