package xenserver

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccVMResourceConfig(name_label string, memory int, vcpu int, bootable string, mode string, mac string, device string) string {
	return fmt.Sprintf(`
data "xenserver_sr" "sr" {
  name_label = "Local storage"
}

resource "xenserver_vdi" "vdi" {
  name_label       = "local-storage-vdi"
  sr_uuid          = data.xenserver_sr.sr.data_items[0].uuid
  virtual_size     = 100 * 1024 * 1024 * 1024
}

data "xenserver_network" "network" {}

resource "xenserver_vm" "test_vm" {
  name_label = "%s"
  template_name = "Windows 11"
  static_mem_max = %d * 1024 * 1024 * 1024
  vcpus         = %d
  hard_drive = [
    { 
      vdi_uuid = xenserver_vdi.vdi.uuid,
      bootable = %s,
      mode = "%s"
    },
  ]
  network_interface = [
    {
      other_config = {
        ethtool-gso = "off"
      }
      mac          = "%s"
      device       = "%s"
      network_uuid = data.xenserver_network.network.data_items[1].uuid,
    },
  ]
  other_config = {
  	"flag" = "1"
  }
}
`, name_label, memory, vcpu, bootable, mode, mac, device)
}

func TestAccVMResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Testing with expected failure
			{
				Config:      providerConfig + testAccVMResourceConfig("invalid vm config", 4, 4, "true", "RW", "invalid mac address", "0"),
				ExpectError: regexp.MustCompile("Input is not a valid MAC address"),
			},
			{
				Config:      providerConfig + testAccVMResourceConfig("invalid vm config", 4, 4, "false", "invalid mode", "11:22:33:44:55:66", "1"),
				ExpectError: regexp.MustCompile(`mode value must be one of:\n\["RO" "RW"\]`),
			},
			// Create and Read testing
			{
				Config: providerConfig + testAccVMResourceConfig("test vm 1", 4, 4, "true", "RW", "11:22:33:44:55:66", "0"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xenserver_vm.test_vm", "name_label", "test vm 1"),
					resource.TestCheckResourceAttr("xenserver_vm.test_vm", "template_name", "Windows 11"),
					resource.TestCheckResourceAttr("xenserver_vm.test_vm", "static_mem_min", "4294967296"),
					resource.TestCheckResourceAttr("xenserver_vm.test_vm", "static_mem_max", "4294967296"),
					resource.TestCheckResourceAttr("xenserver_vm.test_vm", "dynamic_mem_min", "4294967296"),
					resource.TestCheckResourceAttr("xenserver_vm.test_vm", "dynamic_mem_max", "4294967296"),
					resource.TestCheckResourceAttr("xenserver_vm.test_vm", "vcpus", "4"),
					resource.TestCheckResourceAttrSet("xenserver_vm.test_vm", "cores_per_socket"),
					resource.TestCheckResourceAttr("xenserver_vm.test_vm", "hard_drive.#", "1"),
					resource.TestCheckResourceAttr("xenserver_vm.test_vm", "hard_drive.0.%", "4"),
					resource.TestCheckResourceAttr("xenserver_vm.test_vm", "hard_drive.0.mode", "RW"),
					resource.TestCheckResourceAttr("xenserver_vm.test_vm", "hard_drive.0.bootable", "true"),
					resource.TestCheckResourceAttr("xenserver_vm.test_vm", "network_interface.#", "1"),
					resource.TestCheckResourceAttr("xenserver_vm.test_vm", "network_interface.0.%", "5"),
					resource.TestCheckResourceAttr("xenserver_vm.test_vm", "network_interface.0.device", "0"),
					resource.TestCheckResourceAttr("xenserver_vm.test_vm", "network_interface.0.mac", "11:22:33:44:55:66"),
					resource.TestCheckResourceAttrSet("xenserver_vm.test_vm", "network_interface.0.vif_ref"),
					resource.TestCheckResourceAttr("xenserver_vm.test_vm", "other_config.%", "1"),
					resource.TestCheckResourceAttr("xenserver_vm.test_vm", "other_config.flag", "1"),
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("xenserver_vm.test_vm", "uuid"),
				),
			},
			// Update with expected failure
			{
				Config:      providerConfig + testAccVMResourceConfig("test vm 1", 3, 4, "true", "RW", "44:55:66:11:22:33", "0"),
				ExpectError: regexp.MustCompile(`"network_interface.mac" doesn't expected to be updated.*`),
			},
			{
				Config:      providerConfig + testAccVMResourceConfig("test vm 1", 3, 3, "false", "RO", "11:22:33:44:55:66", "1"),
				ExpectError: regexp.MustCompile("3 cores could not fit to 2 cores-per-socket topology*"),
			},
			// Update and Read testing
			// change the network_interface device
			{
				Config: providerConfig + testAccVMResourceConfig("test vm 1", 3, 2, "false", "RO", "11:22:33:44:55:66", "1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("xenserver_vm.test_vm", "name_label", "test vm 1"),
					resource.TestCheckResourceAttr("xenserver_vm.test_vm", "template_name", "Windows 11"),
					resource.TestCheckResourceAttr("xenserver_vm.test_vm", "static_mem_min", "3221225472"),
					resource.TestCheckResourceAttr("xenserver_vm.test_vm", "static_mem_max", "3221225472"),
					resource.TestCheckResourceAttr("xenserver_vm.test_vm", "dynamic_mem_min", "3221225472"),
					resource.TestCheckResourceAttr("xenserver_vm.test_vm", "dynamic_mem_max", "3221225472"),
					resource.TestCheckResourceAttr("xenserver_vm.test_vm", "vcpus", "2"),
					resource.TestCheckResourceAttrSet("xenserver_vm.test_vm", "cores_per_socket"),
					resource.TestCheckResourceAttr("xenserver_vm.test_vm", "hard_drive.#", "1"),
					resource.TestCheckResourceAttr("xenserver_vm.test_vm", "hard_drive.0.mode", "RO"),
					resource.TestCheckResourceAttr("xenserver_vm.test_vm", "hard_drive.0.bootable", "false"),
					resource.TestCheckResourceAttr("xenserver_vm.test_vm", "network_interface.#", "1"),
					resource.TestCheckResourceAttr("xenserver_vm.test_vm", "network_interface.0.device", "1"),
					resource.TestCheckResourceAttr("xenserver_vm.test_vm", "network_interface.0.mac", "11:22:33:44:55:66"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "xenserver_vm.test_vm",
				ImportState:       true,
				ImportStateVerify: true,
				// This is not normally necessary, but is here because this
				// example code does not have an actual upstream service.
				// Once the Read method is able to refresh information from
				// the upstream service, this can be removed.
				ImportStateVerifyIgnore: []string{},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
