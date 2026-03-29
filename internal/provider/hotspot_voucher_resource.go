package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rslarson/terraform-provider-unifi/internal/client"
)

var _ resource.Resource = &HotspotVoucherResource{}

type HotspotVoucherResource struct {
	client *client.Client
}

type HotspotVoucherResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	SiteID               types.String `tfsdk:"site_id"`
	Name                 types.String `tfsdk:"name"`
	Code                 types.String `tfsdk:"code"`
	TimeLimitMinutes     types.Int64  `tfsdk:"time_limit_minutes"`
	AuthorizedGuestLimit types.Int64  `tfsdk:"authorized_guest_limit"`
	DataUsageLimitMBytes types.Int64  `tfsdk:"data_usage_limit_mbytes"`
	RxRateLimitKbps      types.Int64  `tfsdk:"rx_rate_limit_kbps"`
	TxRateLimitKbps      types.Int64  `tfsdk:"tx_rate_limit_kbps"`
}

func NewHotspotVoucherResource() resource.Resource {
	return &HotspotVoucherResource{}
}

func (r *HotspotVoucherResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_hotspot_voucher"
}

func (r *HotspotVoucherResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a UniFi hotspot voucher. This resource supports create and delete only; updates are not supported.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the hotspot voucher.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site_id": schema.StringAttribute{
				Description: "The site ID where the hotspot voucher is managed.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the hotspot voucher.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"code": schema.StringAttribute{
				Description: "The voucher code returned by the API.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"time_limit_minutes": schema.Int64Attribute{
				Description: "Time limit in minutes (1-1000000).",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.Between(1, 1000000),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"authorized_guest_limit": schema.Int64Attribute{
				Description: "Maximum number of authorized guests.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"data_usage_limit_mbytes": schema.Int64Attribute{
				Description: "Data usage limit in megabytes (1-1048576).",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(1, 1048576),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"rx_rate_limit_kbps": schema.Int64Attribute{
				Description: "Download rate limit in kbps (2-100000).",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(2, 100000),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"tx_rate_limit_kbps": schema.Int64Attribute{
				Description: "Upload rate limit in kbps (2-100000).",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(2, 100000),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *HotspotVoucherResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := extractClient(req.ProviderData, "Resource")
	resp.Diagnostics.Append(diags...)
	if c != nil {
		r.client = c
	}
}

func (r *HotspotVoucherResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan HotspotVoucherResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.HotspotVoucherCreateRequest{
		Name:             plan.Name.ValueString(),
		Count:            1,
		TimeLimitMinutes: int(plan.TimeLimitMinutes.ValueInt64()),
	}

	if !plan.AuthorizedGuestLimit.IsNull() {
		v := int(plan.AuthorizedGuestLimit.ValueInt64())
		createReq.AuthorizedGuestLimit = &v
	}
	if !plan.DataUsageLimitMBytes.IsNull() {
		v := int(plan.DataUsageLimitMBytes.ValueInt64())
		createReq.DataUsageLimitMBytes = &v
	}
	if !plan.RxRateLimitKbps.IsNull() {
		v := int(plan.RxRateLimitKbps.ValueInt64())
		createReq.RxRateLimitKbps = &v
	}
	if !plan.TxRateLimitKbps.IsNull() {
		v := int(plan.TxRateLimitKbps.ValueInt64())
		createReq.TxRateLimitKbps = &v
	}

	vouchers, err := r.client.CreateHotspotVoucher(ctx, plan.SiteID.ValueString(), createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating hotspot voucher", err.Error())
		return
	}

	if len(vouchers) == 0 {
		resp.Diagnostics.AddError("Error creating hotspot voucher", "API returned no vouchers")
		return
	}

	result := &vouchers[0]
	plan.ID = types.StringValue(result.ID)
	hotspotVoucherAPIToModel(result, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *HotspotVoucherResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state HotspotVoucherResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetHotspotVoucher(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading hotspot voucher", err.Error())
		return
	}

	hotspotVoucherAPIToModel(result, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *HotspotVoucherResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update not supported",
		"Hotspot vouchers do not support updates. Delete and recreate the voucher instead.",
	)
}

func (r *HotspotVoucherResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state HotspotVoucherResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteHotspotVoucher(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting hotspot voucher", err.Error())
		return
	}
}
