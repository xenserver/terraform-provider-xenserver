locals {
  env_vars = { for tuple in regexall("export\\s*(\\S*)\\s*=\\s*(\\S*)\\s*", file("../../.env")) : tuple[0] => tuple[1] }
}

terraform {
  required_providers {
    xenserver = {
      source = "xenserver/xenserver"
    }
  }
}

provider "xenserver" {
  host             = local.env_vars["XENSERVER_HOST"]
  username         = local.env_vars["XENSERVER_USERNAME"]
  password         = local.env_vars["XENSERVER_PASSWORD"]
  skip_verify      = local.env_vars["XENSERVER_SKIP_VERIFY"]
  server_cert_path = local.env_vars["XENSERVER_SERVER_CERT_PATH"]
}

data "xenserver_host" "host" {}

output "host_output" {
  value = data.xenserver_host.host.data_items
}

data "xenserver_network" "network" {}

output "network_output" {
  value = data.xenserver_network.network.data_items
}

data "xenserver_nic" "nic" {}

output "nic_output" {
  value = data.xenserver_nic.nic.data_items
}

data "xenserver_pif" "pif" {}

output "pif_data_out" {
  value = data.xenserver_pif.pif.data_items
}

data "xenserver_sr" "sr" {}

output "local_storage_output" {
  value = data.xenserver_sr.sr.data_items
}

data "xenserver_vm" "vm_data" {
    name_label = "TestVM"
}

output "vm_data_out" {
  value = data.xenserver_vm.vm_data.data_items
}
