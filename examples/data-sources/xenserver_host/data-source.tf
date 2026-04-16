# Copyright © 2026. Citrix Systems, Inc. All Rights Reserved.
# Licensed under the Mozilla Public License 2.0 (MPL-2.0).

data "xenserver_host" "host" {
  name_label = "Test Host"
}

output "host_output" {
  value = data.xenserver_host.host.data_items
}