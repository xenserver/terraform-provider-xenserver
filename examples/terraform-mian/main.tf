terraform {
  required_providers {
    xenserver = {
      source = "xenserver/xenserver"
    }
  }
}

provider "xenserver" {
  host     = "https://10.70.40.100"
  username = "root"
  password = "BOfpcNyZ5cMe"
}

data "xenserver_pif" "pif_data" {
  device     = "eth0"
  management = true
}

output "pif_data_out" {
  value = data.xenserver_pif.pif_data
}

resource "xenserver_vm" "vm" {
  name_label    = "Test Centos Vm"
  template_name = "CentOS 7"
  other_config = {
    flag = "1"
  }
}

output "vm_out" {
  value = xenserver_vm.vm
}
