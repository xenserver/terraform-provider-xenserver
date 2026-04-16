# Copyright © 2026. Citrix Systems, Inc. All Rights Reserved.
# Licensed under the Mozilla Public License 2.0 (MPL-2.0).

data "xenserver_vm" "vm_data" {}

output "vm_data_out" {
  value = data.xenserver_vm.vm_data.data_items
}