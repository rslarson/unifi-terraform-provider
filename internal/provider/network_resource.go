package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
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
				Description: "VLAN ID (2-4000).",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.Between(2, 4000),
				},
			},
			"trusted_dhcp_server_ip_addresses": schema.ListAttribute{
				Description: "List of trusted DHCP server IP addresses for DHCP guarding.",
				Optional:    true,
				ElementType: types.StringType,
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

	network, diags := networkModelToAPI(ctx, plan.Name.ValueString(), plan.Management.ValueString(), plan.Enabled.ValueBool(), plan.VlanID.ValueInt64(), plan.TrustedDhcpServerIPAddresses)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.CreateNetwork(ctx, plan.SiteID.ValueString(), network)
	if err != nil {
		resp.Diagnostics.AddError("Error creating network", err.Error())
		return
	}

	plan.ID = types.StringValue(result.ID)
	plan.Name, plan.Management, plan.Enabled, plan.VlanID, plan.TrustedDhcpServerIPAddresses, diags = networkAPIToModel(ctx, result)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
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

	var diags diag.Diagnostics
	state.Name, state.Management, state.Enabled, state.VlanID, state.TrustedDhcpServerIPAddresses, diags = networkAPIToModel(ctx, result)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *NetworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan NetworkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state NetworkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	network, diags := networkModelToAPI(ctx, plan.Name.ValueString(), plan.Management.ValueString(), plan.Enabled.ValueBool(), plan.VlanID.ValueInt64(), plan.TrustedDhcpServerIPAddresses)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.UpdateNetwork(ctx, plan.SiteID.ValueString(), state.ID.ValueString(), network)
	if err != nil {
		resp.Diagnostics.AddError("Error updating network", err.Error())
		return
	}

	plan.ID = state.ID
	plan.Name, plan.Management, plan.Enabled, plan.VlanID, plan.TrustedDhcpServerIPAddresses, diags = networkAPIToModel(ctx, result)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
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
	siteID, resourceID, err := parseCompositeID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site_id"), siteID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), resourceID)...)
}
