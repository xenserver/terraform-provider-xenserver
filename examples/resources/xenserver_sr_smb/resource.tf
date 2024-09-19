resource "xenserver_sr_smb" "smb_test" {
  name_label       = "SMB storage"
  name_description = "A test SMB storage repository"
  storage_location = "\\\\server\\path"
  username         = "username"
  password         = "password"
}

resource "xenserver_sr_smb" "smb_test" {
  name_label       = "SMB storage"
  storage_location = <<-EOF
    \\server\path
EOF
}
