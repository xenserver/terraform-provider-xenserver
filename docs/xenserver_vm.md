
# xenserver_vm

Provides a XenServer virtual machine resource. This can be used to create, modify, and delete virtual machines.

## Example Usage
```
resource "xenserver_vm" "vm_example" {
  base_template_name = "vm-template"
  name_label         = "vm-example"
  static_mem_min  = 8589934592
  static_mem_max  = 8589934592
  dynamic_mem_min = 8589934592
  dynamic_mem_max = 8589934592
  vcpus           = 1

  hard_drive {
    is_from_template = true
    user_device      = "0"
  }

  cdrom {
    is_from_template = true
    user_device      = "3"
  }
}
```

## Arguments Reference

The following arguments are supported:

* `name_label` - (Required) Specifies the name of the VM.
* `base_template_name` - (Required) Specifies the name of the template to use to create the VM.
* `static_mem_min` - (Required) Statically-set (i.e. absolute) mininum (bytes). The value of this field indicates the least amount of memory this VM can boot with without crashing.
* `static_mem_max` - (Required) Statically-set (i.e. absolute) maximum (bytes). The value of this field at VM start time acts as a hard limit of the amount of memory a guest can use.
* `dynamic_mem_min` - (Required) Dynamic minimum (bytes).
* `dynamic_mem_max` - (Required) Dynamic maximum (bytes).
* `vcpus` - (Required) Specifies the number of VCPUs at boot.
* `boot_order` - (Optional) Specifies the boot order of the VM.
* `cdrom` - (Optional) A virtual block device of type CD. A `cdrom` block as defined below.
* `hard_drive` - (Optional) A virtual block device of type disk. A `hard_drive` block as defined below.
* `network_interface` - (Optional) A virtual network interface. A `network_interface` block as defined below. **!NEEDS WORK!**
* `other_config` - (Optional) Specifies any number of given key-value pairs in the VM's `other-config` map.

---

The `cdrom` block supports:

* `vdi_uuid` - (Optional) The UUID of the virtual disk image.
* `bootable` - (Optional) Boolean, if the VBD is bootable.
* `is_from_template` - (Required?) ?
* `mode` - (Optional) The mode the VBD should be mounted with.
* `user_device` - (Required?) The user-friendly device name e.g. 0,1,2,etc.

---

The `hard_drive` block supports:

* `vdi_uuid` - (Optional) The UUID of the virtual disk image.
* `bootable` - (Optional) Boolean, if the VBD is bootable.
* `is_from_template` - (Required?) ?
* `mode` - (Optional) The mode the VBD should be mounted with.
* `user_device` - (Required?) The user-friendly device name e.g. 0,1,2,etc.

---

The `network_interface` block supports:

* `network_uuid` - () The UUID of the virtual network interface.
* `mtu` - () The maximum transmission unit in octets.
* `device` - () The order in which VIF backends are created.

---

## Attributes Reference

The following attributes are exported:

* `id` - The instance ID.
