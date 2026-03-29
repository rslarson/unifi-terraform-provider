package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rslarson/terraform-provider-unifi/internal/client"
)

var _ datasource.DataSource = &NetworkDataSource{}

type NetworkDataSource struct {
	client *client.Client
}

type NetworkDataSourceModel struct {
	ID                           types.String `tfsdk:"id"`
	SiteID                       types.String `tfsdk:"site_id"`
	Name                         types.String `tfsdk:"name"`
	Management                   types.String `tfsdk:"management"`
	Enabled                      types.Bool   `tfsdk:"enabled"`
	VlanID                       types.Int64  `tfsdk:"vlan_id"`
	TrustedDhcpServerIPAddresses types.List   `tfsdk:"trusted_dhcp_server_ip_addresses"`

	// Gateway and Switch managed fields.
	IsolationEnabled      types.Bool `tfsdk:"isolation_enabled"`
	CellularBackupEnabled types.Bool `tfsdk:"cellular_backup_enabled"`

	// Gateway only fields.
	InternetAccessEnabled types.Bool   `tfsdk:"internet_access_enabled"`
	MdnsForwardingEnabled types.Bool   `tfsdk:"mdns_forwarding_enabled"`
	ZoneID                types.String `tfsdk:"zone_id"`

	// Switch only fields.
	DeviceID types.String `tfsdk:"device_id"`

	// IPv4 configuration.
	IPv4Configuration types.Object `tfsdk:"ipv4_configuration"`
}

func NewNetworkDataSource() datasource.DataSource {
	return &NetworkDataSource{}
}

func (d *NetworkDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network"
}

func (d *NetworkDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches details of a UniFi network.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the network.",
				Required:    true,
			},
			"site_id": schema.StringAttribute{
				Description: "The site ID where the network exists.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name of the network.",
				Computed:    true,
			},
			"management": schema.StringAttribute{
				Description: "Management type: UNMANAGED, GATEWAY, or SWITCH.",
				Computed:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the network is enabled.",
				Computed:    true,
			},
			"vlan_id": schema.Int64Attribute{
				Description: "VLAN ID.",
				Computed:    true,
			},
			"trusted_dhcp_server_ip_addresses": schema.ListAttribute{
				Description: "List of trusted DHCP server IP addresses.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"isolation_enabled": schema.BoolAttribute{
				Description: "Whether network isolation is enabled.",
				Computed:    true,
			},
			"cellular_backup_enabled": schema.BoolAttribute{
				Description: "Whether cellular backup is enabled.",
				Computed:    true,
			},
			"internet_access_enabled": schema.BoolAttribute{
				Description: "Whether internet access is enabled. Gateway managed networks only.",
				Computed:    true,
			},
			"mdns_forwarding_enabled": schema.BoolAttribute{
				Description: "Whether mDNS forwarding is enabled. Gateway managed networks only.",
				Computed:    true,
			},
			"zone_id": schema.StringAttribute{
				Description: "The firewall zone ID. Gateway managed networks only.",
				Computed:    true,
			},
			"device_id": schema.StringAttribute{
				Description: "The L3 switch device UUID. Switch managed networks only.",
				Computed:    true,
			},
			"ipv4_configuration": schema.SingleNestedAttribute{
				Description: "IPv4 configuration for the network.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"auto_scale_enabled": schema.BoolAttribute{
						Description: "Whether auto-scaling of the subnet is enabled.",
						Computed:    true,
					},
					"host_ip_address": schema.StringAttribute{
						Description: "The host IP address (gateway address) for this network.",
						Computed:    true,
					},
					"prefix_length": schema.Int64Attribute{
						Description: "The subnet prefix length.",
						Computed:    true,
					},
					"dhcp_mode": schema.StringAttribute{
						Description: "DHCP mode: SERVER or RELAY.",
						Computed:    true,
					},
					"dhcp_start": schema.StringAttribute{
						Description: "DHCP range start IP address.",
						Computed:    true,
					},
					"dhcp_stop": schema.StringAttribute{
						Description: "DHCP range stop IP address.",
						Computed:    true,
					},
					"dhcp_lease_time_seconds": schema.Int64Attribute{
						Description: "DHCP lease time in seconds.",
						Computed:    true,
					},
					"dhcp_dns_servers": schema.ListAttribute{
						Description: "List of DNS server IP addresses for DHCP.",
						Computed:    true,
						ElementType: types.StringType,
					},
					"dhcp_gateway_override": schema.StringAttribute{
						Description: "Override the gateway IP address sent to DHCP clients.",
						Computed:    true,
					},
					"dhcp_domain_name": schema.StringAttribute{
						Description: "Domain name for DHCP clients.",
						Computed:    true,
					},
					"dhcp_relay_addresses": schema.ListAttribute{
						Description: "List of DHCP relay server IP addresses.",
						Computed:    true,
						ElementType: types.StringType,
					},
				},
			},
		},
	}
}

func (d *NetworkDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := extractClient(req.ProviderData, "Data Source")
	resp.Diagnostics.Append(diags...)
	if c != nil {
		d.client = c
	}
}

func (d *NetworkDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config NetworkDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.GetNetwork(ctx, config.SiteID.ValueString(), config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading network", err.Error())
		return
	}

	// Preserve identity fields.
	id := config.ID
	siteID := config.SiteID
	diags := networkAPIToDataSourceModel(ctx, result, &config)
	config.ID = id
	config.SiteID = siteID
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, config)...)
}
