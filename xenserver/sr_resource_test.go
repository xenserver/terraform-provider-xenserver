package xenserver

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccSRResourceConfigLocal(name_label string, name_description string, typeString string, shared string, extra_config string) string {
	return fmt.Sprintf(`
resource "xenserver_sr" "test_sr" {
	name_label       = "%s"
	name_description = "%s"
	type             = "%s"
	shared           = %s
	%s
}
`, name_label, name_description, typeString, shared, extra_config)
}

func TestAccSRResourceLocal(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testAccSRResourceConfigLocal("Test SR Local", "", "dummy", "false", ""),
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
					resource.TestCheckResourceAttrSet("xenserver_sr.test_sr", "uuid"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "xenserver_sr.test_sr",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{},
			},
			{
				Config:      providerConfig + testAccSRResourceConfigLocal("Test SR Local 2", "Test SR Description", "dummy", "true", ""),
				ExpectError: regexp.MustCompile(`"shared" doesn't expected to be updated`),
			},
			{
				Config:      providerConfig + testAccSRResourceConfigLocal("Test SR Local 2", "Test SR Description", "dummy", "false", `host = "cbdad2c6-b181-4047-ba2a-b4914bdecdbd"`),
				ExpectError: regexp.MustCompile(`"host" doesn't expected to be updated`),
			},
			{
				Config:      providerConfig + testAccSRResourceConfigLocal("Test SR Local 2", "Test SR Description", "dummy", "false", `device_config = {"key" = "value"}`),
				ExpectError: regexp.MustCompile(`"device_config" doesn't expected to be updated`),
			},
			{
				Config:      providerConfig + testAccSRResourceConfigLocal("Test SR Local 2", "Test SR Description", "user", "false", ""),
				ExpectError: regexp.MustCompile(`"type" doesn't expected to be updated`),
			},
			{
				Config:      providerConfig + testAccSRResourceConfigLocal("Test SR Local 2", "Test SR Description", "dummy", "false", `content_type = "etx4"`),
				ExpectError: regexp.MustCompile(`"content_type" doesn't expected to be updated`),
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccSRResourceConfigLocal("Test SR Local 2", "Test SR Description", "dummy", "false", ""),
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
					resource.TestCheckResourceAttrSet("xenserver_sr.test_sr", "uuid"),
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
					resource.TestCheckResourceAttrSet("xenserver_sr.test_sr", "uuid"),
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
					resource.TestCheckResourceAttrSet("xenserver_sr.test_sr", "uuid"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
