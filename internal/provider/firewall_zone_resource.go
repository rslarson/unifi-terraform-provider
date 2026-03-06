package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rslarson/terraform-provider-unifi/internal/client"
)

var (
	_ resource.Resource                = &FirewallZoneResource{}
	_ resource.ResourceWithImportState = &FirewallZoneResource{}
)

type FirewallZoneResource struct {
	client *client.Client
}

type FirewallZoneResourceModel struct {
	ID         types.String `tfsdk:"id"`
	SiteID     types.String `tfsdk:"site_id"`
	Name       types.String `tfsdk:"name"`
	NetworkIDs types.List   `tfsdk:"network_ids"`
}

func NewFirewallZoneResource() resource.Resource {
	return &FirewallZoneResource{}
}

func (r *FirewallZoneResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall_zone"
}

func (r *FirewallZoneResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a UniFi custom firewall zone.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the firewall zone.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site_id": schema.StringAttribute{
				Description: "The site ID where the firewall zone is managed.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the firewall zone.",
				Required:    true,
			},
			"network_ids": schema.ListAttribute{
				Description: "List of network IDs associated with this firewall zone.",
				Required:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *FirewallZoneResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := extractClient(req.ProviderData, "Resource")
	resp.Diagnostics.Append(diags...)
	if c != nil {
		r.client = c
	}
}

func (r *FirewallZoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan FirewallZoneResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zone, diags := firewallZoneModelToAPI(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.CreateFirewallZone(ctx, plan.SiteID.ValueString(), zone)
	if err != nil {
		resp.Diagnostics.AddError("Error creating firewall zone", err.Error())
		return
	}

	plan.ID = types.StringValue(result.ID)
	var d diag.Diagnostics
	plan.Name, plan.NetworkIDs, d = firewallZoneAPIToModel(ctx, result)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *FirewallZoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state FirewallZoneResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetFirewallZone(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading firewall zone", err.Error())
		return
	}

	var diags diag.Diagnostics
	state.Name, state.NetworkIDs, diags = firewallZoneAPIToModel(ctx, result)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *FirewallZoneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan FirewallZoneResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state FirewallZoneResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zone, diags := firewallZoneModelToAPI(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.UpdateFirewallZone(ctx, plan.SiteID.ValueString(), state.ID.ValueString(), zone)
	if err != nil {
		resp.Diagnostics.AddError("Error updating firewall zone", err.Error())
		return
	}

	plan.ID = state.ID
	var d diag.Diagnostics
	plan.Name, plan.NetworkIDs, d = firewallZoneAPIToModel(ctx, result)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *FirewallZoneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state FirewallZoneResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteFirewallZone(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting firewall zone", err.Error())
		return
	}
}

func (r *FirewallZoneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importCompositeState(ctx, req, resp)
}
