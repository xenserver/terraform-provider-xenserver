package xenserver

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccSRResourceConfigLocal(name_label string, name_description string) string {
	return fmt.Sprintf(`
resource "xenserver_sr" "test_sr" {
	name_label       = "%s"
	name_description = "%s"
	type             = "dummy"
	shared           = false
}
`, name_label, name_description)
}

func TestAccSRResourceLocal(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testAccSRResourceConfigLocal("Test SR Local", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xenserver_sr.test_sr", "name_label", "Test SR Local"),
					resource.TestCheckResourceAttr("xenserver_sr.test_sr", "type", "dummy"),
					resource.TestCheckResourceAttr("xenserver_sr.test_sr", "shared", "false"),
					resource.TestCheckResourceAttr("xenserver_sr.test_sr", "name_description", ""),
					resource.TestCheckResourceAttr("xenserver_sr.test_sr", "content_type", ""),
					resource.TestCheckResourceAttr("xenserver_sr.test_sr", "sm_config.%", "0"),
					resource.TestCheckResourceAttr("xenserver_sr.test_sr", "device_config.%", "0"),
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("xenserver_sr.test_sr", "host"),
					resource.TestCheckResourceAttrSet("xenserver_sr.test_sr", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "xenserver_sr.test_sr",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{},
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccSRResourceConfigLocal("Test SR Local 2", "Test SR Description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xenserver_sr.test_sr", "name_label", "Test SR Local 2"),
					resource.TestCheckResourceAttr("xenserver_sr.test_sr", "type", "dummy"),
					resource.TestCheckResourceAttr("xenserver_sr.test_sr", "shared", "false"),
					resource.TestCheckResourceAttr("xenserver_sr.test_sr", "name_description", "Test SR Description"),
					resource.TestCheckResourceAttr("xenserver_sr.test_sr", "content_type", ""),
					resource.TestCheckResourceAttr("xenserver_sr.test_sr", "sm_config.%", "0"),
					resource.TestCheckResourceAttr("xenserver_sr.test_sr", "device_config.%", "0"),
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("xenserver_sr.test_sr", "host"),
					resource.TestCheckResourceAttrSet("xenserver_sr.test_sr", "id"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccSRResourceConfigShared(name_label string) string {
	return fmt.Sprintf(`
resource "xenserver_sr" "test_sr" {
	name_label    = "%s"
	type          = "nfs"
	shared        = true
	device_config = {
		server       = "%s"
		serverpath   = "%s"
		nfsversion   = "3"
	}
}
`, name_label, os.Getenv("NFS_SERVER"), os.Getenv("NFS_SERVER_PATH"))
}

func TestAccSRResourceShared(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testAccSRResourceConfigShared("Test NFS SR"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xenserver_sr.test_sr", "name_label", "Test NFS SR"),
					resource.TestCheckResourceAttr("xenserver_sr.test_sr", "name_description", ""),
					resource.TestCheckResourceAttr("xenserver_sr.test_sr", "type", "nfs"),
					resource.TestCheckResourceAttr("xenserver_sr.test_sr", "shared", "true"),
					resource.TestCheckResourceAttr("xenserver_sr.test_sr", "content_type", ""),
					resource.TestCheckResourceAttr("xenserver_sr.test_sr", "sm_config.%", "0"),
					resource.TestCheckResourceAttr("xenserver_sr.test_sr", "device_config.%", "3"),
					resource.TestCheckResourceAttr("xenserver_sr.test_sr", "device_config.serverpath", os.Getenv("NFS_SERVER_PATH")),
					resource.TestCheckResourceAttr("xenserver_sr.test_sr", "device_config.server", os.Getenv("NFS_SERVER")),
					resource.TestCheckResourceAttr("xenserver_sr.test_sr", "device_config.nfsversion", "3"),
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("xenserver_sr.test_sr", "host"),
					resource.TestCheckResourceAttrSet("xenserver_sr.test_sr", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "xenserver_sr.test_sr",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{},
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccSRResourceConfigShared("Test NFS SR 2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xenserver_sr.test_sr", "name_label", "Test NFS SR 2"),
					resource.TestCheckResourceAttr("xenserver_sr.test_sr", "name_description", ""),
					resource.TestCheckResourceAttr("xenserver_sr.test_sr", "type", "nfs"),
					resource.TestCheckResourceAttr("xenserver_sr.test_sr", "shared", "true"),
					resource.TestCheckResourceAttr("xenserver_sr.test_sr", "content_type", ""),
					resource.TestCheckResourceAttr("xenserver_sr.test_sr", "sm_config.%", "0"),
					resource.TestCheckResourceAttr("xenserver_sr.test_sr", "device_config.%", "3"),
					resource.TestCheckResourceAttr("xenserver_sr.test_sr", "device_config.serverpath", os.Getenv("NFS_SERVER_PATH")),
					resource.TestCheckResourceAttr("xenserver_sr.test_sr", "device_config.server", os.Getenv("NFS_SERVER")),
					resource.TestCheckResourceAttr("xenserver_sr.test_sr", "device_config.nfsversion", "3"),
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("xenserver_sr.test_sr", "host"),
					resource.TestCheckResourceAttrSet("xenserver_sr.test_sr", "id"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
