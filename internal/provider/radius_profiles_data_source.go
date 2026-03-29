package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rslarson/terraform-provider-unifi/internal/client"
)

var _ datasource.DataSource = &RadiusProfilesDataSource{}

type RadiusProfilesDataSource struct {
	client *client.Client
}

type RadiusProfilesDataSourceModel struct {
	SiteID         types.String             `tfsdk:"site_id"`
	RadiusProfiles []RadiusProfileItemModel `tfsdk:"radius_profiles"`
}

type RadiusProfileItemModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func NewRadiusProfilesDataSource() datasource.DataSource {
	return &RadiusProfilesDataSource{}
}

func (d *RadiusProfilesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_radius_profiles"
}

func (d *RadiusProfilesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches all RADIUS profiles for a UniFi site.",
		Attributes: map[string]schema.Attribute{
			"site_id": schema.StringAttribute{
				Description: "The site ID.",
				Required:    true,
			},
			"radius_profiles": schema.ListNestedAttribute{
				Description: "List of RADIUS profiles.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "RADIUS profile ID.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "RADIUS profile name.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *RadiusProfilesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := extractClient(req.ProviderData, "Data Source")
	resp.Diagnostics.Append(diags...)
	if c != nil {
		d.client = c
	}
}

func (d *RadiusProfilesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config RadiusProfilesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profiles, err := d.client.ListRadiusProfiles(ctx, config.SiteID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing RADIUS profiles", err.Error())
		return
	}

	state := RadiusProfilesDataSourceModel{
		SiteID:         config.SiteID,
		RadiusProfiles: make([]RadiusProfileItemModel, 0, len(profiles)),
	}
	for _, p := range profiles {
		state.RadiusProfiles = append(state.RadiusProfiles, RadiusProfileItemModel{
			ID:   types.StringValue(p.ID),
			Name: types.StringValue(p.Name),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
