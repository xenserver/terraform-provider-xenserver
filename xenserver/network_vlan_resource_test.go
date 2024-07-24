package xenserver

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccVlanResourceConfig(name_label string, name_description string, mtu int, tag int, nic string, extra_config string) string {
	return fmt.Sprintf(`
resource "xenserver_network_vlan" "test_vlan" {
	name_label = "%s"
	name_description = "%s"
	mtu = %d
	vlan_tag = %d
	nic = "%s"
	%s
}
`, name_label, name_description, mtu, tag, nic, extra_config)
}

func TestAccVlanResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      providerConfig + testAccVlanResourceConfig("test external network 1", "", -1, 1, "NIC 0", ""),
				ExpectError: regexp.MustCompile("Attribute mtu value must be at least 0"),
			},
			{
				Config:      providerConfig + testAccVlanResourceConfig("test external network 1", "", 1500, 1, "Error NIC 0", ""),
				ExpectError: regexp.MustCompile(`Attribute nic must start with "NIC", "Bond" or "NIC-SR-IOV"`),
			},
			// Create and Read testing
			{
				Config: providerConfig + testAccVlanResourceConfig("test external network 1", "", 1500, 1, "NIC 0", ""),
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
			{
				Config:      providerConfig + testAccVlanResourceConfig("test external network 1", "", 1500, 1, "NIC 1", ""),
				ExpectError: regexp.MustCompile(`"nic" doesn't expected to be updated`),
			},
			{
				Config:      providerConfig + testAccVlanResourceConfig("test external network 1", "", 1500, 2, "NIC 0", ""),
				ExpectError: regexp.MustCompile(`"vlan_tag" doesn't expected to be updated`),
			},
			{
				Config:      providerConfig + testAccVlanResourceConfig("test external network 1", "", 1500, 1, "NIC 0", "managed = false"),
				ExpectError: regexp.MustCompile(`"managed" doesn't expected to be updated`),
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccVlanResourceConfig("test external network 2", "Test description", 1600, 1, "NIC 0", ""),
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
