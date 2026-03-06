package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rslarson/terraform-provider-unifi/internal/client"
)

var _ datasource.DataSource = &DeviceDataSource{}

type DeviceDataSource struct {
	client *client.Client
}

type DeviceDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	SiteID            types.String `tfsdk:"site_id"`
	MacAddress        types.String `tfsdk:"mac_address"`
	IPAddress         types.String `tfsdk:"ip_address"`
	Name              types.String `tfsdk:"name"`
	Model             types.String `tfsdk:"model"`
	Supported         types.Bool   `tfsdk:"supported"`
	State             types.String `tfsdk:"state"`
	FirmwareVersion   types.String `tfsdk:"firmware_version"`
	FirmwareUpdatable types.Bool   `tfsdk:"firmware_updatable"`
	AdoptedAt         types.String `tfsdk:"adopted_at"`
	ProvisionedAt     types.String `tfsdk:"provisioned_at"`
	ConfigurationID   types.String `tfsdk:"configuration_id"`
	UplinkDeviceID    types.String `tfsdk:"uplink_device_id"`
}

func NewDeviceDataSource() datasource.DataSource {
	return &DeviceDataSource{}
}

func (d *DeviceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device"
}

func (d *DeviceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches details of an adopted UniFi device.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The device ID.",
				Required:    true,
			},
			"site_id": schema.StringAttribute{
				Description: "The site ID.",
				Required:    true,
			},
			"mac_address": schema.StringAttribute{
				Description: "MAC address of the device.",
				Computed:    true,
			},
			"ip_address": schema.StringAttribute{
				Description: "IP address of the device.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name of the device.",
				Computed:    true,
			},
			"model": schema.StringAttribute{
				Description: "Device model identifier.",
				Computed:    true,
			},
			"supported": schema.BoolAttribute{
				Description: "Whether the device is supported.",
				Computed:    true,
			},
			"state": schema.StringAttribute{
				Description: "Device state: ONLINE, OFFLINE, PENDING_ADOPTION, UPDATING, GETTING_READY, ADOPTING, DELETING, CONNECTION_INTERRUPTED, or ISOLATED.",
				Computed:    true,
			},
			"firmware_version": schema.StringAttribute{
				Description: "Current firmware version.",
				Computed:    true,
			},
			"firmware_updatable": schema.BoolAttribute{
				Description: "Whether the firmware can be updated.",
				Computed:    true,
			},
			"adopted_at": schema.StringAttribute{
				Description: "Timestamp when the device was adopted.",
				Computed:    true,
			},
			"provisioned_at": schema.StringAttribute{
				Description: "Timestamp when the device was last provisioned.",
				Computed:    true,
			},
			"configuration_id": schema.StringAttribute{
				Description: "Configuration identifier.",
				Computed:    true,
			},
			"uplink_device_id": schema.StringAttribute{
				Description: "ID of the parent/uplink device.",
				Computed:    true,
			},
		},
	}
}

func (d *DeviceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := extractClient(req.ProviderData, "Data Source")
	resp.Diagnostics.Append(diags...)
	if c != nil {
		d.client = c
	}
}

func (d *DeviceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DeviceDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.GetDevice(ctx, config.SiteID.ValueString(), config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading device", err.Error())
		return
	}

	deviceAPIToModel(&config, result)

	resp.Diagnostics.Append(resp.State.Set(ctx, config)...)
}
