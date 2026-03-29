package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rslarson/terraform-provider-unifi/internal/client"
)

var (
	_ resource.Resource                = &FirewallPolicyResource{}
	_ resource.ResourceWithImportState = &FirewallPolicyResource{}
)

type FirewallPolicyResource struct {
	client *client.Client
}

type FirewallPolicyResourceModel struct {
	ID                          types.String `tfsdk:"id"`
	SiteID                      types.String `tfsdk:"site_id"`
	Enabled                     types.Bool   `tfsdk:"enabled"`
	Name                        types.String `tfsdk:"name"`
	Description                 types.String `tfsdk:"description"`
	ActionType                  types.String `tfsdk:"action_type"`
	AllowReturnTraffic          types.Bool   `tfsdk:"allow_return_traffic"`
	SourceZoneID                types.String `tfsdk:"source_zone_id"`
	SourceTrafficFilterType     types.String `tfsdk:"source_traffic_filter_type"`
	SourceTrafficFilterValues   types.List   `tfsdk:"source_traffic_filter_values"`
	DestinationZoneID           types.String `tfsdk:"destination_zone_id"`
	DestinationTrafficFilterType   types.String `tfsdk:"destination_traffic_filter_type"`
	DestinationTrafficFilterValues types.List   `tfsdk:"destination_traffic_filter_values"`
	IPVersion                   types.String `tfsdk:"ip_version"`
	ConnectionStateFilter       types.List   `tfsdk:"connection_state_filter"`
	IpsecFilter                 types.String `tfsdk:"ipsec_filter"`
	LoggingEnabled              types.Bool   `tfsdk:"logging_enabled"`
}

func NewFirewallPolicyResource() resource.Resource {
	return &FirewallPolicyResource{}
}

func (r *FirewallPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall_policy"
}

func (r *FirewallPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a UniFi firewall policy.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the firewall policy.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site_id": schema.StringAttribute{
				Description: "The site ID where the firewall policy is managed.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the firewall policy is enabled.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name of the firewall policy.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description of the firewall policy.",
				Optional:    true,
			},
			"action_type": schema.StringAttribute{
				Description: "Action type: ALLOW, BLOCK, or REJECT.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(client.ActionAllow, client.ActionBlock, client.ActionReject),
				},
			},
			"allow_return_traffic": schema.BoolAttribute{
				Description: "Whether to allow return traffic (only when action_type is ALLOW).",
				Optional:    true,
			},
			"source_zone_id": schema.StringAttribute{
				Description: "Source firewall zone ID.",
				Required:    true,
			},
			"source_traffic_filter_type": schema.StringAttribute{
				Description: "Source traffic filter type: NETWORK, IP_ADDRESS, MAC_ADDRESS, or PORT.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(client.TrafficFilterNetwork, client.TrafficFilterIPAddress, client.TrafficFilterMacAddress, client.TrafficFilterPort),
				},
			},
			"source_traffic_filter_values": schema.ListAttribute{
				Description: "Source traffic filter values.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"destination_zone_id": schema.StringAttribute{
				Description: "Destination firewall zone ID.",
				Required:    true,
			},
			"destination_traffic_filter_type": schema.StringAttribute{
				Description: "Destination traffic filter type: NETWORK, IP_ADDRESS, MAC_ADDRESS, or PORT.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(client.TrafficFilterNetwork, client.TrafficFilterIPAddress, client.TrafficFilterMacAddress, client.TrafficFilterPort),
				},
			},
			"destination_traffic_filter_values": schema.ListAttribute{
				Description: "Destination traffic filter values.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"ip_version": schema.StringAttribute{
				Description: "IP version: IPV4, IPV6, or IPV4_AND_IPV6.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(client.IPVersionIPv4, client.IPVersionIPv6, client.IPVersionIPv4AndV6),
				},
			},
			"connection_state_filter": schema.ListAttribute{
				Description: "Connection state filter: NEW, INVALID, ESTABLISHED, RELATED.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"ipsec_filter": schema.StringAttribute{
				Description: "IPsec filter: MATCH_ENCRYPTED or MATCH_NOT_ENCRYPTED.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(client.IpsecMatchEncrypted, client.IpsecMatchNotEncrypted),
				},
			},
			"logging_enabled": schema.BoolAttribute{
				Description: "Whether logging is enabled for this policy.",
				Required:    true,
			},
		},
	}
}

func (r *FirewallPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := extractClient(req.ProviderData, "Resource")
	resp.Diagnostics.Append(diags...)
	if c != nil {
		r.client = c
	}
}

func (r *FirewallPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan FirewallPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, diags := firewallPolicyModelToAPI(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.CreateFirewallPolicy(ctx, plan.SiteID.ValueString(), policy)
	if err != nil {
		resp.Diagnostics.AddError("Error creating firewall policy", err.Error())
		return
	}

	plan.ID = types.StringValue(result.ID)
	var d diag.Diagnostics
	d = firewallPolicyAPIToModel(ctx, result, &plan)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *FirewallPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state FirewallPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetFirewallPolicy(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading firewall policy", err.Error())
		return
	}

	diags := firewallPolicyAPIToModel(ctx, result, &state)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *FirewallPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan FirewallPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state FirewallPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, diags := firewallPolicyModelToAPI(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.UpdateFirewallPolicy(ctx, plan.SiteID.ValueString(), state.ID.ValueString(), policy)
	if err != nil {
		resp.Diagnostics.AddError("Error updating firewall policy", err.Error())
		return
	}

	plan.ID = state.ID
	var d diag.Diagnostics
	d = firewallPolicyAPIToModel(ctx, result, &plan)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *FirewallPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state FirewallPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteFirewallPolicy(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting firewall policy", err.Error())
		return
	}
}

func (r *FirewallPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importCompositeState(ctx, req, resp)
}
