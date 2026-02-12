// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package xenserver

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccHostDataSourceConfig(extra_config string) string {
	return fmt.Sprintf(`
data "xenserver_host" "host_data" {
   %s
}
`, extra_config)
}

func TestAccHostDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: providerConfig + testAccHostDataSourceConfig(""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.xenserver_host.host_data", "data_items.#"),
				),
			},
			{
				Config: providerConfig + testAccHostDataSourceConfig("is_coordinator = true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.xenserver_host.host_data", "data_items.#", "1"),
				),
			},
		},
	})
}
