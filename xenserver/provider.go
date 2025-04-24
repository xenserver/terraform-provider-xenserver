package xenserver

import (
	"context"
	"errors"
	"os"
	"strings"

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
var terraformProviderVersion string

// xsProvider defines the provider implementation.
type xsProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version         string
	session         *xenapi.Session
	sessionRef      xenapi.SessionRef
	coordinatorConf coordinatorConf
}

type coordinatorConf struct {
	Host     string
	Username string
	Password string
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
				Optional: true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "The password of target XenServer host." + "<br />" +
					"Can be set by using the environment variable **XENSERVER_PASSWORD**.",
				Optional:  true,
				Sensitive: true,
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

	terraformProviderVersion = p.version
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
			"Missing Host Configuration",
			"The provider cannot create the XenServer API client as there is a missing or empty value for the host. "+
				"Set the host value in the configuration or use the XENSERVER_HOST environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}
	if username == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Missing Username Configuration",
			"The provider cannot create the XenServer API client as there is a missing or empty value for the username. "+
				"Set the username value in the configuration or use the XENSERVER_USERNAME environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}
	if password == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Missing Password Configuration",
			"The provider cannot create the XenServer API client as there is a missing or empty value for the password. "+
				"Set the password value in the configuration or use the XENSERVER_PASSWORD environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "host", host)
	ctx = tflog.SetField(ctx, "username", username)
	tflog.Debug(ctx, "Creating XenServer API session")

	session, sessionRef, err := loginServer(host, username, password)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create XenServer API client",
			"An unexpected error occurred when creating the XenServer API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"XenServer client Error: "+err.Error(),
		)
		return
	}

	p.coordinatorConf.Host = host
	p.coordinatorConf.Username = username
	p.coordinatorConf.Password = password
	p.session = session
	p.sessionRef = sessionRef

	// the xsProvider type itself is made available for resources and data sources
	resp.DataSourceData = p
	resp.ResourceData = p
}

func loginServer(host string, username string, password string) (*xenapi.Session, xenapi.SessionRef, error) {
	// check if host, username, password are non-empty
	if host == "" || username == "" || password == "" {
		return nil, "", errors.New("host, username, password cannot be empty")
	}

	if !strings.HasPrefix(host, "http") {
		host = "https://" + host
	}

	session := xenapi.NewSession(&xenapi.ClientOpts{
		URL: host,
		Headers: map[string]string{
			"User-Agent": "XenServer Terraform Provider/" + terraformProviderVersion,
		},
	})

	sessionRef, err := session.LoginWithPassword(username, password, "1.0", "terraform provider")
	if err != nil {
		return nil, "", errors.New(err.Error())
	}

	return session, sessionRef, nil
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
