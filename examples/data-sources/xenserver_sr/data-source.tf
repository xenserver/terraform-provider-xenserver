# Copyright © 2026. Citrix Systems, Inc. All Rights Reserved.
# Licensed under the Mozilla Public License 2.0 (MPL-2.0).

data "xenserver_sr" "sr" {
  name_label = "Local storage"
}

output "local_storage_output" {
  value = data.xenserver_sr.sr.data_items
}