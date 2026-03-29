package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rslarson/terraform-provider-unifi/internal/client"
)

var _ datasource.DataSource = &ClientsDataSource{}

type ClientsDataSource struct {
	client *client.Client
}

type ClientsDataSourceModel struct {
	SiteID  types.String       `tfsdk:"site_id"`
	Clients []ClientItemModel  `tfsdk:"clients"`
}

type ClientItemModel struct {
	ID        types.String `tfsdk:"id"`
	Type      types.String `tfsdk:"type"`
	Name      types.String `tfsdk:"name"`
	IPAddress types.String `tfsdk:"ip_address"`
}

func NewClientsDataSource() datasource.DataSource {
	return &ClientsDataSource{}
}

func (d *ClientsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_clients"
}

func (d *ClientsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches all connected clients for a UniFi site.",
		Attributes: map[string]schema.Attribute{
			"site_id": schema.StringAttribute{
				Description: "The site ID.",
				Required:    true,
			},
			"clients": schema.ListNestedAttribute{
				Description: "List of connected clients.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Client ID.",
							Computed:    true,
						},
						"type": schema.StringAttribute{
							Description: "Client connection type: WIRED, WIRELESS, VPN, or TELEPORT.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Client name.",
							Computed:    true,
						},
						"ip_address": schema.StringAttribute{
							Description: "IP address of the client.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *ClientsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := extractClient(req.ProviderData, "Data Source")
	resp.Diagnostics.Append(diags...)
	if c != nil {
		d.client = c
	}
}

func (d *ClientsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ClientsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clients, err := d.client.ListClients(ctx, config.SiteID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing clients", err.Error())
		return
	}

	state := ClientsDataSourceModel{
		SiteID:  config.SiteID,
		Clients: make([]ClientItemModel, 0, len(clients)),
	}
	for _, c := range clients {
		state.Clients = append(state.Clients, ClientItemModel{
			ID:        types.StringValue(c.ID),
			Type:      types.StringValue(c.Type),
			Name:      types.StringValue(c.Name),
			IPAddress: types.StringValue(c.IPAddress),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
