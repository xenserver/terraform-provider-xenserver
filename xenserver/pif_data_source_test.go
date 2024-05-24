package xenserver

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

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
				),
			},
		},
	})
}
