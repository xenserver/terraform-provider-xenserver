package xenserver

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccVMDataSourceConfig(name_label string) string {
	return fmt.Sprintf(`
resource "xenserver_vm" "test_vm" {
	name_label = "%s"
	template_name = "CentOS 7"
}

data "xenserver_vm" "test_vm_data" {
	name_label = xenserver_vm.test_vm.name_label
}
`, name_label)
}

func TestAccVMDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: providerConfig + testAccVMDataSourceConfig("virtual machine test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xenserver_vm.test_vm", "name_label", "virtual machine test"),
					resource.TestCheckResourceAttr("data.xenserver_vm.test_vm_data", "name_label", "virtual machine test"),
					resource.TestCheckResourceAttrSet("data.xenserver_vm.test_vm_data", "data_items.#"),
				),
			},
		},
	})
}
