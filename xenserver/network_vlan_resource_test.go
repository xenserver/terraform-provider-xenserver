package xenserver

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccVlanResourceConfig(name_label string, name_description string, mtu int64) string {
	return fmt.Sprintf(`
resource "xenserver_network_vlan" "test_vlan" {
	name_label = "%s"
	name_description = "%s"
	mtu = %d
	vlan_tag = 1
	nic = "NIC 0"
}
`, name_label, name_description, mtu)
}

func TestAccVlanResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testAccVlanResourceConfig("test external network 1", "", 1500),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xenserver_network_vlan.test_vlan", "name_label", "test external network 1"),
					resource.TestCheckResourceAttr("xenserver_network_vlan.test_vlan", "name_description", ""),
					resource.TestCheckResourceAttr("xenserver_network_vlan.test_vlan", "other_config.%", "0"),
					resource.TestCheckResourceAttr("xenserver_network_vlan.test_vlan", "mtu", "1500"),
					resource.TestCheckResourceAttr("xenserver_network_vlan.test_vlan", "managed", "true"),
					resource.TestCheckResourceAttr("xenserver_network_vlan.test_vlan", "vlan_tag", "1"),
					resource.TestCheckResourceAttr("xenserver_network_vlan.test_vlan", "nic", "NIC 0"),
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("xenserver_network_vlan.test_vlan", "uuid"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "xenserver_network_vlan.test_vlan",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{},
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccVlanResourceConfig("test external network 2", "Test description", 1600),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xenserver_network_vlan.test_vlan", "name_label", "test external network 2"),
					resource.TestCheckResourceAttr("xenserver_network_vlan.test_vlan", "name_description", "Test description"),
					resource.TestCheckResourceAttr("xenserver_network_vlan.test_vlan", "other_config.%", "0"),
					resource.TestCheckResourceAttr("xenserver_network_vlan.test_vlan", "mtu", "1600"),
					resource.TestCheckResourceAttr("xenserver_network_vlan.test_vlan", "managed", "true"),
					resource.TestCheckResourceAttr("xenserver_network_vlan.test_vlan", "vlan_tag", "1"),
					resource.TestCheckResourceAttr("xenserver_network_vlan.test_vlan", "nic", "NIC 0"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
