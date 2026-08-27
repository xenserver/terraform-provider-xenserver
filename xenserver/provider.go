// Copyright © 2026. Citrix Systems, Inc. All Rights Reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package xenserver

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"xenapi"
)

// Ensure Provider satisfies various provider interfaces.
var _ provider.Provider = &xsProvider{}
var _ provider.ProviderWithFunctions = &xsProvider{}
var _ provider.ProviderWithValidateConfig = &xsProvider{}
var terraformProviderVersion string

// xsProvider defines the provider implementation.
type xsProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version         string
	session         *xenapi.Session
	coordinatorConf coordinatorConf
}

type coordinatorConf struct {
	Host       string
	Username   string
	Password   string
	Insecure   bool
	CACertPath string
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &xsProvider{
			version: version,
		}
	}
}

// providerModel describes the provider data model.
type providerModel struct {
	Host       types.String `tfsdk:"host"`
	Username   types.String `tfsdk:"username"`
	Password   types.String `tfsdk:"password"`
	Insecure   types.Bool   `tfsdk:"insecure"`
	CACertPath types.String `tfsdk:"ca_cert_path"`
}

type resolvedProviderConfig struct {
	host       string
	username   string
	password   string
	caCertPath string
	insecure   bool
}

func (p *xsProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "xenserver"
	resp.Version = p.version
}

func (p *xsProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The XenServer provider facilitates the management and deployment of XenServer resources. Prior to utilisation, it is necessary to configure the provider with the required credentials. For security purposes, please ensure you have reviewed the document to [protect sensitive input variables](https://developer.hashicorp.com/terraform/tutorials/configuration-language/sensitive-variables). Comprehensive information regarding resource and data source usage is available within the left-hand navigation panel.",
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				MarkdownDescription: "The address of target XenServer host." + "<br />" +
					"Can be set by using the environment variable **XENSERVER_HOST**.",
				Optional: true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "The user name of target XenServer host." + "<br />" +
					"Can be set by using the environment variable **XENSERVER_USERNAME**.",
				Optional:  true,
				Sensitive: true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "The password of target XenServer host." + "<br />" +
					"Can be set by using the environment variable **XENSERVER_PASSWORD**.",
				Optional:  true,
				Sensitive: true,
			},
			"insecure": schema.BoolAttribute{
				MarkdownDescription: "Whether to skip TLS certificate verification when connecting to the XenServer host. Defaults to `false`. " +
					"Set to `true` to disable verification — this is intended for **development and testing only** and must not be used in production or CI environments. " +
					"When `false`, `ca_cert_path` must be set." + "<br />" +
					"Can be set by using the environment variable **XENSERVER_INSECURE**.",
				Optional: true,
			},
			"ca_cert_path": schema.StringAttribute{
				MarkdownDescription: "The path to the CA certificate (PEM) used to verify the XenServer host. Required when `insecure` is `false`." + "<br />" +
					"Can be set by using the environment variable **XENSERVER_CA_CERT_PATH**.",
				Optional: true,
			},
		},
	}
}

func (p *xsProvider) ValidateConfig(ctx context.Context, req provider.ValidateConfigRequest, resp *provider.ValidateConfigResponse) {
	var data providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateProviderConfig(&data, &resp.Diagnostics)
}

func effectiveString(v types.String, envKey string) (value string, ok bool) {
	if v.IsUnknown() {
		return "", false
	}
	if !v.IsNull() {
		return v.ValueString(), true
	}
	return os.Getenv(envKey), true
}

func effectiveInsecure(v types.Bool, diags *diag.Diagnostics) (insecure bool, known bool) {
	if v.IsUnknown() {
		return false, false
	}
	if !v.IsNull() {
		return v.ValueBool(), true
	}
	env := os.Getenv("XENSERVER_INSECURE")
	if env == "" {
		return false, true
	}
	parsed, err := strconv.ParseBool(strings.ToLower(env))
	if err != nil {
		diags.AddAttributeError(
			path.Root("insecure"),
			"Invalid Insecure Configuration",
			"The environment variable XENSERVER_INSECURE must be a valid boolean ('true' or 'false').",
		)
		return false, false
	}
	return parsed, true
}

func validateProviderConfig(data *providerModel, diags *diag.Diagnostics) resolvedProviderConfig {
	host, hostKnown := effectiveString(data.Host, "XENSERVER_HOST")
	username, usernameKnown := effectiveString(data.Username, "XENSERVER_USERNAME")
	password, passwordKnown := effectiveString(data.Password, "XENSERVER_PASSWORD")
	caCertPath, certKnown := effectiveString(data.CACertPath, "XENSERVER_CA_CERT_PATH")
	insecure, insecureKnown := effectiveInsecure(data.Insecure, diags)

	if hostKnown && host == "" {
		diags.AddAttributeError(
			path.Root("host"),
			"Missing Host Configuration",
			"The provider cannot create the XenServer API client as there is a missing or empty value for the host. "+
				"Set the host value in the configuration or use the XENSERVER_HOST environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}
	if usernameKnown && username == "" {
		diags.AddAttributeError(
			path.Root("username"),
			"Missing Username Configuration",
			"The provider cannot create the XenServer API client as there is a missing or empty value for the username. "+
				"Set the username value in the configuration or use the XENSERVER_USERNAME environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}
	if passwordKnown && password == "" {
		diags.AddAttributeError(
			path.Root("password"),
			"Missing Password Configuration",
			"The provider cannot create the XenServer API client as there is a missing or empty value for the password. "+
				"Set the password value in the configuration or use the XENSERVER_PASSWORD environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}
	if insecureKnown && !insecure && certKnown && caCertPath == "" {
		diags.AddAttributeError(
			path.Root("ca_cert_path"),
			"Missing CA Certificate Path Configuration",
			"The provider cannot create the XenServer API client as there is a missing or empty value for the ca_cert_path. "+
				"Set the ca_cert_path value in the configuration or use the XENSERVER_CA_CERT_PATH environment variable. "+
				"Alternatively set insecure = true to skip verification (development and testing only).",
		)
	}

	return resolvedProviderConfig{
		host:       host,
		username:   username,
		password:   password,
		caCertPath: caCertPath,
		insecure:   insecure,
	}
}

func (p *xsProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Debug(ctx, "Configuring XenServer Client")
	var data providerModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	terraformProviderVersion = p.version

	cfg := validateProviderConfig(&data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "username", "password")
	ctx = tflog.SetField(ctx, "host", cfg.host)
	ctx = tflog.SetField(ctx, "insecure", cfg.insecure)
	ctx = tflog.SetField(ctx, "ca_cert_path", cfg.caCertPath)
	tflog.Debug(ctx, "Creating XenServer API session")

	session, err := loginServer(cfg.host, cfg.username, cfg.password, cfg.insecure, cfg.caCertPath)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create XenServer API client",
			"An unexpected error occurred when creating the XenServer API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"XenServer client Error: "+err.Error(),
		)
		return
	}

	p.coordinatorConf.Host = ensureHTTPS(cfg.host)
	p.coordinatorConf.Username = cfg.username
	p.coordinatorConf.Password = cfg.password
	p.coordinatorConf.Insecure = cfg.insecure
	p.coordinatorConf.CACertPath = cfg.caCertPath
	p.session = session

	// the xsProvider type itself is made available for resources and data sources
	resp.DataSourceData = p
	resp.ResourceData = p
}

func ensureHTTPS(host string) string {
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimPrefix(host, "https://")
	return "https://" + host
}

func loginServer(host string, username string, password string, insecure bool, caCertPath string) (*xenapi.Session, error) {
	// check if host, username, password are non-empty
	if host == "" || username == "" || password == "" {
		return nil, errors.New("host, username, password cannot be empty")
	}

	if !insecure && caCertPath == "" {
		return nil, errors.New("ca_cert_path cannot be empty when insecure is false")
	}

	if !insecure {
		if _, err := loadCACertPool(caCertPath); err != nil {
			return nil, err
		}
	}

	host = ensureHTTPS(host)

	opts := &xenapi.ClientOpts{
		URL: host,
		Headers: map[string]string{
			"User-Agent": "XenServerTerraformProvider/" + terraformProviderVersion,
		},
	}
	if !insecure {
		opts.SecureOpts = &xenapi.SecureOpts{
			ServerCert: caCertPath,
		}
	}
	session := xenapi.NewSession(opts)

	_, err := session.LoginWithPassword(username, password, "1.0", "terraform provider")
	if err != nil {
		return nil, errors.New(err.Error())
	}

	return session, nil
}

func (p *xsProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewVMResource,
		NewPoolResource,
		NewSRResource,
		NewNFSResource,
		NewSMBResource,
		NewVDIResource,
		NewVlanResource,
		NewSnapshotResource,
		NewPIFConfigureResource,
	}
}

func (p *xsProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewPIFDataSource,
		NewSRDataSource,
		NewVMDataSource,
		NewNetworkDataSource,
		NewNICDataSource,
		NewHostDataSource,
	}
}

func (p *xsProvider) Functions(_ context.Context) []func() function.Function {
	return nil
}
