package xenserver

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
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

// xsProvider defines the provider implementation.
type xsProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
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
	Host     types.String `tfsdk:"host"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

func (p *xsProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "xenserver"
	resp.Version = p.version
}

func (p *xsProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				MarkdownDescription: "The URL of target Xenserver host",
				Required:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "The user name of target Xenserver host",
				Required:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "The password of target Xenserver host",
				Required:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *xsProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Debug(ctx, "Configuring XenServer Client")
	var data providerModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If practitioner provided a configuration value for any of the
	// attributes, it must be a known value.

	if data.Host.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Unknown XenServer API Host",
			"The provider cannot create the XenServer API client as there is an unknown configuration value for the XenServer API host. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the XENSERVER_HOST environment variable.",
		)
	}
	if data.Username.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Unknown XenServer API Username",
			"The provider cannot create the XenServer API client as there is an unknown configuration value for the XenServer API username. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the XENSERVER_USERNAME environment variable.",
		)
	}
	if data.Password.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Unknown XenServer API Password",
			"The provider cannot create the XenServer API client as there is an unknown configuration value for the XenServer API password. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the XENSERVER_PASSWORD environment variable.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	host := os.Getenv("XENSERVER_HOST")
	username := os.Getenv("XENSERVER_USERNAME")
	password := os.Getenv("XENSERVER_PASSWORD")

	if !data.Host.IsNull() {
		host = data.Host.ValueString()
	}
	if !data.Username.IsNull() {
		username = data.Username.ValueString()
	}
	if !data.Password.IsNull() {
		password = data.Password.ValueString()
	}

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.

	if host == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Missing XenServer API Host",
			"The provider cannot create the XenServer API client as there is a missing or empty value for the XenServer API host. "+
				"Set the host value in the configuration or use the XENSERVER_HOST environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}
	if username == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Missing XenServer API Username",
			"The provider cannot create the XenServer API client as there is a missing or empty value for the XenServer API username. "+
				"Set the username value in the configuration or use the XENSERVER_USERNAME environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}
	if password == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Missing XenServer API Password",
			"The provider cannot create the XenServer API client as there is a missing or empty value for the XenServer API password. "+
				"Set the password value in the configuration or use the XENSERVER_PASSWORD environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "host", host)
	ctx = tflog.SetField(ctx, "username", username)
	ctx = tflog.SetField(ctx, "password", password)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "password")
	tflog.Debug(ctx, "Creating XenServer API session")

	session := xenapi.NewSession(&xenapi.ClientOpts{
		URL: host,
		Headers: map[string]string{
			"User-Agent": "XS SDK for Go v1.0",
		},
	})
	_, err := session.LoginWithPassword(username, password, "1.0", "terraform provider")
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create XENSERVER API client",
			"An unexpected error occurred when creating the XENSERVER API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"XENSERVER client Error: "+err.Error(),
		)
		return
	}

	resp.DataSourceData = session
	resp.ResourceData = session
}

func (p *xsProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewVMResource,
		NewNetworkResource,
	}
}

func (p *xsProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewPIFDataSource,
		NewSRDataSource,
	}
}

func (p *xsProvider) Functions(_ context.Context) []func() function.Function {
	return nil
	// return []func() function.Function{
	// 	NewExampleFunction,
	// }
}
