package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rslarson/terraform-provider-unifi/internal/client"
)

var _ datasource.DataSource = &DeviceTagsDataSource{}

type DeviceTagsDataSource struct {
	client *client.Client
}

type DeviceTagsDataSourceModel struct {
	SiteID     types.String          `tfsdk:"site_id"`
	DeviceTags []DeviceTagItemModel  `tfsdk:"device_tags"`
}

type DeviceTagItemModel struct {
	ID        types.String   `tfsdk:"id"`
	Name      types.String   `tfsdk:"name"`
	DeviceIDs []types.String `tfsdk:"device_ids"`
}

func NewDeviceTagsDataSource() datasource.DataSource {
	return &DeviceTagsDataSource{}
}

func (d *DeviceTagsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device_tags"
}

func (d *DeviceTagsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches all device tags for a UniFi site.",
		Attributes: map[string]schema.Attribute{
			"site_id": schema.StringAttribute{
				Description: "The site ID.",
				Required:    true,
			},
			"device_tags": schema.ListNestedAttribute{
				Description: "List of device tags.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Device tag ID.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Device tag name.",
							Computed:    true,
						},
						"device_ids": schema.ListAttribute{
							Description: "List of device IDs associated with this tag.",
							Computed:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
		},
	}
}

func (d *DeviceTagsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := extractClient(req.ProviderData, "Data Source")
	resp.Diagnostics.Append(diags...)
	if c != nil {
		d.client = c
	}
}

func (d *DeviceTagsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DeviceTagsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tags, err := d.client.ListDeviceTags(ctx, config.SiteID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing device tags", err.Error())
		return
	}

	state := DeviceTagsDataSourceModel{
		SiteID:     config.SiteID,
		DeviceTags: make([]DeviceTagItemModel, 0, len(tags)),
	}
	for _, t := range tags {
		ids := make([]types.String, 0, len(t.DeviceIDs))
		for _, id := range t.DeviceIDs {
			ids = append(ids, types.StringValue(id))
		}
		state.DeviceTags = append(state.DeviceTags, DeviceTagItemModel{
			ID:        types.StringValue(t.ID),
			Name:      types.StringValue(t.Name),
			DeviceIDs: ids,
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
