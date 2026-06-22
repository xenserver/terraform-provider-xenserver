package xenserver

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccPIFConfigureResourceConfig(disallow_unplug string, mode string) string {
	return fmt.Sprintf(`
data "xenserver_pif" "pif" {
  device = "eth1"
}

resource "xenserver_pif_configure" "pif_update" {
  uuid = data.xenserver_pif.pif.data_items[0].uuid
  disallow_unplug = %s
  interface = {
    mode = "%s"
  }
}
`, disallow_unplug, mode)
}

func TestAccPIFConfigureResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      providerConfig + testAccPIFConfigureResourceConfig("false", "wrong-type"),
				ExpectError: regexp.MustCompile(`Invalid Attribute Value Match`),
			},
			// Create and Read testing
			{
				Config: providerConfig + testAccPIFConfigureResourceConfig("false", "None"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xenserver_pif_configure.pif_update", "disallow_unplug", "false"),
					resource.TestCheckResourceAttr("xenserver_pif_configure.pif_update", "interface.mode", "None"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "xenserver_pif_configure.pif_update",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"disallow_unplug", "interface"},
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccPIFConfigureResourceConfig("true", "DHCP"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xenserver_pif_configure.pif_update", "disallow_unplug", "true"),
					resource.TestCheckResourceAttr("xenserver_pif_configure.pif_update", "interface.mode", "DHCP"),
				),
			},
			// Revert changes
			{
				Config: providerConfig + testAccPIFConfigureResourceConfig("false", "None"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xenserver_pif_configure.pif_update", "disallow_unplug", "false"),
					resource.TestCheckResourceAttr("xenserver_pif_configure.pif_update", "interface.mode", "None"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
