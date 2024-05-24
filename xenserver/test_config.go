package xenserver

import (
	"fmt"
	"os"
)

var (
	providerConfig = fmt.Sprintf(`
  provider "xenserver" {
	host     = "%s"
	username = "%s"
	password = "%s"
}
`, os.Getenv("XENSERVER_HOST"), os.Getenv("XENSERVER_USERNAME"), os.Getenv("XENSERVER_PASSWORD"))
)

func testAccPifDataSourceConfig(device string) string {
	return fmt.Sprintf(`
data "xenserver_pif" "test_pif_data" {
	device = "%s"
	management = true
}
`, device)
}
