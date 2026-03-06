package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rslarson/terraform-provider-unifi/internal/client"
)

var _ datasource.DataSource = &SitesDataSource{}

type SitesDataSource struct {
	client *client.Client
}

type SitesDataSourceModel struct {
	Sites []SiteModel `tfsdk:"sites"`
}

type SiteModel struct {
	ID                types.String `tfsdk:"id"`
	InternalReference types.String `tfsdk:"internal_reference"`
	Name              types.String `tfsdk:"name"`
}

func NewSitesDataSource() datasource.DataSource {
	return &SitesDataSource{}
}

func (d *SitesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sites"
}

func (d *SitesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches all sites managed by the UniFi console.",
		Attributes: map[string]schema.Attribute{
			"sites": schema.ListNestedAttribute{
				Description: "List of sites.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Site ID.",
							Computed:    true,
						},
						"internal_reference": schema.StringAttribute{
							Description: "Internal reference identifier.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Site name.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *SitesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SitesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	sites, err := d.client.ListSites(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing sites", err.Error())
		return
	}

	var state SitesDataSourceModel
	for _, s := range sites {
		state.Sites = append(state.Sites, SiteModel{
			ID:                types.StringValue(s.ID),
			InternalReference: types.StringValue(s.InternalReference),
			Name:              types.StringValue(s.Name),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
