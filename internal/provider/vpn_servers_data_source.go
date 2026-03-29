package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rslarson/terraform-provider-unifi/internal/client"
)

var _ datasource.DataSource = &VpnServersDataSource{}

type VpnServersDataSource struct {
	client *client.Client
}

type VpnServersDataSourceModel struct {
	SiteID     types.String         `tfsdk:"site_id"`
	VpnServers []VpnServerItemModel `tfsdk:"vpn_servers"`
}

type VpnServerItemModel struct {
	ID      types.String `tfsdk:"id"`
	Type    types.String `tfsdk:"type"`
	Name    types.String `tfsdk:"name"`
	Enabled types.Bool   `tfsdk:"enabled"`
}

func NewVpnServersDataSource() datasource.DataSource {
	return &VpnServersDataSource{}
}

func (d *VpnServersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpn_servers"
}

func (d *VpnServersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches all VPN servers for a UniFi site.",
		Attributes: map[string]schema.Attribute{
			"site_id": schema.StringAttribute{
				Description: "The site ID.",
				Required:    true,
			},
			"vpn_servers": schema.ListNestedAttribute{
				Description: "List of VPN servers.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "VPN server ID.",
							Computed:    true,
						},
						"type": schema.StringAttribute{
							Description: "VPN server type.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "VPN server name.",
							Computed:    true,
						},
						"enabled": schema.BoolAttribute{
							Description: "Whether the VPN server is enabled.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *VpnServersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := extractClient(req.ProviderData, "Data Source")
	resp.Diagnostics.Append(diags...)
	if c != nil {
		d.client = c
	}
}

func (d *VpnServersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config VpnServersDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	servers, err := d.client.ListVpnServers(ctx, config.SiteID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing VPN servers", err.Error())
		return
	}

	state := VpnServersDataSourceModel{
		SiteID:     config.SiteID,
		VpnServers: make([]VpnServerItemModel, 0, len(servers)),
	}
	for _, s := range servers {
		state.VpnServers = append(state.VpnServers, VpnServerItemModel{
			ID:      types.StringValue(s.ID),
			Type:    types.StringValue(s.Type),
			Name:    types.StringValue(s.Name),
			Enabled: types.BoolValue(s.Enabled),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
