package xenserver

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccNFSResourceConfig(name_label string, name_description string, version string, storage_location string, extra_config string) string {
	return fmt.Sprintf(`
resource "xenserver_sr_nfs" "test_nfs" {
	name_label       = "%s"
	name_description = "%s"
	version          = "%s"
	storage_location = "%s"
	%s
}
`, name_label, name_description, version, storage_location, extra_config)
}

func TestAccNFSResource(t *testing.T) {
	storage_location := os.Getenv("NFS_SERVER") + ":" + os.Getenv("NFS_SERVER_PATH")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config:      providerConfig + testAccNFSResourceConfig("Test NFS storage repository 2", "Test NFS Description", "5", storage_location, ""),
				ExpectError: regexp.MustCompile(`Invalid Attribute Value Match`),
			},
			{
				Config: providerConfig + testAccNFSResourceConfig("Test NFS storage repository", "", "3", storage_location, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xenserver_sr_nfs.test_nfs", "name_label", "Test NFS storage repository"),
					resource.TestCheckResourceAttr("xenserver_sr_nfs.test_nfs", "name_description", ""),
					resource.TestCheckResourceAttr("xenserver_sr_nfs.test_nfs", "storage_location", storage_location),
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
			{
				Config:      providerConfig + testAccNFSResourceConfig("Test NFS storage repository 2", "Test NFS Description", "3", "", ""),
				ExpectError: regexp.MustCompile(`"storage_location" doesn't expected to be updated`),
			},
			{
				Config:      providerConfig + testAccNFSResourceConfig("Test NFS storage repository 2", "Test NFS Description", "4", storage_location, ""),
				ExpectError: regexp.MustCompile(`"version" doesn't expected to be updated`),
			},
			{
				Config:      providerConfig + testAccNFSResourceConfig("Test NFS storage repository 2", "Test NFS Description", "3", storage_location, `advanced_options = "key:value"`),
				ExpectError: regexp.MustCompile(`"advanced_options" doesn't expected to be updated`),
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccNFSResourceConfig("Test NFS storage repository 2", "Test NFS Description", "3", storage_location, ""),
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
