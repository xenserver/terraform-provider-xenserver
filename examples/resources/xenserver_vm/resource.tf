data "xenserver_sr" "sr" {
  name_label = "Local storage"
}

resource "xenserver_vdi" "vdi1" {
  name_label   = "local-storage-vdi-1"
  sr_uuid      = data.xenserver_sr.sr.data_items[0].uuid
  virtual_size = 100 * 1024 * 1024 * 1024
}
resource "xenserver_vdi" "vdi2" {
  name_label   = "local-storage-vdi-2"
  sr_uuid      = data.xenserver_sr.sr.data_items[0].uuid
  virtual_size = 100 * 1024 * 1024 * 1024
}

data "xenserver_network" "network" {}

# Create a Windows 11 VM that is cloned from the base template
resource "xenserver_vm" "windows_vm" {
  name_label       = "Windows VM"
  template_name    = "Windows 11"
  static_mem_max   = 4 * 1024 * 1024 * 1024
  vcpus            = 4
  cores_per_socket = 2
  cdrom            = "win11-x64_uefi.iso"
  boot_mode        = "uefi_security"
  boot_order       = "cdn"

  hard_drive = [
    {
      vdi_uuid = xenserver_vdi.vdi1.uuid,
      bootable = true,
      mode     = "RW"
    },
    {
      vdi_uuid = xenserver_vdi.vdi2.uuid,
      bootable = false,
      mode     = "RO"
    },
  ]

  network_interface = [
    {
      device       = "0"
      network_uuid = data.xenserver_network.network.data_items[0].uuid,
    },
    {
      device = "1"
      other_config = {
        ethtool-gso = "off"
      }
      mac          = "11:22:33:44:55:66"
      network_uuid = data.xenserver_network.network.data_items[1].uuid,
    },
  ]

  other_config = {
    "tf_created" = "true"
  }
}

# Create a Linux VM that is cloned from the custom template
resource "xenserver_vm" "linux_vm" {
  name_label       = "Linux VM"
  template_name    = "CustomTemplate"
  static_mem_max   = 4 * 1024 * 1024 * 1024
  vcpus            = 4
  check_ip_timeout = 60 * 5

  # Don't need to set up a hard drive if the custom template includes it

  # The network interfaces in the custom template would be removed, so we need to create new one by user-defined.
  network_interface = [
    {
      network_uuid = data.xenserver_network.network.data_items[0].uuid,
      device       = "0"
    },
  ]

  connection {
    type     = "ssh"
    user     = "root"
    password = var.password
    host     = self.default_ip
  }

  provisioner "remote-exec" {
    inline = [
      "cat /etc/os-release",
    ]
  }
}

variable "password" {
  type        = string
  description = "The password for the Linux VM"
  sensitive   = true
}

# Create multiple VMs
locals {
  virtual_machines = {
    "windows-vm" = {
      name_label       = "Windows VM"
      template_name    = "Windows 11"
      static_mem_max   = 8 * 1024 * 1024 * 1024
      vcpus            = 4
      hard_drive = [
        {
          vdi_uuid = xenserver_vdi.vdi1.uuid,
          bootable = true,
          mode     = "RW"
        },
      ]
      network_interface = [
        {
          network_uuid = data.xenserver_network.network.data_items[0].uuid,
          device       = "0"
        },
      ]
    }
    "linux-vm" = {
      name_label       = "Linux VM"
      template_name    = "Debian Bullseye 11"
      static_mem_max   = 4 * 1024 * 1024 * 1024
      vcpus            = 2
      hard_drive = [
        {
          vdi_uuid = xenserver_vdi.vdi2.uuid,
          bootable = true,
          mode     = "RW"
        },
      ]
      network_interface = [
        {
          network_uuid = data.xenserver_network.network.data_items[0].uuid,
          device       = "0"
        },
      ]
    }
  }
}

resource "xenserver_vm" "vm" {
  for_each = local.virtual_machines
  name_label        = each.value.name_label
  template_name     = each.value.template_name 
  static_mem_max    = each.value.static_mem_max
  vcpus             = each.value.vcpus
  hard_drive        = each.value.hard_drive
  network_interface = each.value.network_interface
}

output "vm_out" {
  value = {
    for vm in xenserver_vm.vm : vm.name_label => vm
  }
}
