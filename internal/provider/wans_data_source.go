package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rslarson/terraform-provider-unifi/internal/client"
)

var _ datasource.DataSource = &WansDataSource{}

type WansDataSource struct {
	client *client.Client
}

type WansDataSourceModel struct {
	SiteID types.String   `tfsdk:"site_id"`
	Wans   []WanItemModel `tfsdk:"wans"`
}

type WanItemModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func NewWansDataSource() datasource.DataSource {
	return &WansDataSource{}
}

func (d *WansDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_wans"
}

func (d *WansDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches all WAN interfaces for a UniFi site.",
		Attributes: map[string]schema.Attribute{
			"site_id": schema.StringAttribute{
				Description: "The site ID.",
				Required:    true,
			},
			"wans": schema.ListNestedAttribute{
				Description: "List of WAN interfaces.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "WAN interface ID.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "WAN interface name.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *WansDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := extractClient(req.ProviderData, "Data Source")
	resp.Diagnostics.Append(diags...)
	if c != nil {
		d.client = c
	}
}

func (d *WansDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config WansDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wans, err := d.client.ListWans(ctx, config.SiteID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing WANs", err.Error())
		return
	}

	state := WansDataSourceModel{
		SiteID: config.SiteID,
		Wans:   make([]WanItemModel, 0, len(wans)),
	}
	for _, w := range wans {
		state.Wans = append(state.Wans, WanItemModel{
			ID:   types.StringValue(w.ID),
			Name: types.StringValue(w.Name),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
