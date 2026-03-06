package provider

import (
	"context"
	"fmt"

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
		},
	}
}

func (d *NetworkDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	config.Name = types.StringValue(result.Name)
	config.Management = types.StringValue(result.Management)
	config.Enabled = types.BoolValue(result.Enabled)
	config.VlanID = types.Int64Value(int64(result.VlanID))

	if result.DhcpGuarding != nil && len(result.DhcpGuarding.TrustedDhcpServerIPAddresses) > 0 {
		ips, diags := types.ListValueFrom(ctx, types.StringType, result.DhcpGuarding.TrustedDhcpServerIPAddresses)
		resp.Diagnostics.Append(diags...)
		config.TrustedDhcpServerIPAddresses = ips
	} else {
		config.TrustedDhcpServerIPAddresses = types.ListNull(types.StringType)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, config)...)
}
