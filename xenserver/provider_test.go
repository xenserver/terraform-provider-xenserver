// Copyright © 2026. Citrix Systems, Inc. All Rights Reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package xenserver

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"xenserver": providerserver.NewProtocol6WithError(New("test")()),
}

var providerConfig = fmt.Sprintf(`
provider "xenserver" {
	host     = "%s"
	username = "%s"
	password = "%s"
	%s
}
`, os.Getenv("XENSERVER_HOST"), os.Getenv("XENSERVER_USERNAME"), os.Getenv("XENSERVER_PASSWORD"), providerTLSConfig())

// providerTLSConfig renders insecure / ca_cert_path only when their env vars are set.
func providerTLSConfig() string {
	var lines []string
	if v := os.Getenv("XENSERVER_INSECURE"); v != "" {
		lines = append(lines, "insecure = "+v)
	}
	if v := os.Getenv("XENSERVER_CA_CERT_PATH"); v != "" {
		lines = append(lines, `ca_cert_path = "`+v+`"`)
	}
	return strings.Join(lines, "\n\t")
}

// skipIfEnvUnset skips the test when any of the given env vars is unset.
func skipIfEnvUnset(t *testing.T, keys ...string) {
	t.Helper()
	var missing []string
	for _, key := range keys {
		if os.Getenv(key) == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		t.Skipf("Skip, missing env: %s", strings.Join(missing, ", "))
	}
}

func TestValidateProviderConfig(t *testing.T) {
	cases := []struct {
		name    string
		model   providerModel
		env     map[string]string
		wantErr bool
	}{
		{
			name: "valid: verification with certificate",
			model: providerModel{
				Host:       types.StringValue("192.0.2.1"),
				Username:   types.StringValue("root"),
				Password:   types.StringValue("secret"),
				Insecure:   types.BoolValue(false),
				CACertPath: types.StringValue("/opt/cert.pem"),
			},
		},
		{
			name: "valid: insecure true without certificate",
			model: providerModel{
				Host:       types.StringValue("192.0.2.1"),
				Username:   types.StringValue("root"),
				Password:   types.StringValue("secret"),
				Insecure:   types.BoolValue(true),
				CACertPath: types.StringNull(),
			},
		},
		{
			name: "valid: required values from environment",
			model: providerModel{
				Host:       types.StringNull(),
				Username:   types.StringNull(),
				Password:   types.StringNull(),
				Insecure:   types.BoolValue(true),
				CACertPath: types.StringNull(),
			},
			env: map[string]string{
				"XENSERVER_HOST":     "192.0.2.1",
				"XENSERVER_USERNAME": "root",
				"XENSERVER_PASSWORD": "secret",
			},
		},
		{
			name: "invalid: insecure false without certificate",
			model: providerModel{
				Host:       types.StringValue("192.0.2.1"),
				Username:   types.StringValue("root"),
				Password:   types.StringValue("secret"),
				Insecure:   types.BoolValue(false),
				CACertPath: types.StringNull(),
			},
			wantErr: true,
		},
		{
			name: "invalid: default insecure (unset) requires certificate",
			model: providerModel{
				Host:       types.StringValue("192.0.2.1"),
				Username:   types.StringValue("root"),
				Password:   types.StringValue("secret"),
				Insecure:   types.BoolNull(),
				CACertPath: types.StringNull(),
			},
			wantErr: true,
		},
		{
			name: "invalid: missing host",
			model: providerModel{
				Host:       types.StringNull(),
				Username:   types.StringValue("root"),
				Password:   types.StringValue("secret"),
				Insecure:   types.BoolValue(true),
				CACertPath: types.StringNull(),
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Neutralise ambient environment, then apply the case's env.
			for _, key := range []string{
				"XENSERVER_HOST", "XENSERVER_USERNAME", "XENSERVER_PASSWORD",
				"XENSERVER_INSECURE", "XENSERVER_CA_CERT_PATH",
			} {
				t.Setenv(key, "")
			}
			for k, v := range tc.env {
				t.Setenv(k, v)
			}

			var diags diag.Diagnostics
			validateProviderConfig(&tc.model, &diags)
			if got := diags.HasError(); got != tc.wantErr {
				t.Fatalf("validateProviderConfig() hasError = %v, want %v (diagnostics: %v)", got, tc.wantErr, diags)
			}
		})
	}
}
