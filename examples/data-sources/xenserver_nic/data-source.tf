# Copyright © 2026. Citrix Systems, Inc. All Rights Reserved.
# Licensed under the Mozilla Public License 2.0 (MPL-2.0).

data "xenserver_nic" "nic" {
  network_type = "vlan"
}

output "nic_output" {
  value = data.xenserver_nic.nic.data_items
}