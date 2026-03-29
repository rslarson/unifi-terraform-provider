package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rslarson/terraform-provider-unifi/internal/client"
)

var _ datasource.DataSource = &TrafficMatchingListDataSource{}

type TrafficMatchingListDataSource struct {
	client *client.Client
}

type TrafficMatchingListDataSourceModel struct {
	ID     types.String `tfsdk:"id"`
	SiteID types.String `tfsdk:"site_id"`
	Type   types.String `tfsdk:"type"`
	Name   types.String `tfsdk:"name"`
	Items  types.List   `tfsdk:"items"`
}

func NewTrafficMatchingListDataSource() datasource.DataSource {
	return &TrafficMatchingListDataSource{}
}

func (d *TrafficMatchingListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_traffic_matching_list"
}

func (d *TrafficMatchingListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches details of a UniFi traffic matching list.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the traffic matching list.",
				Required:    true,
			},
			"site_id": schema.StringAttribute{
				Description: "The site ID.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "Traffic matching list type.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name of the traffic matching list.",
				Computed:    true,
			},
			"items": schema.ListNestedAttribute{
				Description: "List of traffic matching items.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Description: "Item type.",
							Computed:    true,
						},
						"value": schema.StringAttribute{
							Description: "IP address value.",
							Computed:    true,
						},
						"subnet": schema.StringAttribute{
							Description: "Subnet value.",
							Computed:    true,
						},
						"start": schema.StringAttribute{
							Description: "Start of IP address range.",
							Computed:    true,
						},
						"end": schema.StringAttribute{
							Description: "End of IP address range.",
							Computed:    true,
						},
						"port_number": schema.Int64Attribute{
							Description: "Port number.",
							Computed:    true,
						},
						"start_port": schema.Int64Attribute{
							Description: "Start port of range.",
							Computed:    true,
						},
						"end_port": schema.Int64Attribute{
							Description: "End port of range.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *TrafficMatchingListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := extractClient(req.ProviderData, "Data Source")
	resp.Diagnostics.Append(diags...)
	if c != nil {
		d.client = c
	}
}

func (d *TrafficMatchingListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config TrafficMatchingListDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.GetTrafficMatchingList(ctx, config.SiteID.ValueString(), config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading traffic matching list", err.Error())
		return
	}

	var resourceModel TrafficMatchingListResourceModel
	resourceModel.ID = config.ID
	resourceModel.SiteID = config.SiteID
	diags := trafficMatchingListAPIToModel(ctx, result, &resourceModel)
	resp.Diagnostics.Append(diags...)

	config.Type = resourceModel.Type
	config.Name = resourceModel.Name
	config.Items = resourceModel.Items

	resp.Diagnostics.Append(resp.State.Set(ctx, config)...)
}
