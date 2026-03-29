package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rslarson/terraform-provider-unifi/internal/client"
)

var _ datasource.DataSource = &FirewallPolicyDataSource{}

type FirewallPolicyDataSource struct {
	client *client.Client
}

type FirewallPolicyDataSourceModel struct {
	ID                             types.String `tfsdk:"id"`
	SiteID                         types.String `tfsdk:"site_id"`
	Enabled                        types.Bool   `tfsdk:"enabled"`
	Name                           types.String `tfsdk:"name"`
	Description                    types.String `tfsdk:"description"`
	ActionType                     types.String `tfsdk:"action_type"`
	AllowReturnTraffic             types.Bool   `tfsdk:"allow_return_traffic"`
	SourceZoneID                   types.String `tfsdk:"source_zone_id"`
	SourceTrafficFilterType        types.String `tfsdk:"source_traffic_filter_type"`
	SourceTrafficFilterValues      types.List   `tfsdk:"source_traffic_filter_values"`
	DestinationZoneID              types.String `tfsdk:"destination_zone_id"`
	DestinationTrafficFilterType   types.String `tfsdk:"destination_traffic_filter_type"`
	DestinationTrafficFilterValues types.List   `tfsdk:"destination_traffic_filter_values"`
	IPVersion                      types.String `tfsdk:"ip_version"`
	ConnectionStateFilter          types.List   `tfsdk:"connection_state_filter"`
	IpsecFilter                    types.String `tfsdk:"ipsec_filter"`
	LoggingEnabled                 types.Bool   `tfsdk:"logging_enabled"`
}

func NewFirewallPolicyDataSource() datasource.DataSource {
	return &FirewallPolicyDataSource{}
}

func (d *FirewallPolicyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall_policy"
}

func (d *FirewallPolicyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches details of a UniFi firewall policy.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the firewall policy.",
				Required:    true,
			},
			"site_id": schema.StringAttribute{
				Description: "The site ID.",
				Required:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the firewall policy is enabled.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name of the firewall policy.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description of the firewall policy.",
				Computed:    true,
			},
			"action_type": schema.StringAttribute{
				Description: "Action type: ALLOW, BLOCK, or REJECT.",
				Computed:    true,
			},
			"allow_return_traffic": schema.BoolAttribute{
				Description: "Whether return traffic is allowed.",
				Computed:    true,
			},
			"source_zone_id": schema.StringAttribute{
				Description: "Source firewall zone ID.",
				Computed:    true,
			},
			"source_traffic_filter_type": schema.StringAttribute{
				Description: "Source traffic filter type.",
				Computed:    true,
			},
			"source_traffic_filter_values": schema.ListAttribute{
				Description: "Source traffic filter values.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"destination_zone_id": schema.StringAttribute{
				Description: "Destination firewall zone ID.",
				Computed:    true,
			},
			"destination_traffic_filter_type": schema.StringAttribute{
				Description: "Destination traffic filter type.",
				Computed:    true,
			},
			"destination_traffic_filter_values": schema.ListAttribute{
				Description: "Destination traffic filter values.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"ip_version": schema.StringAttribute{
				Description: "IP version.",
				Computed:    true,
			},
			"connection_state_filter": schema.ListAttribute{
				Description: "Connection state filter.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"ipsec_filter": schema.StringAttribute{
				Description: "IPsec filter.",
				Computed:    true,
			},
			"logging_enabled": schema.BoolAttribute{
				Description: "Whether logging is enabled.",
				Computed:    true,
			},
		},
	}
}

func (d *FirewallPolicyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := extractClient(req.ProviderData, "Data Source")
	resp.Diagnostics.Append(diags...)
	if c != nil {
		d.client = c
	}
}

func (d *FirewallPolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config FirewallPolicyDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.GetFirewallPolicy(ctx, config.SiteID.ValueString(), config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading firewall policy", err.Error())
		return
	}

	var resourceModel FirewallPolicyResourceModel
	resourceModel.ID = config.ID
	resourceModel.SiteID = config.SiteID
	diags := firewallPolicyAPIToModel(ctx, result, &resourceModel)
	resp.Diagnostics.Append(diags...)

	config.Enabled = resourceModel.Enabled
	config.Name = resourceModel.Name
	config.Description = resourceModel.Description
	config.ActionType = resourceModel.ActionType
	config.AllowReturnTraffic = resourceModel.AllowReturnTraffic
	config.SourceZoneID = resourceModel.SourceZoneID
	config.SourceTrafficFilterType = resourceModel.SourceTrafficFilterType
	config.SourceTrafficFilterValues = resourceModel.SourceTrafficFilterValues
	config.DestinationZoneID = resourceModel.DestinationZoneID
	config.DestinationTrafficFilterType = resourceModel.DestinationTrafficFilterType
	config.DestinationTrafficFilterValues = resourceModel.DestinationTrafficFilterValues
	config.IPVersion = resourceModel.IPVersion
	config.ConnectionStateFilter = resourceModel.ConnectionStateFilter
	config.IpsecFilter = resourceModel.IpsecFilter
	config.LoggingEnabled = resourceModel.LoggingEnabled

	resp.Diagnostics.Append(resp.State.Set(ctx, config)...)
}
