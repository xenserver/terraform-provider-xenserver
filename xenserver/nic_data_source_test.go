package xenserver

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccNICDataSourceConfig(network_type string) string {
	return fmt.Sprintf(`
data "xenserver_nic" "test_nic_data" {
	network_type = "%s"
}
`, network_type)
}

func TestAccNICDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccNICDataSourceConfig("vlan"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.xenserver_nic.test_nic_data", "network_type", "vlan"),
					resource.TestCheckResourceAttrSet("data.xenserver_nic.test_nic_data", "data_items.#"),
				),
			},
		},
	})
}
