package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rslarson/terraform-provider-unifi/internal/client"
)

var _ datasource.DataSource = &AclRuleDataSource{}

type AclRuleDataSource struct {
	client *client.Client
}

type AclRuleDataSourceModel struct {
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

func NewAclRuleDataSource() datasource.DataSource {
	return &AclRuleDataSource{}
}

func (d *AclRuleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_acl_rule"
}

func (d *AclRuleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches details of a UniFi ACL rule.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the ACL rule.",
				Required:    true,
			},
			"site_id": schema.StringAttribute{
				Description: "The site ID.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "ACL rule type: IPV4 or MAC.",
				Computed:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the ACL rule is enabled.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name of the ACL rule.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description of the ACL rule.",
				Computed:    true,
			},
			"action": schema.StringAttribute{
				Description: "Action to take: ALLOW or BLOCK.",
				Computed:    true,
			},
			"source_filter_type": schema.StringAttribute{
				Description: "Source filter type.",
				Computed:    true,
			},
			"source_filter_values": schema.ListAttribute{
				Description: "Source filter values.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"source_filter_ports": schema.ListAttribute{
				Description: "Source filter port numbers.",
				Computed:    true,
				ElementType: types.Int64Type,
			},
			"destination_filter_type": schema.StringAttribute{
				Description: "Destination filter type.",
				Computed:    true,
			},
			"destination_filter_values": schema.ListAttribute{
				Description: "Destination filter values.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"destination_filter_ports": schema.ListAttribute{
				Description: "Destination filter port numbers.",
				Computed:    true,
				ElementType: types.Int64Type,
			},
			"protocol_filter": schema.ListAttribute{
				Description: "Protocol filter.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"enforcing_device_ids": schema.ListAttribute{
				Description: "List of enforcing device UUIDs.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"network_id_filter": schema.StringAttribute{
				Description: "Network ID filter (MAC type only).",
				Computed:    true,
			},
		},
	}
}

func (d *AclRuleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := extractClient(req.ProviderData, "Data Source")
	resp.Diagnostics.Append(diags...)
	if c != nil {
		d.client = c
	}
}

func (d *AclRuleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config AclRuleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.GetAclRule(ctx, config.SiteID.ValueString(), config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading ACL rule", err.Error())
		return
	}

	// Reuse the resource model for conversion, then copy fields to the data source model.
	var resourceModel AclRuleResourceModel
	resourceModel.ID = config.ID
	resourceModel.SiteID = config.SiteID
	diags := aclRuleAPIToModel(ctx, result, &resourceModel)
	resp.Diagnostics.Append(diags...)

	config.Type = resourceModel.Type
	config.Enabled = resourceModel.Enabled
	config.Name = resourceModel.Name
	config.Description = resourceModel.Description
	config.Action = resourceModel.Action
	config.SourceFilterType = resourceModel.SourceFilterType
	config.SourceFilterValues = resourceModel.SourceFilterValues
	config.SourceFilterPorts = resourceModel.SourceFilterPorts
	config.DestinationFilterType = resourceModel.DestinationFilterType
	config.DestinationFilterValues = resourceModel.DestinationFilterValues
	config.DestinationFilterPorts = resourceModel.DestinationFilterPorts
	config.ProtocolFilter = resourceModel.ProtocolFilter
	config.EnforcingDeviceIDs = resourceModel.EnforcingDeviceIDs
	config.NetworkIDFilter = resourceModel.NetworkIDFilter

	resp.Diagnostics.Append(resp.State.Set(ctx, config)...)
}
