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

func testAccSRDataSourceConfig(name_label string) string {
	return fmt.Sprintf(`
data "xenserver_sr" "test_sr_data" {
	name_label = "%s"
}
`, name_label)
}

func TestAccSRDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccSRDataSourceConfig("Local storage"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.xenserver_sr.test_sr_data", "name_label", "Local storage"),
					resource.TestCheckResourceAttrSet("data.xenserver_sr.test_sr_data", "data_items.#"),
				),
			},
		},
	})
}
