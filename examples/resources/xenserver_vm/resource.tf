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
  name_label                   = "Windows VM"
  template_name                = "Windows 11"
  static_mem_max               = 4 * 1024 * 1024 * 1024
  vcpus                        = 4
  cores_per_socket             = 2
  cdrom                        = "win11-x64_uefi.iso"
  boot_mode                    = "uefi_security"
  boot_order                   = "cdn"
  allow_windows_update_drivers = true

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

# Create a Windows 11 VM that is copy from the custom template
resource "xenserver_vm" "windows_vm_copy" {
  name_label            = "Windows VM Copy From Custom Template"
  template_name         = "Custom Windows 11 Template"
  static_mem_max        = 4 * 1024 * 1024 * 1024
  vcpus                 = 4
  cores_per_socket      = 2
  sr_for_full_disk_copy = data.xenserver_sr.sr.data_items[0].uuid

  network_interface = [
    {
      device       = "0"
      network_uuid = data.xenserver_network.network.data_items[0].uuid,
    },
  ]
}

# Create a Windows VM that is cloned from the custom template (contains Windows guest tools)
# and assign a static IPv4 address
resource "xenserver_vm" "windows_vm_static_ip" {
  name_label       = "Windows VM Static IP"
  template_name    = "Custom Windows 11 Template"
  static_mem_max   = 4 * 1024 * 1024 * 1024
  vcpus            = 4
  check_ip_timeout = 60 * 5

  network_interface = [
    {
      device       = "0"
      network_uuid = data.xenserver_network.network.data_items[0].uuid
      ipv4_address = "10.62.50.49/20"
      ipv4_gateway = "10.62.48.1"
    },
  ]

  # This requires the guest template to be pre-configured for remote access: 
  # eg. WinRM enabled with a listener, the firewall open for the port (5985/5986), and an authentication method matching the connection block (use_ntlm, or Basic + HTTPS). 
  connection {
    type     = "winrm"
    user     = "Administrator"
    password = var.password
    host     = self.ip_addresses["0"]
    port     = 5986
    https    = true
    insecure = true
    use_ntlm = false
    timeout  = "10m"
  }

  provisioner "remote-exec" {
    inline = [
      "whoami",
    ]
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
      name_label     = "Windows VM"
      template_name  = "Windows 11"
      static_mem_max = 8 * 1024 * 1024 * 1024
      vcpus          = 4
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
      name_label     = "Linux VM"
      template_name  = "Debian Bullseye 11"
      static_mem_max = 4 * 1024 * 1024 * 1024
      vcpus          = 2
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
  for_each          = local.virtual_machines
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
