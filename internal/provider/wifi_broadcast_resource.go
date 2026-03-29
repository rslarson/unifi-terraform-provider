package provider

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
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

// wifiPassphraseRegex matches strings containing only printable ASCII
// characters (0x20 through 0x7E) as required by IEEE 802.11i for WPA
// passphrases.
var wifiPassphraseRegex = regexp.MustCompile(`^[\x20-\x7E]+$`)

// passphraseValidators returns the shared validators for both passphrase and
// passphrase_wo attributes: mutual exclusion, length bounds, and character set.
func passphraseValidators(conflictsWith path.Expression) []validator.String {
	return []validator.String{
		stringvalidator.ConflictsWith(conflictsWith),
		stringvalidator.LengthBetween(8, 63),
		stringvalidator.RegexMatches(wifiPassphraseRegex, "must contain only printable ASCII characters (spaces through tilde)"),
	}
}

var (
	_ resource.Resource                   = &WifiBroadcastResource{}
	_ resource.ResourceWithImportState    = &WifiBroadcastResource{}
	_ resource.ResourceWithValidateConfig = &WifiBroadcastResource{}
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
	PassphraseWO                        types.String `tfsdk:"passphrase_wo"`
	PassphraseWOVersion                 types.Int64  `tfsdk:"passphrase_wo_version"`
	NetworkType                         types.String `tfsdk:"network_type"`
	NetworkID                           types.String `tfsdk:"network_id"`
	ClientIsolationEnabled              types.Bool   `tfsdk:"client_isolation_enabled"`
	HideName                            types.Bool   `tfsdk:"hide_name"`
	MulticastToUnicastConversionEnabled types.Bool   `tfsdk:"multicast_to_unicast_conversion_enabled"`
	UapsdEnabled                        types.Bool   `tfsdk:"uapsd_enabled"`
	BasicDataRateGHz24                  types.Int64  `tfsdk:"basic_data_rate_24ghz"`
	BasicDataRateGHz5                   types.Int64  `tfsdk:"basic_data_rate_5ghz"`
	ClientFilterAction                  types.String `tfsdk:"client_filter_action"`
	ClientFilterMacAddresses            types.List   `tfsdk:"client_filter_mac_addresses"`
	BlackoutScheduleDays                types.List   `tfsdk:"blackout_schedule_days"`
	BroadcastingFrequenciesGHz          types.List   `tfsdk:"broadcasting_frequencies_ghz"`
	BroadcastingDeviceFilterType        types.String `tfsdk:"broadcasting_device_filter_type"`
	BroadcastingDeviceFilterIds         types.List   `tfsdk:"broadcasting_device_filter_ids"`
	MulticastFilterAction               types.String `tfsdk:"multicast_filter_action"`
	MdnsProxyMode                       types.String `tfsdk:"mdns_proxy_mode"`
	BandSteeringEnabled                 types.Bool   `tfsdk:"band_steering_enabled"`
	MloEnabled                          types.Bool   `tfsdk:"mlo_enabled"`
	ArpProxyEnabled                     types.Bool   `tfsdk:"arp_proxy_enabled"`
	BssTransitionEnabled                types.Bool   `tfsdk:"bss_transition_enabled"`
	AdvertiseDeviceName                 types.Bool   `tfsdk:"advertise_device_name"`
}

// BlackoutScheduleDayModel represents a single blackout schedule day entry.
type BlackoutScheduleDayModel struct {
	Type      types.String `tfsdk:"type"`
	Day       types.String `tfsdk:"day"`
	StartTime types.String `tfsdk:"start_time"`
	EndTime   types.String `tfsdk:"end_time"`
}

// blackoutScheduleDayAttrTypes returns the attribute types for the blackout_schedule_days nested object.
func blackoutScheduleDayAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"type":       types.StringType,
		"day":        types.StringType,
		"start_time": types.StringType,
		"end_time":   types.StringType,
	}
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
					stringvalidator.OneOf(client.BroadcastTypeStandard, client.BroadcastTypeIoTOptimized),
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
						client.SecurityOpen, client.SecurityWPA2Personal, client.SecurityWPA3Personal,
						client.SecurityWPA2WPA3Personal, client.SecurityWPA2Enterprise,
						client.SecurityWPA3Enterprise, client.SecurityWPA2WPA3Enterprise,
					),
				},
			},
			"passphrase": schema.StringAttribute{
				Description: "WiFi passphrase. Required for personal security types. Must be 8-63 printable ASCII characters per IEEE 802.11i. Prefer passphrase_wo to avoid storing the value in state.",
				Optional:    true,
				Sensitive:   true,
				Validators:  passphraseValidators(path.MatchRoot("passphrase_wo")),
			},
			"passphrase_wo": schema.StringAttribute{
				Description: "Write-only WiFi passphrase. Required for personal security types. Must be 8-63 printable ASCII characters per IEEE 802.11i. The value is never stored in the Terraform state or plan files. Use passphrase_wo_version to trigger updates when the passphrase changes.",
				Optional:    true,
				WriteOnly:   true,
				Validators:  passphraseValidators(path.MatchRoot("passphrase")),
			},
			"passphrase_wo_version": schema.Int64Attribute{
				Description: "An integer that tracks changes to passphrase_wo. Increment this value to signal that passphrase_wo has changed and should be re-applied.",
				Optional:    true,
			},
			"network_type": schema.StringAttribute{
				Description: "Network assignment type: NATIVE or SPECIFIC.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(client.NetworkTypeNative, client.NetworkTypeSpecific),
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
			"basic_data_rate_24ghz": schema.Int64Attribute{
				Description: "Basic data rate for 2.4 GHz in Kbps. Valid values: 1000, 2000, 5500, 6000, 9000, 11000, 12000, 24000.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.OneOf(1000, 2000, 5500, 6000, 9000, 11000, 12000, 24000),
				},
			},
			"basic_data_rate_5ghz": schema.Int64Attribute{
				Description: "Basic data rate for 5 GHz in Kbps. Valid values: 6000, 9000, 12000, 24000.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.OneOf(6000, 9000, 12000, 24000),
				},
			},
			"client_filter_action": schema.StringAttribute{
				Description: "Client filtering policy action: ALLOW or BLOCK.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(client.FilterActionAllow, client.FilterActionBlock),
				},
			},
			"client_filter_mac_addresses": schema.ListAttribute{
				Description: "List of MAC addresses for the client filtering policy.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"blackout_schedule_days": schema.ListNestedAttribute{
				Description: "Blackout schedule configuration days.",
				Optional:    true,
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Description: "Schedule type: ALL_DAY or TIME_RANGE.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf(client.BlackoutAllDay, client.BlackoutTimeRange),
							},
						},
						"day": schema.StringAttribute{
							Description: "Day of week: SUN, MON, TUE, WED, THU, FRI, SAT.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("SUN", "MON", "TUE", "WED", "THU", "FRI", "SAT"),
							},
						},
						"start_time": schema.StringAttribute{
							Description: "Start time in HH:mm format. Required when type is TIME_RANGE.",
							Optional:    true,
						},
						"end_time": schema.StringAttribute{
							Description: "End time in HH:mm format. Required when type is TIME_RANGE.",
							Optional:    true,
						},
					},
				},
			},
			"broadcasting_frequencies_ghz": schema.ListAttribute{
				Description: "Broadcasting frequencies in GHz. STANDARD type only. Valid values: 2.4, 5, 6.",
				Optional:    true,
				Computed:    true,
				ElementType: types.Float64Type,
			},
			"broadcasting_device_filter_type": schema.StringAttribute{
				Description: "Broadcasting device filter type: DEVICES or DEVICE_TAGS.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(client.DeviceFilterDevices, client.DeviceFilterDeviceTags),
				},
			},
			"broadcasting_device_filter_ids": schema.ListAttribute{
				Description: "List of device IDs or device tag IDs for the broadcasting device filter.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"multicast_filter_action": schema.StringAttribute{
				Description: "Multicast filtering policy action: ALLOW or BLOCK.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(client.FilterActionAllow, client.FilterActionBlock),
				},
			},
			"mdns_proxy_mode": schema.StringAttribute{
				Description: "mDNS proxy configuration mode: AUTO or CUSTOM.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(client.MdnsProxyAuto, client.MdnsProxyCustom),
				},
			},
			"band_steering_enabled": schema.BoolAttribute{
				Description: "Whether band steering is enabled. STANDARD type only.",
				Optional:    true,
				Computed:    true,
			},
			"mlo_enabled": schema.BoolAttribute{
				Description: "Whether MLO (Multi-Link Operation) is enabled. STANDARD type only.",
				Optional:    true,
				Computed:    true,
			},
			"arp_proxy_enabled": schema.BoolAttribute{
				Description: "Whether ARP proxy is enabled. STANDARD type only.",
				Optional:    true,
				Computed:    true,
			},
			"bss_transition_enabled": schema.BoolAttribute{
				Description: "Whether BSS transition (802.11v) is enabled. STANDARD type only.",
				Optional:    true,
				Computed:    true,
			},
			"advertise_device_name": schema.BoolAttribute{
				Description: "Whether to advertise the device name. STANDARD type only.",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func (r *WifiBroadcastResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config WifiBroadcastResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Warn if passphrase_wo is set without passphrase_wo_version.
	if !config.PassphraseWO.IsNull() && config.PassphraseWOVersion.IsNull() {
		resp.Diagnostics.AddWarning(
			"Missing passphrase_wo_version",
			"passphrase_wo is set but passphrase_wo_version is not. Without a version, "+
				"Terraform cannot detect passphrase changes. Set passphrase_wo_version and "+
				"increment it whenever the passphrase changes.",
		)
	}

	// Validate that personal security types have a passphrase, and non-personal
	// types do not.
	if !config.SecurityType.IsNull() && !config.SecurityType.IsUnknown() {
		hasPassphrase := !config.Passphrase.IsNull() || !config.PassphraseWO.IsNull()
		isPersonal := client.IsPersonalSecurityType(config.SecurityType.ValueString())
		if isPersonal && !hasPassphrase {
			resp.Diagnostics.AddError(
				"Missing passphrase",
				"Security type "+config.SecurityType.ValueString()+" requires a passphrase. "+
					"Set either passphrase or passphrase_wo.",
			)
		} else if !isPersonal && hasPassphrase {
			resp.Diagnostics.AddError(
				"Unexpected passphrase",
				"Security type "+config.SecurityType.ValueString()+" does not use a passphrase. "+
					"Remove the passphrase or passphrase_wo attribute.",
			)
		}
	}
}

func (r *WifiBroadcastResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := extractClient(req.ProviderData, "Resource")
	resp.Diagnostics.Append(diags...)
	if c != nil {
		r.client = c
	}
}

func (r *WifiBroadcastResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan WifiBroadcastResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read the write-only passphrase_wo value from the config, since the
	// framework nulls it out in plan/state.
	var passphraseWO types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("passphrase_wo"), &passphraseWO)...)
	if resp.Diagnostics.HasError() {
		return
	}

	broadcast, diags := wifiBroadcastModelToAPI(ctx, plan, passphraseWO)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.CreateWifiBroadcast(ctx, plan.SiteID.ValueString(), broadcast)
	if err != nil {
		resp.Diagnostics.AddError("Error creating WiFi broadcast", err.Error())
		return
	}

	plan.ID = types.StringValue(result.ID)
	resp.Diagnostics.Append(wifiBroadcastAPIToModel(ctx, &plan, result)...)

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
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading WiFi broadcast", err.Error())
		return
	}

	resp.Diagnostics.Append(wifiBroadcastAPIToModel(ctx, &state, result)...)

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

	// Read the write-only passphrase_wo value from the config.
	var passphraseWO types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("passphrase_wo"), &passphraseWO)...)
	if resp.Diagnostics.HasError() {
		return
	}

	broadcast, diags := wifiBroadcastModelToAPI(ctx, plan, passphraseWO)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.UpdateWifiBroadcast(ctx, plan.SiteID.ValueString(), state.ID.ValueString(), broadcast)
	if err != nil {
		resp.Diagnostics.AddError("Error updating WiFi broadcast", err.Error())
		return
	}

	plan.ID = state.ID
	resp.Diagnostics.Append(wifiBroadcastAPIToModel(ctx, &plan, result)...)

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
	importCompositeState(ctx, req, resp)
}
