package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rslarson/terraform-provider-unifi/internal/client"
)

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

// networkModelToAPI converts Terraform model fields into a client.Network.
func networkModelToAPI(ctx context.Context, name string, management string, enabled bool, vlanID int64, dhcpIPs types.List) (*client.Network, diag.Diagnostics) {
	var diags diag.Diagnostics

	network := &client.Network{
		Name:       name,
		Management: management,
		Enabled:    enabled,
		VlanID:     int(vlanID),
	}

	if !dhcpIPs.IsNull() {
		var ips []string
		diags.Append(dhcpIPs.ElementsAs(ctx, &ips, false)...)
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

// firewallZoneModelToAPI converts Terraform model fields into a client.FirewallZone.
func firewallZoneModelToAPI(ctx context.Context, name string, networkIDs types.List) (*client.FirewallZone, diag.Diagnostics) {
	var diags diag.Diagnostics
	var ids []string
	diags.Append(networkIDs.ElementsAs(ctx, &ids, false)...)
	if diags.HasError() {
		return nil, diags
	}
	return &client.FirewallZone{
		Name:       name,
		NetworkIDs: ids,
	}, diags
}
