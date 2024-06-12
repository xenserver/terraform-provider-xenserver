resource "xenserver_network" "network" {
  name_label = "Test VM Network"
}

output "network_out" {
  value = xenserver_network.network.id
}
