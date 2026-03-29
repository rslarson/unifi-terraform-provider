package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rslarson/terraform-provider-unifi/internal/client"
)

var _ datasource.DataSource = &WifiBroadcastDataSource{}

type WifiBroadcastDataSource struct {
	client *client.Client
}

type WifiBroadcastDataSourceModel struct {
	ID                                  types.String `tfsdk:"id"`
	SiteID                              types.String `tfsdk:"site_id"`
	Type                                types.String `tfsdk:"type"`
	Name                                types.String `tfsdk:"name"`
	Enabled                             types.Bool   `tfsdk:"enabled"`
	SecurityType                        types.String `tfsdk:"security_type"`
	NetworkType                         types.String `tfsdk:"network_type"`
	NetworkID                           types.String `tfsdk:"network_id"`
	ClientIsolationEnabled              types.Bool   `tfsdk:"client_isolation_enabled"`
	HideName                            types.Bool   `tfsdk:"hide_name"`
	MulticastToUnicastConversionEnabled types.Bool   `tfsdk:"multicast_to_unicast_conversion_enabled"`
	UapsdEnabled                        types.Bool   `tfsdk:"uapsd_enabled"`
	BasicDataRateGHz24                  types.Int64  `tfsdk:"basic_data_rate_24ghz"`
	BasicDataRateGHz5                   types.Int64  `tfsdk:"basic_data_rate_5ghz"`
	ClientFilterAction                  types.String `tfsdk:"client_filter_action"`
	ClientFilterMacAddresses            types.List   `tfsdk:"client_filter_mac_addresses"`
	BlackoutScheduleDays                types.List   `tfsdk:"blackout_schedule_days"`
	BroadcastingFrequenciesGHz          types.List   `tfsdk:"broadcasting_frequencies_ghz"`
	BroadcastingDeviceFilterType        types.String `tfsdk:"broadcasting_device_filter_type"`
	BroadcastingDeviceFilterIds         types.List   `tfsdk:"broadcasting_device_filter_ids"`
	MulticastFilterAction               types.String `tfsdk:"multicast_filter_action"`
	MdnsProxyMode                       types.String `tfsdk:"mdns_proxy_mode"`
	BandSteeringEnabled                 types.Bool   `tfsdk:"band_steering_enabled"`
	MloEnabled                          types.Bool   `tfsdk:"mlo_enabled"`
	ArpProxyEnabled                     types.Bool   `tfsdk:"arp_proxy_enabled"`
	BssTransitionEnabled                types.Bool   `tfsdk:"bss_transition_enabled"`
	AdvertiseDeviceName                 types.Bool   `tfsdk:"advertise_device_name"`
}

func NewWifiBroadcastDataSource() datasource.DataSource {
	return &WifiBroadcastDataSource{}
}

func (d *WifiBroadcastDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_wifi_broadcast"
}

func (d *WifiBroadcastDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches details of a UniFi WiFi broadcast.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the WiFi broadcast.",
				Required:    true,
			},
			"site_id": schema.StringAttribute{
				Description: "The site ID.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "Broadcast type.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "SSID name.",
				Computed:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the broadcast is enabled.",
				Computed:    true,
			},
			"security_type": schema.StringAttribute{
				Description: "Security configuration type.",
				Computed:    true,
			},
			"network_type": schema.StringAttribute{
				Description: "Network assignment type.",
				Computed:    true,
			},
			"network_id": schema.StringAttribute{
				Description: "Associated network ID.",
				Computed:    true,
			},
			"client_isolation_enabled": schema.BoolAttribute{
				Description: "Whether client isolation is enabled.",
				Computed:    true,
			},
			"hide_name": schema.BoolAttribute{
				Description: "Whether the SSID is hidden.",
				Computed:    true,
			},
			"multicast_to_unicast_conversion_enabled": schema.BoolAttribute{
				Description: "Whether multicast to unicast conversion is enabled.",
				Computed:    true,
			},
			"uapsd_enabled": schema.BoolAttribute{
				Description: "Whether U-APSD is enabled.",
				Computed:    true,
			},
			"basic_data_rate_24ghz": schema.Int64Attribute{
				Description: "Basic data rate for 2.4 GHz in Kbps.",
				Computed:    true,
			},
			"basic_data_rate_5ghz": schema.Int64Attribute{
				Description: "Basic data rate for 5 GHz in Kbps.",
				Computed:    true,
			},
			"client_filter_action": schema.StringAttribute{
				Description: "Client filtering policy action.",
				Computed:    true,
			},
			"client_filter_mac_addresses": schema.ListAttribute{
				Description: "List of MAC addresses for the client filtering policy.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"blackout_schedule_days": schema.ListNestedAttribute{
				Description: "Blackout schedule configuration days.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Description: "Schedule type: ALL_DAY or TIME_RANGE.",
							Computed:    true,
						},
						"day": schema.StringAttribute{
							Description: "Day of week.",
							Computed:    true,
						},
						"start_time": schema.StringAttribute{
							Description: "Start time in HH:mm format.",
							Computed:    true,
						},
						"end_time": schema.StringAttribute{
							Description: "End time in HH:mm format.",
							Computed:    true,
						},
					},
				},
			},
			"broadcasting_frequencies_ghz": schema.ListAttribute{
				Description: "Broadcasting frequencies in GHz.",
				Computed:    true,
				ElementType: types.Float64Type,
			},
			"broadcasting_device_filter_type": schema.StringAttribute{
				Description: "Broadcasting device filter type.",
				Computed:    true,
			},
			"broadcasting_device_filter_ids": schema.ListAttribute{
				Description: "List of device or device tag IDs for the broadcasting device filter.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"multicast_filter_action": schema.StringAttribute{
				Description: "Multicast filtering policy action.",
				Computed:    true,
			},
			"mdns_proxy_mode": schema.StringAttribute{
				Description: "mDNS proxy configuration mode.",
				Computed:    true,
			},
			"band_steering_enabled": schema.BoolAttribute{
				Description: "Whether band steering is enabled.",
				Computed:    true,
			},
			"mlo_enabled": schema.BoolAttribute{
				Description: "Whether MLO (Multi-Link Operation) is enabled.",
				Computed:    true,
			},
			"arp_proxy_enabled": schema.BoolAttribute{
				Description: "Whether ARP proxy is enabled.",
				Computed:    true,
			},
			"bss_transition_enabled": schema.BoolAttribute{
				Description: "Whether BSS transition (802.11v) is enabled.",
				Computed:    true,
			},
			"advertise_device_name": schema.BoolAttribute{
				Description: "Whether to advertise the device name.",
				Computed:    true,
			},
		},
	}
}

func (d *WifiBroadcastDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := extractClient(req.ProviderData, "Data Source")
	resp.Diagnostics.Append(diags...)
	if c != nil {
		d.client = c
	}
}

func (d *WifiBroadcastDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config WifiBroadcastDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.GetWifiBroadcast(ctx, config.SiteID.ValueString(), config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading WiFi broadcast", err.Error())
		return
	}

	resp.Diagnostics.Append(wifiBroadcastAPIToDataSourceModel(ctx, &config, result)...)

	resp.Diagnostics.Append(resp.State.Set(ctx, config)...)
}
