package xenserver

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccSMBResourceConfig(name_label string, name_description string, storage_location string, username string, password string) string {
	return fmt.Sprintf(`
resource "xenserver_sr_smb" "test_smb" {
	name_label       = "%s"
	name_description = "%s"
	storage_location = "%s"
	username         = "%s"
	password         = "%s"
}
`, name_label, name_description, storage_location, username, password)
}

func TestAccSMBResource(t *testing.T) {
	// SMB_SERVER_PATH should be like '\\\\10.70.41.7\\share', then expected_storage_location is '\\10.70.41.7\share'
	expected_storage_location := os.Getenv("SMB_SERVER_PATH")
	storage_location := strings.ReplaceAll(expected_storage_location, "\\", "\\\\")
	username := os.Getenv("SMB_SERVER_USERNAME")
	password := os.Getenv("SMB_SERVER_PASSWORD")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testAccSMBResourceConfig("Test SMB storage repository", "", storage_location, username, password),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xenserver_sr_smb.test_smb", "name_label", "Test SMB storage repository"),
					resource.TestCheckResourceAttr("xenserver_sr_smb.test_smb", "name_description", ""),
					resource.TestCheckResourceAttr("xenserver_sr_smb.test_smb", "storage_location", expected_storage_location),
					resource.TestCheckResourceAttr("xenserver_sr_smb.test_smb", "username", username),
					resource.TestCheckResourceAttr("xenserver_sr_smb.test_smb", "password", password),
					// Verify dynamic values have any value set in the state.

					resource.TestCheckResourceAttrSet("xenserver_sr_smb.test_smb", "uuid"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "xenserver_sr_smb.test_smb",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"username", "password"},
			},
			{
				Config:      providerConfig + testAccSMBResourceConfig("Test SMB storage repository 2", "Test SMB Description", "", username, password),
				ExpectError: regexp.MustCompile(`"storage_location" doesn't expected to be updated`),
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccSMBResourceConfig("Test SMB storage repository 2", "Test SMB Description", storage_location, username, password),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xenserver_sr_smb.test_smb", "name_label", "Test SMB storage repository 2"),
					resource.TestCheckResourceAttr("xenserver_sr_smb.test_smb", "name_description", "Test SMB Description"),
					resource.TestCheckResourceAttr("xenserver_sr_smb.test_smb", "storage_location", expected_storage_location),
					resource.TestCheckResourceAttr("xenserver_sr_smb.test_smb", "username", username),
					resource.TestCheckResourceAttr("xenserver_sr_smb.test_smb", "password", password),
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("xenserver_sr_smb.test_smb", "uuid"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
