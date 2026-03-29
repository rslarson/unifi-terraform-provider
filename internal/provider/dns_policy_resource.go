package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
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
	_ resource.Resource                = &DnsPolicyResource{}
	_ resource.ResourceWithImportState = &DnsPolicyResource{}
)

type DnsPolicyResource struct {
	client *client.Client
}

type DnsPolicyResourceModel struct {
	ID          types.String `tfsdk:"id"`
	SiteID      types.String `tfsdk:"site_id"`
	Type        types.String `tfsdk:"type"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	Name        types.String `tfsdk:"name"`
	Domain      types.String `tfsdk:"domain"`
	IPv4Address types.String `tfsdk:"ipv4_address"`
	IPv6Address types.String `tfsdk:"ipv6_address"`
	Target      types.String `tfsdk:"target"`
	TTLSeconds  types.Int64  `tfsdk:"ttl_seconds"`
	Priority    types.Int64  `tfsdk:"priority"`
	Weight      types.Int64  `tfsdk:"weight"`
	Port        types.Int64  `tfsdk:"port"`
	TxtValue    types.String `tfsdk:"txt_value"`
	ForwardTo   types.List   `tfsdk:"forward_to"`
}

func NewDnsPolicyResource() resource.Resource {
	return &DnsPolicyResource{}
}

func (r *DnsPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_policy"
}

func (r *DnsPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a UniFi DNS policy.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the DNS policy.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site_id": schema.StringAttribute{
				Description: "The site ID where the DNS policy is managed.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Description: "DNS policy type: A_RECORD, AAAA_RECORD, CNAME_RECORD, MX_RECORD, TXT_RECORD, SRV_RECORD, or FORWARD_DOMAIN.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						client.DnsPolicyARecord,
						client.DnsPolicyAAAARecord,
						client.DnsPolicyCNAMERecord,
						client.DnsPolicyMXRecord,
						client.DnsPolicyTXTRecord,
						client.DnsPolicySRVRecord,
						client.DnsPolicyForwardDomain,
					),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the DNS policy is enabled.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name of the DNS policy.",
				Required:    true,
			},
			"domain": schema.StringAttribute{
				Description: "Domain name for the DNS record.",
				Optional:    true,
			},
			"ipv4_address": schema.StringAttribute{
				Description: "IPv4 address (A_RECORD type).",
				Optional:    true,
			},
			"ipv6_address": schema.StringAttribute{
				Description: "IPv6 address (AAAA_RECORD type).",
				Optional:    true,
			},
			"target": schema.StringAttribute{
				Description: "Target hostname (CNAME, MX, or SRV types).",
				Optional:    true,
			},
			"ttl_seconds": schema.Int64Attribute{
				Description: "TTL in seconds (0-86400).",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, 86400),
				},
			},
			"priority": schema.Int64Attribute{
				Description: "Priority (MX or SRV types).",
				Optional:    true,
			},
			"weight": schema.Int64Attribute{
				Description: "Weight (SRV type).",
				Optional:    true,
			},
			"port": schema.Int64Attribute{
				Description: "Port (SRV type).",
				Optional:    true,
			},
			"txt_value": schema.StringAttribute{
				Description: "TXT record value.",
				Optional:    true,
			},
			"forward_to": schema.ListAttribute{
				Description: "List of DNS servers to forward to (FORWARD_DOMAIN type).",
				Optional:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *DnsPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := extractClient(req.ProviderData, "Resource")
	resp.Diagnostics.Append(diags...)
	if c != nil {
		r.client = c
	}
}

func (r *DnsPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DnsPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, diags := dnsPolicyModelToAPI(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.CreateDnsPolicy(ctx, plan.SiteID.ValueString(), policy)
	if err != nil {
		resp.Diagnostics.AddError("Error creating DNS policy", err.Error())
		return
	}

	plan.ID = types.StringValue(result.ID)
	var d diag.Diagnostics
	d = dnsPolicyAPIToModel(ctx, result, &plan)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *DnsPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DnsPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetDnsPolicy(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading DNS policy", err.Error())
		return
	}

	diags := dnsPolicyAPIToModel(ctx, result, &state)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *DnsPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DnsPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state DnsPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, diags := dnsPolicyModelToAPI(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.UpdateDnsPolicy(ctx, plan.SiteID.ValueString(), state.ID.ValueString(), policy)
	if err != nil {
		resp.Diagnostics.AddError("Error updating DNS policy", err.Error())
		return
	}

	plan.ID = state.ID
	var d diag.Diagnostics
	d = dnsPolicyAPIToModel(ctx, result, &plan)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *DnsPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DnsPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteDnsPolicy(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting DNS policy", err.Error())
		return
	}
}

func (r *DnsPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importCompositeState(ctx, req, resp)
}
