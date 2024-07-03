package xenserver

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccVMDataSourceConfig(name_label string) string {
	return fmt.Sprintf(`
data "xenserver_sr" "sr" {
  name_label = "Local storage"
}

resource "xenserver_vdi" "vdi" {
  name_label       = "local-storage-vdi"
  sr_uuid          = data.xenserver_sr.sr.data_items[0].uuid
  virtual_size     = 100 * 1024 * 1024 * 1024
}

data "xenserver_network" "network" {}

resource "xenserver_vm" "test_vm" {
  name_label = "%s"
  template_name = "Windows 11"
  hard_drive = [ 
    { 
      vdi_uuid = xenserver_vdi.vdi.id,
      bootable = true,
      mode = "RW"
    },
  ]

  network_interface = [
    {
      other_config = {
        ethtool-gso = "off"
      }
      device       = "0"
      mtu          = 1600
      mac          = "11:22:33:44:55:66"
      network_uuid = data.xenserver_network.network.data_items[1].uuid,
    },
  ]

  other_config = {
  	"flag" = "1"
  }
}

data "xenserver_vm" "test_vm_data" {
	name_label = xenserver_vm.test_vm.name_label
}
`, name_label)
}

func TestAccVMDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: providerConfig + testAccVMDataSourceConfig("virtual machine test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xenserver_vm.test_vm", "name_label", "virtual machine test"),
					resource.TestCheckResourceAttr("data.xenserver_vm.test_vm_data", "name_label", "virtual machine test"),
					resource.TestCheckResourceAttrSet("data.xenserver_vm.test_vm_data", "data_items.#"),
				),
			},
		},
	})
}
