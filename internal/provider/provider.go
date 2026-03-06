package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rslarson/terraform-provider-unifi/internal/client"
)

var _ provider.Provider = &UnifiProvider{}

type UnifiProvider struct {
	version string
}

type UnifiProviderModel struct {
	APIKey types.String `tfsdk:"api_key"`
	HostID types.String `tfsdk:"host_id"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &UnifiProvider{
			version: version,
		}
	}
}

func (p *UnifiProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "unifi"
	resp.Version = p.version
}

func (p *UnifiProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for managing UniFi Network devices via the UI API.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Description: "API key for authenticating with the UniFi API. Can also be set via the UNIFI_API_KEY environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"host_id": schema.StringAttribute{
				Description: "Host ID of the UniFi console for cloud connector proxying. Can also be set via the UNIFI_HOST_ID environment variable.",
				Optional:    true,
			},
		},
	}
}

func (p *UnifiProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config UnifiProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiKey := os.Getenv("UNIFI_API_KEY")
	if !config.APIKey.IsNull() {
		apiKey = config.APIKey.ValueString()
	}

	hostID := os.Getenv("UNIFI_HOST_ID")
	if !config.HostID.IsNull() {
		hostID = config.HostID.ValueString()
	}

	if apiKey == "" {
		resp.Diagnostics.AddError(
			"Missing API Key",
			"The provider requires an API key. Set it in the provider configuration or via the UNIFI_API_KEY environment variable.",
		)
		return
	}

	if hostID == "" {
		resp.Diagnostics.AddError(
			"Missing Host ID",
			"The provider requires a host ID. Set it in the provider configuration or via the UNIFI_HOST_ID environment variable.",
		)
		return
	}

	c := client.NewClient(apiKey, hostID)
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *UnifiProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewNetworkResource,
		NewWifiBroadcastResource,
		NewFirewallZoneResource,
	}
}

func (p *UnifiProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewNetworkDataSource,
		NewWifiBroadcastDataSource,
		NewFirewallZoneDataSource,
		NewDeviceDataSource,
		NewSitesDataSource,
	}
}
