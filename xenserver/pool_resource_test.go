package xenserver

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccPoolResourceConfig(name_label string, name_descrption string, sr_index string, eth_index string) string {
	return fmt.Sprintf(`
data "xenserver_sr" "sr" {
    name_label = "Local storage"
}
  
// xe pif-reconfigure-ip uuid=<uuid of eth1> mode=dhcp gateway= DNS=
data "xenserver_pif" "pif" {
    device = "eth%s"
}

resource "xenserver_pool" "pool" {
    name_label   = "%s"
	name_description = "%s"
    default_sr = data.xenserver_sr.sr.data_items[%s].uuid
    management_network = data.xenserver_pif.pif.data_items[0].network
}
`, eth_index, name_label, name_descrption, sr_index)
}

func TestAccPoolResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testAccPoolResourceConfig("Test Pool A", "Test Pool A Description", "0", "0"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xenserver_pool.pool", "name_label", "Test Pool A"),
					resource.TestCheckResourceAttr("xenserver_pool.pool", "name_description", "Test Pool A Description"),
					resource.TestCheckResourceAttrSet("xenserver_pool.pool", "default_sr"),
					resource.TestCheckResourceAttrSet("xenserver_pool.pool", "management_network"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "xenserver_pool.pool",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{},
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccPoolResourceConfig("Test Pool B", "Test Pool B Description", "1", "1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xenserver_pool.pool", "name_label", "Test Pool B"),
					resource.TestCheckResourceAttr("xenserver_pool.pool", "name_description", "Test Pool B Description"),
					resource.TestCheckResourceAttrSet("xenserver_pool.pool", "default_sr"),
					resource.TestCheckResourceAttrSet("xenserver_pool.pool", "management_network"),
				),
			},
			// Revert changes
			{
				Config: providerConfig + testAccPoolResourceConfig("Test Pool A", "Test Pool A Description", "0", "0"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xenserver_pool.pool", "name_label", "Test Pool A"),
					resource.TestCheckResourceAttr("xenserver_pool.pool", "name_description", "Test Pool A Description"),
					resource.TestCheckResourceAttrSet("xenserver_pool.pool", "default_sr"),
					resource.TestCheckResourceAttrSet("xenserver_pool.pool", "management_network"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
	// sleep 10s to wait for supports back to enable
	time.Sleep(10 * time.Second)
}
