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
	_ resource.Resource                = &AclRuleResource{}
	_ resource.ResourceWithImportState = &AclRuleResource{}
)

type AclRuleResource struct {
	client *client.Client
}

type AclRuleResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	SiteID                  types.String `tfsdk:"site_id"`
	Type                    types.String `tfsdk:"type"`
	Enabled                 types.Bool   `tfsdk:"enabled"`
	Name                    types.String `tfsdk:"name"`
	Description             types.String `tfsdk:"description"`
	Action                  types.String `tfsdk:"action"`
	SourceFilterType        types.String `tfsdk:"source_filter_type"`
	SourceFilterValues      types.List   `tfsdk:"source_filter_values"`
	SourceFilterPorts       types.List   `tfsdk:"source_filter_ports"`
	DestinationFilterType   types.String `tfsdk:"destination_filter_type"`
	DestinationFilterValues types.List   `tfsdk:"destination_filter_values"`
	DestinationFilterPorts  types.List   `tfsdk:"destination_filter_ports"`
	ProtocolFilter          types.List   `tfsdk:"protocol_filter"`
	EnforcingDeviceIDs      types.List   `tfsdk:"enforcing_device_ids"`
	NetworkIDFilter         types.String `tfsdk:"network_id_filter"`
}

func NewAclRuleResource() resource.Resource {
	return &AclRuleResource{}
}

func (r *AclRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_acl_rule"
}

func (r *AclRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a UniFi ACL rule.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the ACL rule.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site_id": schema.StringAttribute{
				Description: "The site ID where the ACL rule is managed.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Description: "ACL rule type: IPV4 or MAC.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(client.AclTypeIPv4, client.AclTypeMAC),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the ACL rule is enabled.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name of the ACL rule.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description of the ACL rule.",
				Optional:    true,
			},
			"action": schema.StringAttribute{
				Description: "Action to take: ALLOW or BLOCK.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(client.ActionAllow, client.ActionBlock),
				},
			},
			"source_filter_type": schema.StringAttribute{
				Description: "Source filter type: IP_ADDRESSES_OR_SUBNETS, NETWORKS, PORTS, or MAC_ADDRESSES.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(client.AclFilterIPAddressesOrSubnets, client.AclFilterNetworks, client.AclFilterPorts, client.AclFilterMacAddresses),
				},
			},
			"source_filter_values": schema.ListAttribute{
				Description: "Source filter values (IPs, subnets, network IDs, or MAC addresses).",
				Optional:    true,
				ElementType: types.StringType,
			},
			"source_filter_ports": schema.ListAttribute{
				Description: "Source filter port numbers.",
				Optional:    true,
				ElementType: types.Int64Type,
			},
			"destination_filter_type": schema.StringAttribute{
				Description: "Destination filter type: IP_ADDRESSES_OR_SUBNETS, NETWORKS, PORTS, or MAC_ADDRESSES.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(client.AclFilterIPAddressesOrSubnets, client.AclFilterNetworks, client.AclFilterPorts, client.AclFilterMacAddresses),
				},
			},
			"destination_filter_values": schema.ListAttribute{
				Description: "Destination filter values (IPs, subnets, network IDs, or MAC addresses).",
				Optional:    true,
				ElementType: types.StringType,
			},
			"destination_filter_ports": schema.ListAttribute{
				Description: "Destination filter port numbers.",
				Optional:    true,
				ElementType: types.Int64Type,
			},
			"protocol_filter": schema.ListAttribute{
				Description: "Protocol filter: TCP, UDP.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"enforcing_device_ids": schema.ListAttribute{
				Description: "List of device UUIDs to enforce this rule.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"network_id_filter": schema.StringAttribute{
				Description: "Network ID filter (MAC type only).",
				Optional:    true,
			},
		},
	}
}

func (r *AclRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := extractClient(req.ProviderData, "Resource")
	resp.Diagnostics.Append(diags...)
	if c != nil {
		r.client = c
	}
}

func (r *AclRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AclRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, diags := aclRuleModelToAPI(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.CreateAclRule(ctx, plan.SiteID.ValueString(), rule)
	if err != nil {
		resp.Diagnostics.AddError("Error creating ACL rule", err.Error())
		return
	}

	plan.ID = types.StringValue(result.ID)
	var d diag.Diagnostics
	d = aclRuleAPIToModel(ctx, result, &plan)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *AclRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AclRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetAclRule(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading ACL rule", err.Error())
		return
	}

	diags := aclRuleAPIToModel(ctx, result, &state)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *AclRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AclRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state AclRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, diags := aclRuleModelToAPI(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.UpdateAclRule(ctx, plan.SiteID.ValueString(), state.ID.ValueString(), rule)
	if err != nil {
		resp.Diagnostics.AddError("Error updating ACL rule", err.Error())
		return
	}

	plan.ID = state.ID
	var d diag.Diagnostics
	d = aclRuleAPIToModel(ctx, result, &plan)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *AclRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AclRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteAclRule(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting ACL rule", err.Error())
		return
	}
}

func (r *AclRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importCompositeState(ctx, req, resp)
}
