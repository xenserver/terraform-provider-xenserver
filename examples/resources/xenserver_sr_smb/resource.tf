# Copyright © 2026. Citrix Systems, Inc. All Rights Reserved.
# Licensed under the Mozilla Public License 2.0 (MPL-2.0).

resource "xenserver_sr_smb" "smb_test" {
  name_label       = "SMB storage"
  name_description = "A test SMB storage repository"
  storage_location = "\\\\server\\path"
  username         = "username"
  password         = "password"
}

resource "xenserver_sr_smb" "smb_test1" {
  name_label       = "SMB storage"
  storage_location = <<-EOF
    \\server\path
EOF
}

resource "xenserver_sr_smb" "smb_iso_test" {
  name_label       = "SMB ISO library"
  name_description = "A test SMB ISO library"
  type             = "iso"
  storage_location = "\\\\server\\path"
  username         = "username"
  password         = "password"
}

resource "xenserver_sr_smb" "smb_iso_test1" {
  name_label       = "SMB ISO library"
  type             = "iso"
  storage_location = <<-EOF
    \\server\path
EOF
}
