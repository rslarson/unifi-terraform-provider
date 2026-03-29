package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rslarson/terraform-provider-unifi/internal/client"
)

// ============================================================================
// DPI Categories Data Source
// ============================================================================

var _ datasource.DataSource = &DpiCategoriesDataSource{}

type DpiCategoriesDataSource struct {
	client *client.Client
}

type DpiCategoriesDataSourceModel struct {
	Categories []DpiCategoryItemModel `tfsdk:"categories"`
}

type DpiCategoryItemModel struct {
	ID   types.Int64  `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func NewDpiCategoriesDataSource() datasource.DataSource {
	return &DpiCategoriesDataSource{}
}

func (d *DpiCategoriesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dpi_categories"
}

func (d *DpiCategoriesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches all DPI (Deep Packet Inspection) categories.",
		Attributes: map[string]schema.Attribute{
			"categories": schema.ListNestedAttribute{
				Description: "List of DPI categories.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Description: "DPI category ID.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "DPI category name.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *DpiCategoriesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := extractClient(req.ProviderData, "Data Source")
	resp.Diagnostics.Append(diags...)
	if c != nil {
		d.client = c
	}
}

func (d *DpiCategoriesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	categories, err := d.client.ListDpiCategories(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing DPI categories", err.Error())
		return
	}

	state := DpiCategoriesDataSourceModel{
		Categories: make([]DpiCategoryItemModel, 0, len(categories)),
	}
	for _, cat := range categories {
		state.Categories = append(state.Categories, DpiCategoryItemModel{
			ID:   types.Int64Value(int64(cat.ID)),
			Name: types.StringValue(cat.Name),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// ============================================================================
// DPI Applications Data Source
// ============================================================================

var _ datasource.DataSource = &DpiApplicationsDataSource{}

type DpiApplicationsDataSource struct {
	client *client.Client
}

type DpiApplicationsDataSourceModel struct {
	Applications []DpiApplicationItemModel `tfsdk:"applications"`
}

type DpiApplicationItemModel struct {
	ID   types.Int64  `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func NewDpiApplicationsDataSource() datasource.DataSource {
	return &DpiApplicationsDataSource{}
}

func (d *DpiApplicationsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dpi_applications"
}

func (d *DpiApplicationsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches all DPI (Deep Packet Inspection) applications.",
		Attributes: map[string]schema.Attribute{
			"applications": schema.ListNestedAttribute{
				Description: "List of DPI applications.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Description: "DPI application ID.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "DPI application name.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *DpiApplicationsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := extractClient(req.ProviderData, "Data Source")
	resp.Diagnostics.Append(diags...)
	if c != nil {
		d.client = c
	}
}

func (d *DpiApplicationsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	apps, err := d.client.ListDpiApplications(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing DPI applications", err.Error())
		return
	}

	state := DpiApplicationsDataSourceModel{
		Applications: make([]DpiApplicationItemModel, 0, len(apps)),
	}
	for _, app := range apps {
		state.Applications = append(state.Applications, DpiApplicationItemModel{
			ID:   types.Int64Value(int64(app.ID)),
			Name: types.StringValue(app.Name),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
