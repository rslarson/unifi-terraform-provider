package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rslarson/terraform-provider-unifi/internal/client"
)

var _ datasource.DataSource = &DnsPolicyDataSource{}

type DnsPolicyDataSource struct {
	client *client.Client
}

type DnsPolicyDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	SiteID      types.String `tfsdk:"site_id"`
	Type        types.String `tfsdk:"type"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	Name        types.String `tfsdk:"name"`
	Domain      types.String `tfsdk:"domain"`
	IPv4Address types.String `tfsdk:"ipv4_address"`
	IPv6Address types.String `tfsdk:"ipv6_address"`
	Target      types.String `tfsdk:"target"`
	TTLSeconds  types.Int64  `tfsdk:"ttl_seconds"`
	Priority    types.Int64  `tfsdk:"priority"`
	Weight      types.Int64  `tfsdk:"weight"`
	Port        types.Int64  `tfsdk:"port"`
	TxtValue    types.String `tfsdk:"txt_value"`
	ForwardTo   types.List   `tfsdk:"forward_to"`
}

func NewDnsPolicyDataSource() datasource.DataSource {
	return &DnsPolicyDataSource{}
}

func (d *DnsPolicyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_policy"
}

func (d *DnsPolicyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches details of a UniFi DNS policy.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the DNS policy.",
				Required:    true,
			},
			"site_id": schema.StringAttribute{
				Description: "The site ID.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "DNS policy type.",
				Computed:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the DNS policy is enabled.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name of the DNS policy.",
				Computed:    true,
			},
			"domain": schema.StringAttribute{
				Description: "Domain name.",
				Computed:    true,
			},
			"ipv4_address": schema.StringAttribute{
				Description: "IPv4 address.",
				Computed:    true,
			},
			"ipv6_address": schema.StringAttribute{
				Description: "IPv6 address.",
				Computed:    true,
			},
			"target": schema.StringAttribute{
				Description: "Target hostname.",
				Computed:    true,
			},
			"ttl_seconds": schema.Int64Attribute{
				Description: "TTL in seconds.",
				Computed:    true,
			},
			"priority": schema.Int64Attribute{
				Description: "Priority.",
				Computed:    true,
			},
			"weight": schema.Int64Attribute{
				Description: "Weight.",
				Computed:    true,
			},
			"port": schema.Int64Attribute{
				Description: "Port.",
				Computed:    true,
			},
			"txt_value": schema.StringAttribute{
				Description: "TXT record value.",
				Computed:    true,
			},
			"forward_to": schema.ListAttribute{
				Description: "List of DNS servers to forward to.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *DnsPolicyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := extractClient(req.ProviderData, "Data Source")
	resp.Diagnostics.Append(diags...)
	if c != nil {
		d.client = c
	}
}

func (d *DnsPolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DnsPolicyDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.GetDnsPolicy(ctx, config.SiteID.ValueString(), config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading DNS policy", err.Error())
		return
	}

	var resourceModel DnsPolicyResourceModel
	resourceModel.ID = config.ID
	resourceModel.SiteID = config.SiteID
	diags := dnsPolicyAPIToModel(ctx, result, &resourceModel)
	resp.Diagnostics.Append(diags...)

	config.Type = resourceModel.Type
	config.Enabled = resourceModel.Enabled
	config.Name = resourceModel.Name
	config.Domain = resourceModel.Domain
	config.IPv4Address = resourceModel.IPv4Address
	config.IPv6Address = resourceModel.IPv6Address
	config.Target = resourceModel.Target
	config.TTLSeconds = resourceModel.TTLSeconds
	config.Priority = resourceModel.Priority
	config.Weight = resourceModel.Weight
	config.Port = resourceModel.Port
	config.TxtValue = resourceModel.TxtValue
	config.ForwardTo = resourceModel.ForwardTo

	resp.Diagnostics.Append(resp.State.Set(ctx, config)...)
}
