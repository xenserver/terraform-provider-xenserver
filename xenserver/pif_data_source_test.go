package xenserver

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testAccPifDataSourceConfig(device string) string {
	return fmt.Sprintf(`
data "xenserver_pif" "test_pif_data" {
	device = "%s"
	management = true
}
`, device)
}

func testAccPifDataSourceConfig1() string {
	return `
data "xenserver_network" "test_network_data" {}
data "xenserver_pif" "test_pif_data" {
    network = data.xenserver_network.test_network_data.data_items[0].uuid
}
`
}

func TestAccPifDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: providerConfig + testAccPifDataSourceConfig("eth0"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.xenserver_pif.test_pif_data", "device", "eth0"),
					resource.TestCheckResourceAttr("data.xenserver_pif.test_pif_data", "management", "true"),
					resource.TestCheckResourceAttrSet("data.xenserver_pif.test_pif_data", "data_items.#"),
				),
			},
			{
				Config: providerConfig + testAccPifDataSourceConfig1(),
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["data.xenserver_pif.test_pif_data"]
						if !ok {
							return fmt.Errorf("Not found: %s", "data.xenserver_pif.test_pif_data")
						}
						// the length of data_items depends on the number of hosts in the pool
						attr := rs.Primary.Attributes["data_items.#"]
						value, err := strconv.Atoi(attr)
						if err != nil {
							return fmt.Errorf("Error converting attribute value to integer: %w", err)
						}
						if value < 1 {
							return fmt.Errorf("Expected length of data_items to be greater than 0, got: %d", value)
						}
						return nil
					},
				),
			},
		},
	})
}
