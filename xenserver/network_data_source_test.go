// Copyright © 2026. Citrix Systems, Inc. All Rights Reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package xenserver

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccNetworkDataSourceConfig(name_label string) string {
	return fmt.Sprintf(`
data "xenserver_network" "test_network_data" {
	name_label = "%s"
}
`, name_label)
}

func TestAccNetworkDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccNetworkDataSourceConfig("Pool-wide network associated with eth0"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.xenserver_network.test_network_data", "name_label", "Pool-wide network associated with eth0"),
					resource.TestCheckResourceAttrSet("data.xenserver_network.test_network_data", "data_items.#"),
				),
			},
		},
	})
}
