package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rslarson/terraform-provider-unifi/internal/client"
)

var (
	_ resource.Resource                = &WifiBroadcastResource{}
	_ resource.ResourceWithImportState = &WifiBroadcastResource{}
)

type WifiBroadcastResource struct {
	client *client.Client
}

type WifiBroadcastResourceModel struct {
	ID                                  types.String `tfsdk:"id"`
	SiteID                              types.String `tfsdk:"site_id"`
	Type                                types.String `tfsdk:"type"`
	Name                                types.String `tfsdk:"name"`
	Enabled                             types.Bool   `tfsdk:"enabled"`
	SecurityType                        types.String `tfsdk:"security_type"`
	Passphrase                          types.String `tfsdk:"passphrase"`
	NetworkType                         types.String `tfsdk:"network_type"`
	NetworkID                           types.String `tfsdk:"network_id"`
	ClientIsolationEnabled              types.Bool   `tfsdk:"client_isolation_enabled"`
	HideName                            types.Bool   `tfsdk:"hide_name"`
	MulticastToUnicastConversionEnabled types.Bool   `tfsdk:"multicast_to_unicast_conversion_enabled"`
	UapsdEnabled                        types.Bool   `tfsdk:"uapsd_enabled"`
}

func NewWifiBroadcastResource() resource.Resource {
	return &WifiBroadcastResource{}
}

func (r *WifiBroadcastResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_wifi_broadcast"
}

func (r *WifiBroadcastResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a UniFi WiFi broadcast (SSID).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the WiFi broadcast.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site_id": schema.StringAttribute{
				Description: "The site ID where the WiFi broadcast is managed.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Description: "Broadcast type: STANDARD or IOT_OPTIMIZED.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("STANDARD", "IOT_OPTIMIZED"),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the WiFi broadcast (SSID name).",
				Required:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the WiFi broadcast is enabled.",
				Required:    true,
			},
			"security_type": schema.StringAttribute{
				Description: "Security type: OPEN, WPA2_PERSONAL, WPA3_PERSONAL, WPA2_WPA3_PERSONAL, WPA2_ENTERPRISE, WPA3_ENTERPRISE, or WPA2_WPA3_ENTERPRISE.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						"OPEN", "WPA2_PERSONAL", "WPA3_PERSONAL", "WPA2_WPA3_PERSONAL",
						"WPA2_ENTERPRISE", "WPA3_ENTERPRISE", "WPA2_WPA3_ENTERPRISE",
					),
				},
			},
			"passphrase": schema.StringAttribute{
				Description: "WiFi passphrase. Required for personal security types.",
				Optional:    true,
				Sensitive:   true,
			},
			"network_type": schema.StringAttribute{
				Description: "Network assignment type: NATIVE or SPECIFIC.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("NATIVE", "SPECIFIC"),
				},
			},
			"network_id": schema.StringAttribute{
				Description: "Network ID when network_type is SPECIFIC.",
				Optional:    true,
			},
			"client_isolation_enabled": schema.BoolAttribute{
				Description: "Whether client isolation is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"hide_name": schema.BoolAttribute{
				Description: "Whether to hide the SSID name.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"multicast_to_unicast_conversion_enabled": schema.BoolAttribute{
				Description: "Whether multicast to unicast conversion is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"uapsd_enabled": schema.BoolAttribute{
				Description: "Whether U-APSD (Unscheduled Automatic Power Save Delivery) is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

func (r *WifiBroadcastResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *WifiBroadcastResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan WifiBroadcastResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	broadcast := r.modelToAPI(plan)

	result, err := r.client.CreateWifiBroadcast(ctx, plan.SiteID.ValueString(), broadcast)
	if err != nil {
		resp.Diagnostics.AddError("Error creating WiFi broadcast", err.Error())
		return
	}

	plan.ID = types.StringValue(result.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *WifiBroadcastResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state WifiBroadcastResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetWifiBroadcast(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok && apiErr.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading WiFi broadcast", err.Error())
		return
	}

	state.Type = types.StringValue(result.Type)
	state.Name = types.StringValue(result.Name)
	state.Enabled = types.BoolValue(result.Enabled)
	state.ClientIsolationEnabled = types.BoolValue(result.ClientIsolationEnabled)
	state.HideName = types.BoolValue(result.HideName)
	state.MulticastToUnicastConversionEnabled = types.BoolValue(result.MulticastToUnicastConversionEnabled)
	state.UapsdEnabled = types.BoolValue(result.UapsdEnabled)

	if result.SecurityConfiguration != nil {
		state.SecurityType = types.StringValue(result.SecurityConfiguration.Type)
	}
	if result.Network != nil {
		state.NetworkType = types.StringValue(result.Network.Type)
		if result.Network.NetworkID != "" {
			state.NetworkID = types.StringValue(result.Network.NetworkID)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *WifiBroadcastResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan WifiBroadcastResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state WifiBroadcastResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	broadcast := r.modelToAPI(plan)

	_, err := r.client.UpdateWifiBroadcast(ctx, plan.SiteID.ValueString(), state.ID.ValueString(), broadcast)
	if err != nil {
		resp.Diagnostics.AddError("Error updating WiFi broadcast", err.Error())
		return
	}

	plan.ID = state.ID
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *WifiBroadcastResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state WifiBroadcastResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteWifiBroadcast(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting WiFi broadcast", err.Error())
		return
	}
}

func (r *WifiBroadcastResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *WifiBroadcastResource) modelToAPI(plan WifiBroadcastResourceModel) *client.WifiBroadcast {
	broadcast := &client.WifiBroadcast{
		Type:                                plan.Type.ValueString(),
		Name:                                plan.Name.ValueString(),
		Enabled:                             plan.Enabled.ValueBool(),
		ClientIsolationEnabled:              plan.ClientIsolationEnabled.ValueBool(),
		HideName:                            plan.HideName.ValueBool(),
		MulticastToUnicastConversionEnabled: plan.MulticastToUnicastConversionEnabled.ValueBool(),
		UapsdEnabled:                        plan.UapsdEnabled.ValueBool(),
		SecurityConfiguration: &client.SecurityConfiguration{
			Type: plan.SecurityType.ValueString(),
		},
		Network: &client.BroadcastNetwork{
			Type: plan.NetworkType.ValueString(),
		},
	}

	if !plan.Passphrase.IsNull() && !plan.Passphrase.IsUnknown() {
		broadcast.SecurityConfiguration.Passphrase = plan.Passphrase.ValueString()
	}

	if !plan.NetworkID.IsNull() && !plan.NetworkID.IsUnknown() {
		broadcast.Network.NetworkID = plan.NetworkID.ValueString()
	}

	return broadcast
}
