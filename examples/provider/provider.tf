# Copyright © 2026. Citrix Systems, Inc. All Rights Reserved.
# Licensed under the Mozilla Public License 2.0 (MPL-2.0).

provider "xenserver" {
  host     = "https://192.0.2.1"
  username = "root"
  password = var.password
}
