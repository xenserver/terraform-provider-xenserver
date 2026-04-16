# Copyright © 2026. Citrix Systems, Inc. All Rights Reserved.
# Licensed under the Mozilla Public License 2.0 (MPL-2.0).

data "xenserver_network" "network" {
  name_label = "Pool-wide network associated with eth0"
}

output "network_output" {
  value = data.xenserver_network.network.data_items
}