package xenserver

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccImportRawVdiResourceConfig(vdi_path string) string {
	return fmt.Sprintf(`
resource "xenserver_import_raw_vdi" "vdi" {
  raw_vdi_path = "%s"
}
`, vdi_path)
}

func TestAccImportRawVdiResource(t *testing.T) {
	// download the raw vdi file to the local machine
	// and set the VDI_PATH environment variable to its location
	// Example: os.Setenv("VDI_PATH", "/path/to/your/raw.vdi")
	vdi_path := os.Getenv("VDI_PATH")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      providerConfig + testAccImportRawVdiResourceConfig("wrong-path"),
				ExpectError: regexp.MustCompile("Failed to get file"),
			},
			// Create and Read testing
			{
				Config: providerConfig + testAccImportRawVdiResourceConfig(vdi_path),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xenserver_import_raw_vdi.vdi", "raw_vdi_path", vdi_path),
				),
			},
			// Update and Read testing
			{
				Config:      providerConfig + testAccImportRawVdiResourceConfig("other_path"),
				ExpectError: regexp.MustCompile("Update Not Supported"),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
