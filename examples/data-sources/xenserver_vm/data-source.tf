data "xenserver_vm" "vm_data" {}

output "vm_data_out" {
  value = data.xenserver_vm.vm_data.data_items
}