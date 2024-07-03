package xenserver

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccSnapshotResourceConfig(name_label string) string {
	return fmt.Sprintf(`
data "xenserver_sr" "sr" {
	name_label = "Local storage"
}
	
resource "xenserver_vdi" "vdi1" {
	name_label   = "A test vdi"
	sr_uuid      = data.xenserver_sr.sr.data_items[0].uuid
	virtual_size = 100 * 1024 * 1024 * 1024
}

data "xenserver_network" "network" {}

resource "xenserver_vm" "vm" {
	name_label    = "A test virtual-machine"
	template_name = "Windows 11"
	hard_drive = [
		{
		vdi_uuid = xenserver_vdi.vdi1.uuid,
		mode     = "RW"
		},
	]
	network_interface = [
		{
		other_config = {
			ethtool-gso = "off"
		}
		device		 = "0"
		mtu          = 1600
		mac          = "11:22:33:44:55:66"
		network_uuid = data.xenserver_network.network.data_items[1].uuid,
		},
	]
}

resource "xenserver_snapshot" "test_snapshot" {
	name_label = "%s"
	vm_uuid = xenserver_vm.vm.uuid
}
`, name_label)
}

func TestAccSnapshotResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testAccSnapshotResourceConfig("Test snapshot A"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xenserver_snapshot.test_snapshot", "name_label", "Test snapshot A"),
					resource.TestCheckResourceAttr("xenserver_snapshot.test_snapshot", "with_memory", "false"),
					resource.TestCheckResourceAttrSet("xenserver_snapshot.test_snapshot", "uuid"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "xenserver_snapshot.test_snapshot",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{},
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccSnapshotResourceConfig("Test snapshot B"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xenserver_snapshot.test_snapshot", "name_label", "Test snapshot B"),
					resource.TestCheckResourceAttr("xenserver_snapshot.test_snapshot", "with_memory", "false"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
