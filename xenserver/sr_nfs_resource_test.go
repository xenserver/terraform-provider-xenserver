package xenserver

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccNFSResourceConfig(name_label string, name_description string) string {
	return fmt.Sprintf(`
resource "xenserver_sr_nfs" "test_nfs" {
	name_label       = "%s"
	name_description = "%s"
	version          = "3"
	storage_location = "%s"
}
`, name_label, name_description, os.Getenv("NFS_SERVER")+":"+os.Getenv("NFS_SERVER_PATH"))
}

func TestAccNFSResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testAccNFSResourceConfig("Test NFS storage repository", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xenserver_sr_nfs.test_nfs", "name_label", "Test NFS storage repository"),
					resource.TestCheckResourceAttr("xenserver_sr_nfs.test_nfs", "name_description", ""),
					resource.TestCheckResourceAttr("xenserver_sr_nfs.test_nfs", "storage_location", os.Getenv("NFS_SERVER")+":"+os.Getenv("NFS_SERVER_PATH")),
					resource.TestCheckResourceAttr("xenserver_sr_nfs.test_nfs", "version", "3"),
					resource.TestCheckResourceAttr("xenserver_sr_nfs.test_nfs", "advanced_options", ""),
					// Verify dynamic values have any value set in the state.

					resource.TestCheckResourceAttrSet("xenserver_sr_nfs.test_nfs", "uuid"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "xenserver_sr_nfs.test_nfs",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{},
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccNFSResourceConfig("Test NFS storage repository 2", "Test NFS Description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xenserver_sr_nfs.test_nfs", "name_label", "Test NFS storage repository 2"),
					resource.TestCheckResourceAttr("xenserver_sr_nfs.test_nfs", "name_description", "Test NFS Description"),
					resource.TestCheckResourceAttr("xenserver_sr_nfs.test_nfs", "storage_location", os.Getenv("NFS_SERVER")+":"+os.Getenv("NFS_SERVER_PATH")),
					resource.TestCheckResourceAttr("xenserver_sr_nfs.test_nfs", "version", "3"),
					resource.TestCheckResourceAttr("xenserver_sr_nfs.test_nfs", "advanced_options", ""),
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("xenserver_sr_nfs.test_nfs", "uuid"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
