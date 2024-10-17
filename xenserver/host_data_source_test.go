package xenserver

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccHostDataSourceConfig() string {
	return `
data "xenserver_host" "test_host_data" {}
`
}

func TestAccHostDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: providerConfig + testAccHostDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.xenserver_host.test_host_data", "data_items.#"),
				),
			},
		},
	})
}
