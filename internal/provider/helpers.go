package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rslarson/terraform-provider-unifi/internal/client"
)

// importCompositeState parses a "site_id/resource_id" import string and sets
// the site_id and id attributes on the state.
func importCompositeState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	siteID, resourceID, err := parseCompositeID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site_id"), siteID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), resourceID)...)
}

// deviceAPIToModel maps a client.Device response to a DeviceDataSourceModel.
func deviceAPIToModel(model *DeviceDataSourceModel, result *client.Device) {
	model.MacAddress = types.StringValue(result.MacAddress)
	model.IPAddress = types.StringValue(result.IPAddress)
	model.Name = types.StringValue(result.Name)
	model.Model = types.StringValue(result.Model)
	model.Supported = types.BoolValue(result.Supported)
	model.State = types.StringValue(result.State)
	model.FirmwareVersion = types.StringValue(result.FirmwareVersion)
	model.FirmwareUpdatable = types.BoolValue(result.FirmwareUpdatable)
	model.AdoptedAt = types.StringValue(result.AdoptedAt)
	model.ProvisionedAt = types.StringValue(result.ProvisionedAt)
	model.ConfigurationID = types.StringValue(result.ConfigurationID)
	if result.Uplink != nil {
		model.UplinkDeviceID = types.StringValue(result.Uplink.DeviceID)
	} else {
		model.UplinkDeviceID = types.StringNull()
	}
}

// extractClient extracts the *client.Client from provider data, returning diagnostics on failure.
func extractClient(providerData interface{}, typeName string) (*client.Client, diag.Diagnostics) {
	var diags diag.Diagnostics

	if providerData == nil {
		return nil, diags
	}

	c, ok := providerData.(*client.Client)
	if !ok {
		diags.AddError(
			fmt.Sprintf("Unexpected %s Configure Type", typeName),
			fmt.Sprintf("Expected *client.Client, got: %T", providerData),
		)
		return nil, diags
	}

	return c, diags
}

// parseCompositeID splits a "site_id/resource_id" import string into its parts.
func parseCompositeID(id string) (siteID, resourceID string, err error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("expected import ID format: site_id/resource_id, got: %s", id)
	}
	return parts[0], parts[1], nil
}

// networkAPIToModel maps a client.Network to common Terraform model fields.
func networkAPIToModel(ctx context.Context, result *client.Network) (name types.String, management types.String, enabled types.Bool, vlanID types.Int64, dhcpIPs types.List, diags diag.Diagnostics) {
	name = types.StringValue(result.Name)
	management = types.StringValue(result.Management)
	enabled = types.BoolValue(result.Enabled)
	vlanID = types.Int64Value(int64(result.VlanID))

	if result.DhcpGuarding != nil && len(result.DhcpGuarding.TrustedDhcpServerIPAddresses) > 0 {
		ips, d := types.ListValueFrom(ctx, types.StringType, result.DhcpGuarding.TrustedDhcpServerIPAddresses)
		diags.Append(d...)
		dhcpIPs = ips
	} else {
		dhcpIPs = types.ListNull(types.StringType)
	}

	return
}

// networkModelToAPI converts a NetworkResourceModel into a client.Network.
func networkModelToAPI(ctx context.Context, model NetworkResourceModel) (*client.Network, diag.Diagnostics) {
	var diags diag.Diagnostics

	network := &client.Network{
		Name:       model.Name.ValueString(),
		Management: model.Management.ValueString(),
		Enabled:    model.Enabled.ValueBool(),
		VlanID:     int(model.VlanID.ValueInt64()),
	}

	if !model.TrustedDhcpServerIPAddresses.IsNull() {
		var ips []string
		diags.Append(model.TrustedDhcpServerIPAddresses.ElementsAs(ctx, &ips, false)...)
		if diags.HasError() {
			return nil, diags
		}
		network.DhcpGuarding = &client.DhcpGuarding{
			TrustedDhcpServerIPAddresses: ips,
		}
	}

	return network, diags
}

// wifiBroadcastAPIToModel maps a client.WifiBroadcast to common Terraform model fields.
func wifiBroadcastAPIToModel(result *client.WifiBroadcast) (
	broadcastType, name types.String,
	enabled, clientIsolation, hideName, mcastUcast, uapsd types.Bool,
	securityType, networkType, networkID types.String,
) {
	broadcastType = types.StringValue(result.Type)
	name = types.StringValue(result.Name)
	enabled = types.BoolValue(result.Enabled)
	clientIsolation = types.BoolValue(result.ClientIsolationEnabled)
	hideName = types.BoolValue(result.HideName)
	mcastUcast = types.BoolValue(result.MulticastToUnicastConversionEnabled)
	uapsd = types.BoolValue(result.UapsdEnabled)

	if result.SecurityConfiguration != nil {
		securityType = types.StringValue(result.SecurityConfiguration.Type)
	}
	if result.Network != nil {
		networkType = types.StringValue(result.Network.Type)
		if result.Network.NetworkID != "" {
			networkID = types.StringValue(result.Network.NetworkID)
		} else {
			networkID = types.StringNull()
		}
	}

	return
}

// firewallZoneAPIToModel maps a client.FirewallZone response to Terraform model fields.
func firewallZoneAPIToModel(ctx context.Context, result *client.FirewallZone) (name types.String, networkIDs types.List, diags diag.Diagnostics) {
	name = types.StringValue(result.Name)
	ids, d := types.ListValueFrom(ctx, types.StringType, result.NetworkIDs)
	diags.Append(d...)
	networkIDs = ids
	return
}

// wifiBroadcastModelToAPI converts a WifiBroadcastResourceModel to a client.WifiBroadcast.
func wifiBroadcastModelToAPI(plan WifiBroadcastResourceModel) *client.WifiBroadcast {
	broadcast := &client.WifiBroadcast{
		Type:                                plan.Type.ValueString(),
		Name:                                plan.Name.ValueString(),
		Enabled:                             plan.Enabled.ValueBool(),
		ClientIsolationEnabled:              plan.ClientIsolationEnabled.ValueBool(),
		HideName:                            plan.HideName.ValueBool(),
		MulticastToUnicastConversionEnabled: plan.MulticastToUnicastConversionEnabled.ValueBool(),
		UapsdEnabled:                        plan.UapsdEnabled.ValueBool(),
		SecurityConfiguration: &client.SecurityConfiguration{
			Type: plan.SecurityType.ValueString(),
		},
		Network: &client.BroadcastNetwork{
			Type: plan.NetworkType.ValueString(),
		},
	}

	if !plan.Passphrase.IsNull() && !plan.Passphrase.IsUnknown() {
		broadcast.SecurityConfiguration.Passphrase = plan.Passphrase.ValueString()
	}

	if !plan.NetworkID.IsNull() && !plan.NetworkID.IsUnknown() {
		broadcast.Network.NetworkID = plan.NetworkID.ValueString()
	}

	return broadcast
}

// firewallZoneModelToAPI converts a FirewallZoneResourceModel into a client.FirewallZone.
func firewallZoneModelToAPI(ctx context.Context, model FirewallZoneResourceModel) (*client.FirewallZone, diag.Diagnostics) {
	var diags diag.Diagnostics
	var ids []string
	diags.Append(model.NetworkIDs.ElementsAs(ctx, &ids, false)...)
	if diags.HasError() {
		return nil, diags
	}
	return &client.FirewallZone{
		Name:       model.Name.ValueString(),
		NetworkIDs: ids,
	}, diags
}
