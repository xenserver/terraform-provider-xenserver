# Copyright © 2026. Citrix Systems, Inc. All Rights Reserved.
# Licensed under the Mozilla Public License 2.0 (MPL-2.0).

data "xenserver_pif" "pif" {
  device     = "eth0"
  management = true
}

output "pif_data_out" {
  value = data.xenserver_pif.pif.data_items
}
