package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
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
	_ resource.Resource                = &TrafficMatchingListResource{}
	_ resource.ResourceWithImportState = &TrafficMatchingListResource{}
)

type TrafficMatchingListResource struct {
	client *client.Client
}

type TrafficMatchingListResourceModel struct {
	ID     types.String `tfsdk:"id"`
	SiteID types.String `tfsdk:"site_id"`
	Type   types.String `tfsdk:"type"`
	Name   types.String `tfsdk:"name"`
	Items  types.List   `tfsdk:"items"`
}

type TrafficMatchingItemModel struct {
	Type       types.String `tfsdk:"type"`
	Value      types.String `tfsdk:"value"`
	Subnet     types.String `tfsdk:"subnet"`
	Start      types.String `tfsdk:"start"`
	End        types.String `tfsdk:"end"`
	PortNumber types.Int64  `tfsdk:"port_number"`
	StartPort  types.Int64  `tfsdk:"start_port"`
	EndPort    types.Int64  `tfsdk:"end_port"`
}

// trafficMatchingItemAttrTypes returns the attribute type map for the nested items list.
func trafficMatchingItemAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"type":        types.StringType,
		"value":       types.StringType,
		"subnet":      types.StringType,
		"start":       types.StringType,
		"end":         types.StringType,
		"port_number": types.Int64Type,
		"start_port":  types.Int64Type,
		"end_port":    types.Int64Type,
	}
}

func NewTrafficMatchingListResource() resource.Resource {
	return &TrafficMatchingListResource{}
}

func (r *TrafficMatchingListResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_traffic_matching_list"
}

func (r *TrafficMatchingListResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a UniFi traffic matching list.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the traffic matching list.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site_id": schema.StringAttribute{
				Description: "The site ID where the traffic matching list is managed.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Description: "Traffic matching list type: PORTS, IPV4_ADDRESSES, or IPV6_ADDRESSES.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(client.TrafficMatchingPorts, client.TrafficMatchingIPv4Addresses, client.TrafficMatchingIPv6Addresses),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the traffic matching list.",
				Required:    true,
			},
			"items": schema.ListNestedAttribute{
				Description: "List of traffic matching items.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Description: "Item type: IP_ADDRESS, SUBNET, IP_ADDRESS_RANGE, PORT_NUMBER, or PORT_NUMBER_RANGE.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf(client.TrafficItemIPAddress, client.TrafficItemSubnet, client.TrafficItemIPAddressRange, client.TrafficItemPortNumber, client.TrafficItemPortNumberRange),
							},
						},
						"value": schema.StringAttribute{
							Description: "IP address value.",
							Optional:    true,
						},
						"subnet": schema.StringAttribute{
							Description: "Subnet value.",
							Optional:    true,
						},
						"start": schema.StringAttribute{
							Description: "Start of IP address range.",
							Optional:    true,
						},
						"end": schema.StringAttribute{
							Description: "End of IP address range.",
							Optional:    true,
						},
						"port_number": schema.Int64Attribute{
							Description: "Port number.",
							Optional:    true,
						},
						"start_port": schema.Int64Attribute{
							Description: "Start port of range.",
							Optional:    true,
						},
						"end_port": schema.Int64Attribute{
							Description: "End port of range.",
							Optional:    true,
						},
					},
				},
			},
		},
	}
}

func (r *TrafficMatchingListResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := extractClient(req.ProviderData, "Resource")
	resp.Diagnostics.Append(diags...)
	if c != nil {
		r.client = c
	}
}

func (r *TrafficMatchingListResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TrafficMatchingListResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	list, diags := trafficMatchingListModelToAPI(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.CreateTrafficMatchingList(ctx, plan.SiteID.ValueString(), list)
	if err != nil {
		resp.Diagnostics.AddError("Error creating traffic matching list", err.Error())
		return
	}

	plan.ID = types.StringValue(result.ID)
	var d diag.Diagnostics
	d = trafficMatchingListAPIToModel(ctx, result, &plan)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *TrafficMatchingListResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TrafficMatchingListResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetTrafficMatchingList(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading traffic matching list", err.Error())
		return
	}

	diags := trafficMatchingListAPIToModel(ctx, result, &state)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *TrafficMatchingListResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TrafficMatchingListResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state TrafficMatchingListResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	list, diags := trafficMatchingListModelToAPI(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.UpdateTrafficMatchingList(ctx, plan.SiteID.ValueString(), state.ID.ValueString(), list)
	if err != nil {
		resp.Diagnostics.AddError("Error updating traffic matching list", err.Error())
		return
	}

	plan.ID = state.ID
	var d diag.Diagnostics
	d = trafficMatchingListAPIToModel(ctx, result, &plan)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *TrafficMatchingListResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TrafficMatchingListResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteTrafficMatchingList(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting traffic matching list", err.Error())
		return
	}
}

func (r *TrafficMatchingListResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importCompositeState(ctx, req, resp)
}
