package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rslarson/terraform-provider-unifi/internal/client"
)

var _ datasource.DataSource = &VpnTunnelsDataSource{}

type VpnTunnelsDataSource struct {
	client *client.Client
}

type VpnTunnelsDataSourceModel struct {
	SiteID     types.String          `tfsdk:"site_id"`
	VpnTunnels []VpnTunnelItemModel  `tfsdk:"vpn_tunnels"`
}

type VpnTunnelItemModel struct {
	ID   types.String `tfsdk:"id"`
	Type types.String `tfsdk:"type"`
	Name types.String `tfsdk:"name"`
}

func NewVpnTunnelsDataSource() datasource.DataSource {
	return &VpnTunnelsDataSource{}
}

func (d *VpnTunnelsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpn_tunnels"
}

func (d *VpnTunnelsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches all VPN site-to-site tunnels for a UniFi site.",
		Attributes: map[string]schema.Attribute{
			"site_id": schema.StringAttribute{
				Description: "The site ID.",
				Required:    true,
			},
			"vpn_tunnels": schema.ListNestedAttribute{
				Description: "List of VPN tunnels.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "VPN tunnel ID.",
							Computed:    true,
						},
						"type": schema.StringAttribute{
							Description: "VPN tunnel type.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "VPN tunnel name.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *VpnTunnelsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := extractClient(req.ProviderData, "Data Source")
	resp.Diagnostics.Append(diags...)
	if c != nil {
		d.client = c
	}
}

func (d *VpnTunnelsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config VpnTunnelsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tunnels, err := d.client.ListVpnTunnels(ctx, config.SiteID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing VPN tunnels", err.Error())
		return
	}

	state := VpnTunnelsDataSourceModel{
		SiteID:     config.SiteID,
		VpnTunnels: make([]VpnTunnelItemModel, 0, len(tunnels)),
	}
	for _, t := range tunnels {
		state.VpnTunnels = append(state.VpnTunnels, VpnTunnelItemModel{
			ID:   types.StringValue(t.ID),
			Type: types.StringValue(t.Type),
			Name: types.StringValue(t.Name),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
