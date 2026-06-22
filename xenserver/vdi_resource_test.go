package xenserver

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccVDIResourceConfig(name_label string, name_description string, virtual_size string, extra_config string) string {
	return fmt.Sprintf(`
resource "xenserver_sr_nfs" "nfs" {
	name_label       = "test NFS SR"
	version          = "3"
	storage_location = "%s"
}

resource "xenserver_vdi" "test_vdi" {
	name_label       = "%s"
	name_description = "%s"
	sr_uuid          = xenserver_sr_nfs.nfs.uuid
	virtual_size     = %s
	other_config     = {
		"flag" = "1"
	}
	%s
}
`, os.Getenv("NFS_SERVER")+":"+os.Getenv("NFS_SERVER_PATH"), name_label, name_description, virtual_size, extra_config)
}

func TestAccVDIResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testAccVDIResourceConfig("Test VDI", "", "1 * 1024 * 1024 * 1024", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xenserver_vdi.test_vdi", "name_label", "Test VDI"),
					resource.TestCheckResourceAttr("xenserver_vdi.test_vdi", "name_description", ""),
					resource.TestCheckResourceAttr("xenserver_vdi.test_vdi", "virtual_size", "1073741824"),
					resource.TestCheckResourceAttr("xenserver_vdi.test_vdi", "other_config.%", "1"),
					resource.TestCheckResourceAttr("xenserver_vdi.test_vdi", "other_config.flag", "1"),
					// Verify dynamic values have any value set in the state.

					resource.TestCheckResourceAttrSet("xenserver_vdi.test_vdi", "uuid"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "xenserver_vdi.test_vdi",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{},
			},
			{
				Config:      providerConfig + testAccVDIResourceConfig("Test VDI 2", "Test VDI description", "2 * 1024 * 1024 * 1024", ""),
				ExpectError: regexp.MustCompile(`"virtual_size" doesn't expected to be updated`),
			},
			{
				Config:      providerConfig + testAccVDIResourceConfig("Test VDI 2", "Test VDI description", "1 * 1024 * 1024 * 1024", `type = "dummy"`),
				ExpectError: regexp.MustCompile(`"type" doesn't expected to be updated`),
			},
			{
				Config:      providerConfig + testAccVDIResourceConfig("Test VDI 2", "Test VDI description", "1 * 1024 * 1024 * 1024", "sharable = true"),
				ExpectError: regexp.MustCompile(`"sharable" doesn't expected to be updated`),
			},
			{
				Config:      providerConfig + testAccVDIResourceConfig("Test VDI 2", "Test VDI description", "1 * 1024 * 1024 * 1024", "read_only = true"),
				ExpectError: regexp.MustCompile(`"read_only" doesn't expected to be updated`),
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccVDIResourceConfig("Test VDI 2", "Test VDI description", "1 * 1024 * 1024 * 1024", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xenserver_vdi.test_vdi", "name_label", "Test VDI 2"),
					resource.TestCheckResourceAttr("xenserver_vdi.test_vdi", "name_description", "Test VDI description"),
					resource.TestCheckResourceAttr("xenserver_vdi.test_vdi", "virtual_size", "1073741824"),
					resource.TestCheckResourceAttr("xenserver_vdi.test_vdi", "other_config.%", "1"),
					resource.TestCheckResourceAttr("xenserver_vdi.test_vdi", "other_config.flag", "1"),
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("xenserver_vdi.test_vdi", "uuid"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
