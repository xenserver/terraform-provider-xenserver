data "xenserver_host" "host" {
  name_label = "Test Host"
}

output "host_output" {
  value = data.xenserver_host.host.data_items
}