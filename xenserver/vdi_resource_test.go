package xenserver

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccVDIResourceConfig(name_label string, name_description string) string {
	return fmt.Sprintf(`
data "xenserver_sr" "sr" {
	name_label = "Local storage"
}

resource "xenserver_vdi" "test_vdi" {
	name_label       = "%s"
	name_description = "%s"
	sr_uuid          = data.xenserver_sr.sr.data_items[0].uuid
	virtual_size     = 1 * 1024 * 1024 * 1024
	other_config     = {
		"flag" = "1"
	}
}
`, name_label, name_description)
}

func TestAccVDIResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testAccVDIResourceConfig("Test VDI", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xenserver_vdi.test_vdi", "name_label", "Test VDI"),
					resource.TestCheckResourceAttr("xenserver_vdi.test_vdi", "name_description", ""),
					resource.TestCheckResourceAttr("xenserver_vdi.test_vdi", "virtual_size", "1073741824"),
					resource.TestCheckResourceAttr("xenserver_vdi.test_vdi", "other_config.%", "1"),
					resource.TestCheckResourceAttr("xenserver_vdi.test_vdi", "other_config.flag", "1"),
					// Verify dynamic values have any value set in the state.

					resource.TestCheckResourceAttrSet("xenserver_vdi.test_vdi", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "xenserver_vdi.test_vdi",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{},
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccVDIResourceConfig("Test VDI 2", "Test VDI description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xenserver_vdi.test_vdi", "name_label", "Test VDI 2"),
					resource.TestCheckResourceAttr("xenserver_vdi.test_vdi", "name_description", "Test VDI description"),
					resource.TestCheckResourceAttr("xenserver_vdi.test_vdi", "virtual_size", "1073741824"),
					resource.TestCheckResourceAttr("xenserver_vdi.test_vdi", "other_config.%", "1"),
					resource.TestCheckResourceAttr("xenserver_vdi.test_vdi", "other_config.flag", "1"),
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("xenserver_vdi.test_vdi", "id"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
