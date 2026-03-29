package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rslarson/terraform-provider-unifi/internal/client"
)

var _ datasource.DataSource = &PendingDevicesDataSource{}

type PendingDevicesDataSource struct {
	client *client.Client
}

type PendingDevicesDataSourceModel struct {
	PendingDevices []PendingDeviceItemModel `tfsdk:"pending_devices"`
}

type PendingDeviceItemModel struct {
	MacAddress        types.String `tfsdk:"mac_address"`
	IPAddress         types.String `tfsdk:"ip_address"`
	Model             types.String `tfsdk:"model"`
	State             types.String `tfsdk:"state"`
	Supported         types.Bool   `tfsdk:"supported"`
	FirmwareVersion   types.String `tfsdk:"firmware_version"`
	FirmwareUpdatable types.Bool   `tfsdk:"firmware_updatable"`
}

func NewPendingDevicesDataSource() datasource.DataSource {
	return &PendingDevicesDataSource{}
}

func (d *PendingDevicesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pending_devices"
}

func (d *PendingDevicesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches all devices pending adoption on the UniFi console.",
		Attributes: map[string]schema.Attribute{
			"pending_devices": schema.ListNestedAttribute{
				Description: "List of devices pending adoption.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"mac_address": schema.StringAttribute{
							Description: "MAC address of the pending device.",
							Computed:    true,
						},
						"ip_address": schema.StringAttribute{
							Description: "IP address of the pending device.",
							Computed:    true,
						},
						"model": schema.StringAttribute{
							Description: "Device model identifier.",
							Computed:    true,
						},
						"state": schema.StringAttribute{
							Description: "Device state.",
							Computed:    true,
						},
						"supported": schema.BoolAttribute{
							Description: "Whether the device is supported.",
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
					},
				},
			},
		},
	}
}

func (d *PendingDevicesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := extractClient(req.ProviderData, "Data Source")
	resp.Diagnostics.Append(diags...)
	if c != nil {
		d.client = c
	}
}

func (d *PendingDevicesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	devices, err := d.client.ListPendingDevices(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing pending devices", err.Error())
		return
	}

	state := PendingDevicesDataSourceModel{
		PendingDevices: make([]PendingDeviceItemModel, 0, len(devices)),
	}
	for _, dev := range devices {
		state.PendingDevices = append(state.PendingDevices, PendingDeviceItemModel{
			MacAddress:        types.StringValue(dev.MacAddress),
			IPAddress:         types.StringValue(dev.IPAddress),
			Model:             types.StringValue(dev.Model),
			State:             types.StringValue(dev.State),
			Supported:         types.BoolValue(dev.Supported),
			FirmwareVersion:   types.StringValue(dev.FirmwareVersion),
			FirmwareUpdatable: types.BoolValue(dev.FirmwareUpdatable),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
