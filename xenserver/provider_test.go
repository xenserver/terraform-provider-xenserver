package xenserver

import (
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"xenserver": providerserver.NewProtocol6WithError(New("test")()),
}

var (
	providerConfig = fmt.Sprintf(`
provider "xenserver" {
	host     = "%s"
	username = "%s"
	password = "%s"
}
`, os.Getenv("XENSERVER_HOST"), os.Getenv("XENSERVER_USERNAME"), os.Getenv("XENSERVER_PASSWORD"))
)
