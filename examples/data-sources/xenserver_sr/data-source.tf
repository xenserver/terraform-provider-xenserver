data "xenserver_sr" "sr" {
  name_label = "Local storage"
}

output "local_storage_output" {
  value = data.xenserver_sr.sr.data_items
}