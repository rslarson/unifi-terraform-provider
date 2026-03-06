package provider

import (
	"context"
	"fmt"

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
		},
	}
}

func (d *WifiBroadcastDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	d.client = c
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

	config.Type = types.StringValue(result.Type)
	config.Name = types.StringValue(result.Name)
	config.Enabled = types.BoolValue(result.Enabled)
	config.ClientIsolationEnabled = types.BoolValue(result.ClientIsolationEnabled)
	config.HideName = types.BoolValue(result.HideName)
	config.MulticastToUnicastConversionEnabled = types.BoolValue(result.MulticastToUnicastConversionEnabled)
	config.UapsdEnabled = types.BoolValue(result.UapsdEnabled)

	if result.SecurityConfiguration != nil {
		config.SecurityType = types.StringValue(result.SecurityConfiguration.Type)
	}
	if result.Network != nil {
		config.NetworkType = types.StringValue(result.Network.Type)
		if result.Network.NetworkID != "" {
			config.NetworkID = types.StringValue(result.Network.NetworkID)
		} else {
			config.NetworkID = types.StringNull()
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, config)...)
}
