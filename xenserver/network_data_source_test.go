// Copyright © 2026. Citrix Systems, Inc. All Rights Reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package xenserver

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccNetworkDataSourceConfig() string {
	return `
data "xenserver_pif" "test_pif_data" {
	management = true
}

data "xenserver_network" "test_network_data" {
	uuid = data.xenserver_pif.test_pif_data.data_items[0].network
}
`
}

func TestAccNetworkDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccNetworkDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.xenserver_network.test_network_data", "data_items.#"),
					resource.TestCheckResourceAttrPair(
						"data.xenserver_network.test_network_data", "data_items.0.uuid",
						"data.xenserver_pif.test_pif_data", "data_items.0.network",
					),
				),
			},
		},
	})
}
