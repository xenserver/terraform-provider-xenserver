# Copyright © 2026. Citrix Systems, Inc. All Rights Reserved.
# Licensed under the Mozilla Public License 2.0 (MPL-2.0).

resource "xenserver_network_vlan" "vlan" {
  name_label       = "Test external network"
  name_description = "test description"
  managed          = true
  mtu              = 1600
  vlan_tag         = 1
  nic              = "NIC 0"
  other_config = {
    "flag" = "1"
  }
}
