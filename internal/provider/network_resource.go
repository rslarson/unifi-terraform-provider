package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
					stringvalidator.OneOf("UNMANAGED", "GATEWAY", "SWITCH"),
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

func (r *NetworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan NetworkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	network := &client.Network{
		Name:       plan.Name.ValueString(),
		Management: plan.Management.ValueString(),
		Enabled:    plan.Enabled.ValueBool(),
		VlanID:     int(plan.VlanID.ValueInt64()),
	}

	if !plan.TrustedDhcpServerIPAddresses.IsNull() {
		var ips []string
		resp.Diagnostics.Append(plan.TrustedDhcpServerIPAddresses.ElementsAs(ctx, &ips, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		network.DhcpGuarding = &client.DhcpGuarding{
			TrustedDhcpServerIPAddresses: ips,
		}
	}

	result, err := r.client.CreateNetwork(ctx, plan.SiteID.ValueString(), network)
	if err != nil {
		resp.Diagnostics.AddError("Error creating network", err.Error())
		return
	}

	plan.ID = types.StringValue(result.ID)
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
		if apiErr, ok := err.(*client.APIError); ok && apiErr.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading network", err.Error())
		return
	}

	state.Name = types.StringValue(result.Name)
	state.Management = types.StringValue(result.Management)
	state.Enabled = types.BoolValue(result.Enabled)
	state.VlanID = types.Int64Value(int64(result.VlanID))

	if result.DhcpGuarding != nil && len(result.DhcpGuarding.TrustedDhcpServerIPAddresses) > 0 {
		ips, diags := types.ListValueFrom(ctx, types.StringType, result.DhcpGuarding.TrustedDhcpServerIPAddresses)
		resp.Diagnostics.Append(diags...)
		state.TrustedDhcpServerIPAddresses = ips
	} else {
		state.TrustedDhcpServerIPAddresses = types.ListNull(types.StringType)
	}

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

	network := &client.Network{
		Name:       plan.Name.ValueString(),
		Management: plan.Management.ValueString(),
		Enabled:    plan.Enabled.ValueBool(),
		VlanID:     int(plan.VlanID.ValueInt64()),
	}

	if !plan.TrustedDhcpServerIPAddresses.IsNull() {
		var ips []string
		resp.Diagnostics.Append(plan.TrustedDhcpServerIPAddresses.ElementsAs(ctx, &ips, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		network.DhcpGuarding = &client.DhcpGuarding{
			TrustedDhcpServerIPAddresses: ips,
		}
	}

	_, err := r.client.UpdateNetwork(ctx, plan.SiteID.ValueString(), state.ID.ValueString(), network)
	if err != nil {
		resp.Diagnostics.AddError("Error updating network", err.Error())
		return
	}

	plan.ID = state.ID
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
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
