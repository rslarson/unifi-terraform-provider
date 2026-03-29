package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rslarson/terraform-provider-unifi/internal/client"
)

var (
	_ resource.Resource                = &NetworkResource{}
	_ resource.ResourceWithImportState = &NetworkResource{}
)

type NetworkResource struct {
	client *client.Client
}

type NetworkResourceModel struct {
	ID                           types.String `tfsdk:"id"`
	SiteID                       types.String `tfsdk:"site_id"`
	Name                         types.String `tfsdk:"name"`
	Management                   types.String `tfsdk:"management"`
	Enabled                      types.Bool   `tfsdk:"enabled"`
	VlanID                       types.Int64  `tfsdk:"vlan_id"`
	TrustedDhcpServerIPAddresses types.List   `tfsdk:"trusted_dhcp_server_ip_addresses"`

	// Gateway and Switch managed fields.
	IsolationEnabled      types.Bool `tfsdk:"isolation_enabled"`
	CellularBackupEnabled types.Bool `tfsdk:"cellular_backup_enabled"`

	// Gateway only fields.
	InternetAccessEnabled types.Bool   `tfsdk:"internet_access_enabled"`
	MdnsForwardingEnabled types.Bool   `tfsdk:"mdns_forwarding_enabled"`
	ZoneID                types.String `tfsdk:"zone_id"`

	// Switch only fields.
	DeviceID types.String `tfsdk:"device_id"`

	// IPv4 configuration.
	IPv4Configuration types.Object `tfsdk:"ipv4_configuration"`
}

// IPv4ConfigurationModel represents the ipv4_configuration nested attribute.
type IPv4ConfigurationModel struct {
	AutoScaleEnabled     types.Bool   `tfsdk:"auto_scale_enabled"`
	HostIPAddress        types.String `tfsdk:"host_ip_address"`
	PrefixLength         types.Int64  `tfsdk:"prefix_length"`
	DhcpMode             types.String `tfsdk:"dhcp_mode"`
	DhcpStart            types.String `tfsdk:"dhcp_start"`
	DhcpStop             types.String `tfsdk:"dhcp_stop"`
	DhcpLeaseTimeSeconds types.Int64  `tfsdk:"dhcp_lease_time_seconds"`
	DhcpDnsServers       types.List   `tfsdk:"dhcp_dns_servers"`
	DhcpGatewayOverride  types.String `tfsdk:"dhcp_gateway_override"`
	DhcpDomainName       types.String `tfsdk:"dhcp_domain_name"`
	DhcpRelayAddresses   types.List   `tfsdk:"dhcp_relay_addresses"`
}

// ipv4ConfigurationAttrTypes returns the attribute types for the ipv4_configuration object.
func ipv4ConfigurationAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"auto_scale_enabled":      types.BoolType,
		"host_ip_address":         types.StringType,
		"prefix_length":           types.Int64Type,
		"dhcp_mode":               types.StringType,
		"dhcp_start":              types.StringType,
		"dhcp_stop":               types.StringType,
		"dhcp_lease_time_seconds": types.Int64Type,
		"dhcp_dns_servers":        types.ListType{ElemType: types.StringType},
		"dhcp_gateway_override":   types.StringType,
		"dhcp_domain_name":        types.StringType,
		"dhcp_relay_addresses":    types.ListType{ElemType: types.StringType},
	}
}

func NewNetworkResource() resource.Resource {
	return &NetworkResource{}
}

func (r *NetworkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network"
}

func (r *NetworkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a UniFi network.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the network.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site_id": schema.StringAttribute{
				Description: "The site ID where the network is managed.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the network.",
				Required:    true,
			},
			"management": schema.StringAttribute{
				Description: "Management type: UNMANAGED, GATEWAY, or SWITCH.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(client.ManagementUnmanaged, client.ManagementGateway, client.ManagementSwitch),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the network is enabled.",
				Required:    true,
			},
			"vlan_id": schema.Int64Attribute{
				Description: "VLAN ID (1-4009).",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.Between(1, 4009),
				},
			},
			"trusted_dhcp_server_ip_addresses": schema.ListAttribute{
				Description: "List of trusted DHCP server IP addresses for DHCP guarding.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"isolation_enabled": schema.BoolAttribute{
				Description: "Whether network isolation is enabled. Applicable to Gateway and Switch managed networks.",
				Optional:    true,
				Computed:    true,
			},
			"cellular_backup_enabled": schema.BoolAttribute{
				Description: "Whether cellular backup is enabled. Applicable to Gateway and Switch managed networks.",
				Optional:    true,
				Computed:    true,
			},
			"internet_access_enabled": schema.BoolAttribute{
				Description: "Whether internet access is enabled. Gateway managed networks only.",
				Optional:    true,
				Computed:    true,
			},
			"mdns_forwarding_enabled": schema.BoolAttribute{
				Description: "Whether mDNS forwarding is enabled. Gateway managed networks only.",
				Optional:    true,
				Computed:    true,
			},
			"zone_id": schema.StringAttribute{
				Description: "The firewall zone ID to associate with this network. Gateway managed networks only.",
				Optional:    true,
			},
			"device_id": schema.StringAttribute{
				Description: "The L3 switch device UUID. Switch managed networks only.",
				Optional:    true,
			},
			"ipv4_configuration": schema.SingleNestedAttribute{
				Description: "IPv4 configuration for the network. Applicable to Gateway and Switch managed networks.",
				Optional:    true,
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"auto_scale_enabled": schema.BoolAttribute{
						Description: "Whether auto-scaling of the subnet is enabled.",
						Required:    true,
					},
					"host_ip_address": schema.StringAttribute{
						Description: "The host IP address (gateway address) for this network.",
						Required:    true,
					},
					"prefix_length": schema.Int64Attribute{
						Description: "The subnet prefix length (8-30).",
						Required:    true,
						Validators: []validator.Int64{
							int64validator.Between(8, 30),
						},
					},
					"dhcp_mode": schema.StringAttribute{
						Description: "DHCP mode: SERVER or RELAY.",
						Optional:    true,
						Validators: []validator.String{
							stringvalidator.OneOf(client.DhcpModeServer, client.DhcpModeRelay),
						},
					},
					"dhcp_start": schema.StringAttribute{
						Description: "DHCP range start IP address. Used when dhcp_mode is SERVER.",
						Optional:    true,
					},
					"dhcp_stop": schema.StringAttribute{
						Description: "DHCP range stop IP address. Used when dhcp_mode is SERVER.",
						Optional:    true,
					},
					"dhcp_lease_time_seconds": schema.Int64Attribute{
						Description: "DHCP lease time in seconds. Used when dhcp_mode is SERVER.",
						Optional:    true,
					},
					"dhcp_dns_servers": schema.ListAttribute{
						Description: "List of DNS server IP addresses for DHCP. Used when dhcp_mode is SERVER.",
						Optional:    true,
						ElementType: types.StringType,
					},
					"dhcp_gateway_override": schema.StringAttribute{
						Description: "Override the gateway IP address sent to DHCP clients. Used when dhcp_mode is SERVER.",
						Optional:    true,
					},
					"dhcp_domain_name": schema.StringAttribute{
						Description: "Domain name for DHCP clients. Used when dhcp_mode is SERVER.",
						Optional:    true,
					},
					"dhcp_relay_addresses": schema.ListAttribute{
						Description: "List of DHCP relay server IP addresses. Used when dhcp_mode is RELAY.",
						Optional:    true,
						ElementType: types.StringType,
					},
				},
			},
		},
	}
}

func (r *NetworkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := extractClient(req.ProviderData, "Resource")
	resp.Diagnostics.Append(diags...)
	if c != nil {
		r.client = c
	}
}

func (r *NetworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan NetworkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	network, diags := networkModelToAPI(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.CreateNetwork(ctx, plan.SiteID.ValueString(), network)
	if err != nil {
		resp.Diagnostics.AddError("Error creating network", err.Error())
		return
	}

	state := NetworkResourceModel{
		ID:     types.StringValue(result.ID),
		SiteID: plan.SiteID,
	}
	diags = networkAPIToModelFull(ctx, result, &state)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *NetworkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NetworkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetNetwork(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading network", err.Error())
		return
	}

	// Preserve identity fields.
	id := state.ID
	siteID := state.SiteID
	diags := networkAPIToModelFull(ctx, result, &state)
	state.ID = id
	state.SiteID = siteID
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *NetworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan NetworkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var currentState NetworkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &currentState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	network, diags := networkModelToAPI(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.UpdateNetwork(ctx, plan.SiteID.ValueString(), currentState.ID.ValueString(), network)
	if err != nil {
		resp.Diagnostics.AddError("Error updating network", err.Error())
		return
	}

	state := NetworkResourceModel{
		ID:     currentState.ID,
		SiteID: plan.SiteID,
	}
	diags = networkAPIToModelFull(ctx, result, &state)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *NetworkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NetworkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteNetwork(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting network", err.Error())
		return
	}
}

func (r *NetworkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importCompositeState(ctx, req, resp)
}
