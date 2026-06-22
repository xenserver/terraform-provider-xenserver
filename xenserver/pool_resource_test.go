package xenserver

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func pifResource(eth_index string) string {
	return fmt.Sprintf(`
// configure eth1 PIF IP
data "xenserver_pif" "pif_data" {
  device = "eth%s"
}

resource "xenserver_pif_configure" "pif1" {
  uuid = data.xenserver_pif.pif_data.data_items[0].uuid
  interface = {
    mode = "DHCP"
  }
}

resource "xenserver_pif_configure" "pif2" {
  uuid = data.xenserver_pif.pif_data.data_items[1].uuid
  interface = {
    mode = "DHCP"
  }
}

resource "xenserver_pif_configure" "pif3" {
  uuid = data.xenserver_pif.pif_data.data_items[2].uuid
  interface = {
    mode = "DHCP"
  }
}

data "xenserver_pif" "pif" {
    device = "eth%s"
}
`, eth_index, eth_index)
}

func managementNetwork(name_label string, name_description string, host_index string) string {
	return fmt.Sprintf(`
resource "xenserver_pool" "pool" {
    name_label   = "%s"
	name_description = "%s"
    default_sr = xenserver_sr_nfs.nfs.uuid
	management_network = data.xenserver_pif.pif.data_items[%s].network
}
`, name_label, name_description, host_index)
}

func testPoolResource(storage_location string, extra string) string {
	return fmt.Sprintf(`
resource "xenserver_sr_nfs" "nfs" {
	name_label       = "NFS for pool test"
	version          = "3"
	storage_location = "%s"
}

%s
`, storage_location, extra)
}

func joinSupporterParams(name_label string, name_description string, supporterHost string, supporterUsername string, supporterPassowd string) string {
	return fmt.Sprintf(`
resource "xenserver_pool" "pool" {
    name_label   = "%s"
	name_description = "%s"
    default_sr = xenserver_sr_nfs.nfs.uuid
	join_supporters = [
		{
		    host = "%s"
			username = "%s"
			password = "%s"
		}
    ]
}	
`, name_label, name_description, supporterHost, supporterUsername, supporterPassowd)
}

func TestAccPoolResource(t *testing.T) {
	// skip test if TEST_POOL is not set
	if os.Getenv("TEST_POOL") == "" {
		t.Skip("Skipping TestAccPoolResource test due to TEST_POOL not set")
	}

	storageLocation := os.Getenv("NFS_SERVER") + ":" + os.Getenv("NFS_SERVER_PATH")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Pool Join
			{
				Config: providerConfig + testPoolResource(storageLocation, joinSupporterParams(
					"Test Pool A",
					"Test Pool Join",
					os.Getenv("SUPPORTER_HOST"),
					os.Getenv("SUPPORTER_USERNAME"),
					os.Getenv("SUPPORTER_PASSWORD"))),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xenserver_pool.pool", "name_label", "Test Pool A"),
					resource.TestCheckResourceAttr("xenserver_pool.pool", "name_description", "Test Pool Join"),
				),
			},
			{
				Config: providerConfig + testPoolResource(storageLocation, joinSupporterParams(
					"Test Pool B",
					"Test Pool Join again",
					os.Getenv("SUPPORTER_HOST"),
					os.Getenv("SUPPORTER_USERNAME"),
					os.Getenv("SUPPORTER_PASSWORD"))),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xenserver_pool.pool", "name_label", "Test Pool B"),
					resource.TestCheckResourceAttr("xenserver_pool.pool", "name_description", "Test Pool Join again"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccPoolManagementNetwork(t *testing.T) {
	// skip test if TEST_POOL is not set
	if os.Getenv("TEST_POOL") == "" {
		t.Skip("Skipping TestAccPoolManagementNetwork test due to TEST_POOL not set")
	}

	storageLocation := os.Getenv("NFS_SERVER") + ":" + os.Getenv("NFS_SERVER_PATH")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Pool Management Network
			{
				Config: providerConfig + pifResource("3") + testPoolResource(storageLocation, managementNetwork("Test Pool C", "Test Pool Management Network", "2")),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xenserver_pool.pool", "name_label", "Test Pool C"),
					resource.TestCheckResourceAttr("xenserver_pool.pool", "name_description", "Test Pool Management Network"),
					resource.TestCheckResourceAttrSet("xenserver_pool.pool", "management_network"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})

	// sleep 30s to wait for supporters and management network back to enable
	time.Sleep(30 * time.Second)
}
