data "xenserver_pif" "pif" {
  device     = "eth0"
  management = true
}

output "pif_data_out" {
  value = data.xenserver_pif.pif.data_items
}
