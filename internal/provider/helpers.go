package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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
// networkAPIToModelFull populates all fields of a NetworkResourceModel from an API response.
func networkAPIToModelFull(ctx context.Context, result *client.Network, model *NetworkResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.Name = types.StringValue(result.Name)
	model.Management = types.StringValue(result.Management)
	model.Enabled = types.BoolValue(result.Enabled)
	model.VlanID = types.Int64Value(int64(result.VlanID))

	// Trusted DHCP server IPs.
	if result.DhcpGuarding != nil && len(result.DhcpGuarding.TrustedDhcpServerIPAddresses) > 0 {
		ips, d := types.ListValueFrom(ctx, types.StringType, result.DhcpGuarding.TrustedDhcpServerIPAddresses)
		diags.Append(d...)
		model.TrustedDhcpServerIPAddresses = ips
	} else {
		model.TrustedDhcpServerIPAddresses = types.ListNull(types.StringType)
	}

	// Bool pointer fields — map nil to null.
	model.IsolationEnabled = optionalBoolToTF(result.IsolationEnabled)
	model.CellularBackupEnabled = optionalBoolToTF(result.CellularBackupEnabled)
	model.InternetAccessEnabled = optionalBoolToTF(result.InternetAccessEnabled)
	model.MdnsForwardingEnabled = optionalBoolToTF(result.MdnsForwardingEnabled)

	// String fields — map empty to null.
	model.ZoneID = optionalStringToTF(result.ZoneID)
	model.DeviceID = optionalStringToTF(result.DeviceID)

	// IPv4 configuration.
	d := ipv4ConfigAPIToModel(ctx, result.IPv4Configuration, &model.IPv4Configuration)
	diags.Append(d...)

	return diags
}

// networkAPIToDataSourceModel populates all fields of a NetworkDataSourceModel from an API response.
func networkAPIToDataSourceModel(ctx context.Context, result *client.Network, model *NetworkDataSourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.Name = types.StringValue(result.Name)
	model.Management = types.StringValue(result.Management)
	model.Enabled = types.BoolValue(result.Enabled)
	model.VlanID = types.Int64Value(int64(result.VlanID))

	// Trusted DHCP server IPs.
	if result.DhcpGuarding != nil && len(result.DhcpGuarding.TrustedDhcpServerIPAddresses) > 0 {
		ips, d := types.ListValueFrom(ctx, types.StringType, result.DhcpGuarding.TrustedDhcpServerIPAddresses)
		diags.Append(d...)
		model.TrustedDhcpServerIPAddresses = ips
	} else {
		model.TrustedDhcpServerIPAddresses = types.ListNull(types.StringType)
	}

	model.IsolationEnabled = optionalBoolToTF(result.IsolationEnabled)
	model.CellularBackupEnabled = optionalBoolToTF(result.CellularBackupEnabled)
	model.InternetAccessEnabled = optionalBoolToTF(result.InternetAccessEnabled)
	model.MdnsForwardingEnabled = optionalBoolToTF(result.MdnsForwardingEnabled)
	model.ZoneID = optionalStringToTF(result.ZoneID)
	model.DeviceID = optionalStringToTF(result.DeviceID)

	d := ipv4ConfigAPIToModel(ctx, result.IPv4Configuration, &model.IPv4Configuration)
	diags.Append(d...)

	return diags
}

// ipv4ConfigAPIToModel converts a client.IPv4Config to a types.Object for the Terraform model.
func ipv4ConfigAPIToModel(ctx context.Context, cfg *client.IPv4Config, target *types.Object) diag.Diagnostics {
	var diags diag.Diagnostics

	if cfg == nil {
		*target = types.ObjectNull(ipv4ConfigurationAttrTypes())
		return diags
	}

	attrs := map[string]attr.Value{
		"auto_scale_enabled": types.BoolValue(cfg.AutoScaleEnabled),
		"host_ip_address":    types.StringValue(cfg.HostIPAddress),
		"prefix_length":      types.Int64Value(int64(cfg.PrefixLength)),
	}

	// DHCP configuration.
	if cfg.DhcpConfiguration != nil {
		dhcp := cfg.DhcpConfiguration
		attrs["dhcp_mode"] = types.StringValue(dhcp.Mode)

		// Server mode fields.
		if dhcp.IPAddressRange != nil {
			attrs["dhcp_start"] = types.StringValue(dhcp.IPAddressRange.Start)
			attrs["dhcp_stop"] = types.StringValue(dhcp.IPAddressRange.Stop)
		} else {
			attrs["dhcp_start"] = types.StringNull()
			attrs["dhcp_stop"] = types.StringNull()
		}

		if dhcp.LeaseTimeSeconds != nil {
			attrs["dhcp_lease_time_seconds"] = types.Int64Value(int64(*dhcp.LeaseTimeSeconds))
		} else {
			attrs["dhcp_lease_time_seconds"] = types.Int64Null()
		}

		if len(dhcp.DnsServerOverride) > 0 {
			dnsServers, d := types.ListValueFrom(ctx, types.StringType, dhcp.DnsServerOverride)
			diags.Append(d...)
			attrs["dhcp_dns_servers"] = dnsServers
		} else {
			attrs["dhcp_dns_servers"] = types.ListNull(types.StringType)
		}

		attrs["dhcp_gateway_override"] = optionalStringToTF(dhcp.GatewayIPOverride)
		attrs["dhcp_domain_name"] = optionalStringToTF(dhcp.DomainName)

		// Relay mode fields.
		if len(dhcp.ServerIPAddresses) > 0 {
			relayAddrs, d := types.ListValueFrom(ctx, types.StringType, dhcp.ServerIPAddresses)
			diags.Append(d...)
			attrs["dhcp_relay_addresses"] = relayAddrs
		} else {
			attrs["dhcp_relay_addresses"] = types.ListNull(types.StringType)
		}
	} else {
		attrs["dhcp_mode"] = types.StringNull()
		attrs["dhcp_start"] = types.StringNull()
		attrs["dhcp_stop"] = types.StringNull()
		attrs["dhcp_lease_time_seconds"] = types.Int64Null()
		attrs["dhcp_dns_servers"] = types.ListNull(types.StringType)
		attrs["dhcp_gateway_override"] = types.StringNull()
		attrs["dhcp_domain_name"] = types.StringNull()
		attrs["dhcp_relay_addresses"] = types.ListNull(types.StringType)
	}

	obj, d := types.ObjectValue(ipv4ConfigurationAttrTypes(), attrs)
	diags.Append(d...)
	*target = obj

	return diags
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

	// Trusted DHCP server IPs.
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

	// Bool pointer fields.
	network.IsolationEnabled = tfBoolToOptional(model.IsolationEnabled)
	network.CellularBackupEnabled = tfBoolToOptional(model.CellularBackupEnabled)
	network.InternetAccessEnabled = tfBoolToOptional(model.InternetAccessEnabled)
	network.MdnsForwardingEnabled = tfBoolToOptional(model.MdnsForwardingEnabled)

	// String fields.
	if !model.ZoneID.IsNull() && !model.ZoneID.IsUnknown() {
		network.ZoneID = model.ZoneID.ValueString()
	}
	if !model.DeviceID.IsNull() && !model.DeviceID.IsUnknown() {
		network.DeviceID = model.DeviceID.ValueString()
	}

	// IPv4 configuration.
	if !model.IPv4Configuration.IsNull() && !model.IPv4Configuration.IsUnknown() {
		var ipv4Model IPv4ConfigurationModel
		diags.Append(model.IPv4Configuration.As(ctx, &ipv4Model, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}

		ipv4Config := &client.IPv4Config{
			AutoScaleEnabled: ipv4Model.AutoScaleEnabled.ValueBool(),
			HostIPAddress:    ipv4Model.HostIPAddress.ValueString(),
			PrefixLength:     int(ipv4Model.PrefixLength.ValueInt64()),
		}

		// Build DHCP configuration if mode is set.
		if !ipv4Model.DhcpMode.IsNull() && !ipv4Model.DhcpMode.IsUnknown() {
			dhcpConfig := &client.DhcpConfig{
				Mode: ipv4Model.DhcpMode.ValueString(),
			}

			// Server mode fields.
			if !ipv4Model.DhcpStart.IsNull() && !ipv4Model.DhcpStart.IsUnknown() &&
				!ipv4Model.DhcpStop.IsNull() && !ipv4Model.DhcpStop.IsUnknown() {
				dhcpConfig.IPAddressRange = &client.DhcpRange{
					Start: ipv4Model.DhcpStart.ValueString(),
					Stop:  ipv4Model.DhcpStop.ValueString(),
				}
			}

			if !ipv4Model.DhcpLeaseTimeSeconds.IsNull() && !ipv4Model.DhcpLeaseTimeSeconds.IsUnknown() {
				leaseTime := int(ipv4Model.DhcpLeaseTimeSeconds.ValueInt64())
				dhcpConfig.LeaseTimeSeconds = &leaseTime
			}

			if !ipv4Model.DhcpDnsServers.IsNull() && !ipv4Model.DhcpDnsServers.IsUnknown() {
				var dnsServers []string
				diags.Append(ipv4Model.DhcpDnsServers.ElementsAs(ctx, &dnsServers, false)...)
				if diags.HasError() {
					return nil, diags
				}
				dhcpConfig.DnsServerOverride = dnsServers
			}

			if !ipv4Model.DhcpGatewayOverride.IsNull() && !ipv4Model.DhcpGatewayOverride.IsUnknown() {
				dhcpConfig.GatewayIPOverride = ipv4Model.DhcpGatewayOverride.ValueString()
			}

			if !ipv4Model.DhcpDomainName.IsNull() && !ipv4Model.DhcpDomainName.IsUnknown() {
				dhcpConfig.DomainName = ipv4Model.DhcpDomainName.ValueString()
			}

			// Relay mode fields.
			if !ipv4Model.DhcpRelayAddresses.IsNull() && !ipv4Model.DhcpRelayAddresses.IsUnknown() {
				var relayAddrs []string
				diags.Append(ipv4Model.DhcpRelayAddresses.ElementsAs(ctx, &relayAddrs, false)...)
				if diags.HasError() {
					return nil, diags
				}
				dhcpConfig.ServerIPAddresses = relayAddrs
			}

			ipv4Config.DhcpConfiguration = dhcpConfig
		}

		network.IPv4Configuration = ipv4Config
	}

	return network, diags
}

// optionalBoolToTF maps a *bool to a types.Bool (null if nil).
func optionalBoolToTF(b *bool) types.Bool {
	if b == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*b)
}

// tfBoolToOptional maps a types.Bool to a *bool (nil if null/unknown).
func tfBoolToOptional(b types.Bool) *bool {
	if b.IsNull() || b.IsUnknown() {
		return nil
	}
	v := b.ValueBool()
	return &v
}

// optionalStringToTF maps a string to types.String (null if empty).
func optionalStringToTF(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

// wifiBroadcastAPIToModel maps a client.WifiBroadcast to common Terraform model fields.
// It populates the provided WifiBroadcastResourceModel in place.
func wifiBroadcastAPIToModel(ctx context.Context, model *WifiBroadcastResourceModel, result *client.WifiBroadcast) diag.Diagnostics {
	var diags diag.Diagnostics

	model.Type = types.StringValue(result.Type)
	model.Name = types.StringValue(result.Name)
	model.Enabled = types.BoolValue(result.Enabled)
	model.ClientIsolationEnabled = types.BoolValue(result.ClientIsolationEnabled)
	model.HideName = types.BoolValue(result.HideName)
	model.MulticastToUnicastConversionEnabled = types.BoolValue(result.MulticastToUnicastConversionEnabled)
	model.UapsdEnabled = types.BoolValue(result.UapsdEnabled)

	if result.SecurityConfiguration != nil {
		model.SecurityType = types.StringValue(result.SecurityConfiguration.Type)
	}
	if result.Network != nil {
		model.NetworkType = types.StringValue(result.Network.Type)
		if result.Network.NetworkID != "" {
			model.NetworkID = types.StringValue(result.Network.NetworkID)
		} else {
			model.NetworkID = types.StringNull()
		}
	}

	// Basic data rates.
	if result.BasicDataRateKbps != nil {
		model.BasicDataRateGHz24 = types.Int64Value(int64(result.BasicDataRateKbps.GHz24))
		model.BasicDataRateGHz5 = types.Int64Value(int64(result.BasicDataRateKbps.GHz5))
	} else {
		model.BasicDataRateGHz24 = types.Int64Null()
		model.BasicDataRateGHz5 = types.Int64Null()
	}

	// Client filtering policy.
	if result.ClientFilteringPolicy != nil {
		model.ClientFilterAction = types.StringValue(result.ClientFilteringPolicy.Action)
		if result.ClientFilteringPolicy.MacAddressFilter != nil {
			macs, d := types.ListValueFrom(ctx, types.StringType, result.ClientFilteringPolicy.MacAddressFilter)
			diags.Append(d...)
			model.ClientFilterMacAddresses = macs
		} else {
			model.ClientFilterMacAddresses = types.ListNull(types.StringType)
		}
	} else {
		model.ClientFilterAction = types.StringNull()
		model.ClientFilterMacAddresses = types.ListNull(types.StringType)
	}

	// Blackout schedule.
	dayAttrTypes := blackoutScheduleDayAttrTypes()
	if result.BlackoutSchedule != nil && len(result.BlackoutSchedule.Days) > 0 {
		dayObjects := make([]attr.Value, 0, len(result.BlackoutSchedule.Days))
		for _, day := range result.BlackoutSchedule.Days {
			attrs := map[string]attr.Value{
				"type": types.StringValue(day.Type),
				"day":  types.StringValue(day.Day),
			}
			if len(day.TimeRanges) > 0 {
				attrs["start_time"] = types.StringValue(day.TimeRanges[0].StartTime)
				attrs["end_time"] = types.StringValue(day.TimeRanges[0].EndTime)
			} else {
				attrs["start_time"] = types.StringNull()
				attrs["end_time"] = types.StringNull()
			}
			obj, d := types.ObjectValue(dayAttrTypes, attrs)
			diags.Append(d...)
			dayObjects = append(dayObjects, obj)
		}
		daysList, d := types.ListValue(types.ObjectType{AttrTypes: dayAttrTypes}, dayObjects)
		diags.Append(d...)
		model.BlackoutScheduleDays = daysList
	} else {
		model.BlackoutScheduleDays = types.ListNull(types.ObjectType{AttrTypes: dayAttrTypes})
	}

	// Broadcasting frequencies.
	if result.BroadcastingFrequenciesGHz != nil {
		freqs, d := types.ListValueFrom(ctx, types.Float64Type, result.BroadcastingFrequenciesGHz)
		diags.Append(d...)
		model.BroadcastingFrequenciesGHz = freqs
	} else {
		model.BroadcastingFrequenciesGHz = types.ListNull(types.Float64Type)
	}

	// Broadcasting device filter.
	if result.BroadcastingDeviceFilter != nil {
		model.BroadcastingDeviceFilterType = types.StringValue(result.BroadcastingDeviceFilter.Type)
		var ids []string
		if result.BroadcastingDeviceFilter.Type == client.DeviceFilterDevices {
			ids = result.BroadcastingDeviceFilter.DeviceIDs
		} else {
			ids = result.BroadcastingDeviceFilter.DeviceTagIDs
		}
		if ids != nil {
			idList, d := types.ListValueFrom(ctx, types.StringType, ids)
			diags.Append(d...)
			model.BroadcastingDeviceFilterIds = idList
		} else {
			model.BroadcastingDeviceFilterIds = types.ListNull(types.StringType)
		}
	} else {
		model.BroadcastingDeviceFilterType = types.StringNull()
		model.BroadcastingDeviceFilterIds = types.ListNull(types.StringType)
	}

	// Multicast filtering policy.
	if result.MulticastFilteringPolicy != nil {
		model.MulticastFilterAction = types.StringValue(result.MulticastFilteringPolicy.Action)
	} else {
		model.MulticastFilterAction = types.StringNull()
	}

	// mDNS proxy configuration.
	if result.MdnsProxyConfiguration != nil {
		model.MdnsProxyMode = types.StringValue(result.MdnsProxyConfiguration.Mode)
	} else {
		model.MdnsProxyMode = types.StringNull()
	}

	// STANDARD-only boolean fields.
	model.BandSteeringEnabled = optionalBoolToTF(result.BandSteeringEnabled)
	model.MloEnabled = optionalBoolToTF(result.MloEnabled)
	model.ArpProxyEnabled = optionalBoolToTF(result.ArpProxyEnabled)
	model.BssTransitionEnabled = optionalBoolToTF(result.BssTransitionEnabled)
	model.AdvertiseDeviceName = optionalBoolToTF(result.AdvertiseDeviceName)

	return diags
}

// wifiBroadcastAPIToDataSourceModel maps a client.WifiBroadcast to a WifiBroadcastDataSourceModel.
func wifiBroadcastAPIToDataSourceModel(ctx context.Context, model *WifiBroadcastDataSourceModel, result *client.WifiBroadcast) diag.Diagnostics {
	var diags diag.Diagnostics

	model.Type = types.StringValue(result.Type)
	model.Name = types.StringValue(result.Name)
	model.Enabled = types.BoolValue(result.Enabled)
	model.ClientIsolationEnabled = types.BoolValue(result.ClientIsolationEnabled)
	model.HideName = types.BoolValue(result.HideName)
	model.MulticastToUnicastConversionEnabled = types.BoolValue(result.MulticastToUnicastConversionEnabled)
	model.UapsdEnabled = types.BoolValue(result.UapsdEnabled)

	if result.SecurityConfiguration != nil {
		model.SecurityType = types.StringValue(result.SecurityConfiguration.Type)
	}
	if result.Network != nil {
		model.NetworkType = types.StringValue(result.Network.Type)
		if result.Network.NetworkID != "" {
			model.NetworkID = types.StringValue(result.Network.NetworkID)
		} else {
			model.NetworkID = types.StringNull()
		}
	}

	// Basic data rates.
	if result.BasicDataRateKbps != nil {
		model.BasicDataRateGHz24 = types.Int64Value(int64(result.BasicDataRateKbps.GHz24))
		model.BasicDataRateGHz5 = types.Int64Value(int64(result.BasicDataRateKbps.GHz5))
	} else {
		model.BasicDataRateGHz24 = types.Int64Null()
		model.BasicDataRateGHz5 = types.Int64Null()
	}

	// Client filtering policy.
	if result.ClientFilteringPolicy != nil {
		model.ClientFilterAction = types.StringValue(result.ClientFilteringPolicy.Action)
		if result.ClientFilteringPolicy.MacAddressFilter != nil {
			macs, d := types.ListValueFrom(ctx, types.StringType, result.ClientFilteringPolicy.MacAddressFilter)
			diags.Append(d...)
			model.ClientFilterMacAddresses = macs
		} else {
			model.ClientFilterMacAddresses = types.ListNull(types.StringType)
		}
	} else {
		model.ClientFilterAction = types.StringNull()
		model.ClientFilterMacAddresses = types.ListNull(types.StringType)
	}

	// Blackout schedule.
	dayAttrTypes := blackoutScheduleDayAttrTypes()
	if result.BlackoutSchedule != nil && len(result.BlackoutSchedule.Days) > 0 {
		dayObjects := make([]attr.Value, 0, len(result.BlackoutSchedule.Days))
		for _, day := range result.BlackoutSchedule.Days {
			attrs := map[string]attr.Value{
				"type": types.StringValue(day.Type),
				"day":  types.StringValue(day.Day),
			}
			if len(day.TimeRanges) > 0 {
				attrs["start_time"] = types.StringValue(day.TimeRanges[0].StartTime)
				attrs["end_time"] = types.StringValue(day.TimeRanges[0].EndTime)
			} else {
				attrs["start_time"] = types.StringNull()
				attrs["end_time"] = types.StringNull()
			}
			obj, d := types.ObjectValue(dayAttrTypes, attrs)
			diags.Append(d...)
			dayObjects = append(dayObjects, obj)
		}
		daysList, d := types.ListValue(types.ObjectType{AttrTypes: dayAttrTypes}, dayObjects)
		diags.Append(d...)
		model.BlackoutScheduleDays = daysList
	} else {
		model.BlackoutScheduleDays = types.ListNull(types.ObjectType{AttrTypes: dayAttrTypes})
	}

	// Broadcasting frequencies.
	if result.BroadcastingFrequenciesGHz != nil {
		freqs, d := types.ListValueFrom(ctx, types.Float64Type, result.BroadcastingFrequenciesGHz)
		diags.Append(d...)
		model.BroadcastingFrequenciesGHz = freqs
	} else {
		model.BroadcastingFrequenciesGHz = types.ListNull(types.Float64Type)
	}

	// Broadcasting device filter.
	if result.BroadcastingDeviceFilter != nil {
		model.BroadcastingDeviceFilterType = types.StringValue(result.BroadcastingDeviceFilter.Type)
		var ids []string
		if result.BroadcastingDeviceFilter.Type == client.DeviceFilterDevices {
			ids = result.BroadcastingDeviceFilter.DeviceIDs
		} else {
			ids = result.BroadcastingDeviceFilter.DeviceTagIDs
		}
		if ids != nil {
			idList, d := types.ListValueFrom(ctx, types.StringType, ids)
			diags.Append(d...)
			model.BroadcastingDeviceFilterIds = idList
		} else {
			model.BroadcastingDeviceFilterIds = types.ListNull(types.StringType)
		}
	} else {
		model.BroadcastingDeviceFilterType = types.StringNull()
		model.BroadcastingDeviceFilterIds = types.ListNull(types.StringType)
	}

	// Multicast filtering policy.
	if result.MulticastFilteringPolicy != nil {
		model.MulticastFilterAction = types.StringValue(result.MulticastFilteringPolicy.Action)
	} else {
		model.MulticastFilterAction = types.StringNull()
	}

	// mDNS proxy configuration.
	if result.MdnsProxyConfiguration != nil {
		model.MdnsProxyMode = types.StringValue(result.MdnsProxyConfiguration.Mode)
	} else {
		model.MdnsProxyMode = types.StringNull()
	}

	// STANDARD-only boolean fields.
	model.BandSteeringEnabled = optionalBoolToTF(result.BandSteeringEnabled)
	model.MloEnabled = optionalBoolToTF(result.MloEnabled)
	model.ArpProxyEnabled = optionalBoolToTF(result.ArpProxyEnabled)
	model.BssTransitionEnabled = optionalBoolToTF(result.BssTransitionEnabled)
	model.AdvertiseDeviceName = optionalBoolToTF(result.AdvertiseDeviceName)

	return diags
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
// passphraseWO carries the write-only passphrase value read from the config
// (the framework nulls it out in plan/state).
func wifiBroadcastModelToAPI(ctx context.Context, plan WifiBroadcastResourceModel, passphraseWO types.String) (*client.WifiBroadcast, diag.Diagnostics) {
	var diags diag.Diagnostics

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

	// Resolve the passphrase: prefer the write-only passphrase_wo, fall back
	// to the legacy passphrase attribute from the plan.
	if !passphraseWO.IsNull() && !passphraseWO.IsUnknown() {
		broadcast.SecurityConfiguration.Passphrase = passphraseWO.ValueString()
	} else if !plan.Passphrase.IsNull() && !plan.Passphrase.IsUnknown() {
		broadcast.SecurityConfiguration.Passphrase = plan.Passphrase.ValueString()
	}

	if !plan.NetworkID.IsNull() && !plan.NetworkID.IsUnknown() {
		broadcast.Network.NetworkID = plan.NetworkID.ValueString()
	}

	// Basic data rates.
	if !plan.BasicDataRateGHz24.IsNull() && !plan.BasicDataRateGHz24.IsUnknown() ||
		!plan.BasicDataRateGHz5.IsNull() && !plan.BasicDataRateGHz5.IsUnknown() {
		broadcast.BasicDataRateKbps = &client.BasicDataRateKbps{}
		if !plan.BasicDataRateGHz24.IsNull() && !plan.BasicDataRateGHz24.IsUnknown() {
			broadcast.BasicDataRateKbps.GHz24 = int(plan.BasicDataRateGHz24.ValueInt64())
		}
		if !plan.BasicDataRateGHz5.IsNull() && !plan.BasicDataRateGHz5.IsUnknown() {
			broadcast.BasicDataRateKbps.GHz5 = int(plan.BasicDataRateGHz5.ValueInt64())
		}
	}

	// Client filtering policy.
	if !plan.ClientFilterAction.IsNull() && !plan.ClientFilterAction.IsUnknown() {
		broadcast.ClientFilteringPolicy = &client.ClientFilteringPolicy{
			Action: plan.ClientFilterAction.ValueString(),
		}
		if !plan.ClientFilterMacAddresses.IsNull() && !plan.ClientFilterMacAddresses.IsUnknown() {
			var macs []string
			diags.Append(plan.ClientFilterMacAddresses.ElementsAs(ctx, &macs, false)...)
			broadcast.ClientFilteringPolicy.MacAddressFilter = macs
		}
	}

	// Blackout schedule.
	if !plan.BlackoutScheduleDays.IsNull() && !plan.BlackoutScheduleDays.IsUnknown() {
		var dayModels []BlackoutScheduleDayModel
		diags.Append(plan.BlackoutScheduleDays.ElementsAs(ctx, &dayModels, false)...)
		if !diags.HasError() {
			days := make([]client.BlackoutDay, 0, len(dayModels))
			for _, dm := range dayModels {
				day := client.BlackoutDay{
					Type: dm.Type.ValueString(),
					Day:  dm.Day.ValueString(),
				}
				if !dm.StartTime.IsNull() && !dm.StartTime.IsUnknown() &&
					!dm.EndTime.IsNull() && !dm.EndTime.IsUnknown() {
					day.TimeRanges = []client.TimeRange{
						{
							StartTime: dm.StartTime.ValueString(),
							EndTime:   dm.EndTime.ValueString(),
						},
					}
				}
				days = append(days, day)
			}
			broadcast.BlackoutSchedule = &client.BlackoutSchedule{Days: days}
		}
	}

	// Broadcasting frequencies.
	if !plan.BroadcastingFrequenciesGHz.IsNull() && !plan.BroadcastingFrequenciesGHz.IsUnknown() {
		var freqs []float64
		diags.Append(plan.BroadcastingFrequenciesGHz.ElementsAs(ctx, &freqs, false)...)
		broadcast.BroadcastingFrequenciesGHz = freqs
	}

	// Broadcasting device filter.
	if !plan.BroadcastingDeviceFilterType.IsNull() && !plan.BroadcastingDeviceFilterType.IsUnknown() {
		filter := &client.BroadcastingDeviceFilter{
			Type: plan.BroadcastingDeviceFilterType.ValueString(),
		}
		if !plan.BroadcastingDeviceFilterIds.IsNull() && !plan.BroadcastingDeviceFilterIds.IsUnknown() {
			var ids []string
			diags.Append(plan.BroadcastingDeviceFilterIds.ElementsAs(ctx, &ids, false)...)
			if filter.Type == client.DeviceFilterDevices {
				filter.DeviceIDs = ids
			} else {
				filter.DeviceTagIDs = ids
			}
		}
		broadcast.BroadcastingDeviceFilter = filter
	}

	// Multicast filtering policy.
	if !plan.MulticastFilterAction.IsNull() && !plan.MulticastFilterAction.IsUnknown() {
		broadcast.MulticastFilteringPolicy = &client.MulticastFilteringPolicy{
			Action: plan.MulticastFilterAction.ValueString(),
		}
	}

	// mDNS proxy configuration.
	if !plan.MdnsProxyMode.IsNull() && !plan.MdnsProxyMode.IsUnknown() {
		broadcast.MdnsProxyConfiguration = &client.MdnsProxyConfiguration{
			Mode: plan.MdnsProxyMode.ValueString(),
		}
	}

	// STANDARD-only boolean pointer fields.
	broadcast.BandSteeringEnabled = tfBoolToOptional(plan.BandSteeringEnabled)
	broadcast.MloEnabled = tfBoolToOptional(plan.MloEnabled)
	broadcast.ArpProxyEnabled = tfBoolToOptional(plan.ArpProxyEnabled)
	broadcast.BssTransitionEnabled = tfBoolToOptional(plan.BssTransitionEnabled)
	broadcast.AdvertiseDeviceName = tfBoolToOptional(plan.AdvertiseDeviceName)

	return broadcast, diags
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

// ============================================================================
// ACL Rule helpers
// ============================================================================

// aclRuleAPIToModel maps a client.AclRule to the Terraform resource model.
func aclRuleAPIToModel(ctx context.Context, result *client.AclRule, model *AclRuleResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.Type = types.StringValue(result.Type)
	model.Enabled = types.BoolValue(result.Enabled)
	model.Name = types.StringValue(result.Name)
	model.Action = types.StringValue(result.Action)

	if result.Description != "" {
		model.Description = types.StringValue(result.Description)
	} else {
		model.Description = types.StringNull()
	}

	// Source filter
	if result.SourceFilter != nil {
		model.SourceFilterType = types.StringValue(result.SourceFilter.Type)
		srcValues, srcPorts, d := aclFilterAPIToModel(ctx, result.SourceFilter)
		diags.Append(d...)
		model.SourceFilterValues = srcValues
		model.SourceFilterPorts = srcPorts
	} else {
		model.SourceFilterType = types.StringNull()
		model.SourceFilterValues = types.ListNull(types.StringType)
		model.SourceFilterPorts = types.ListNull(types.Int64Type)
	}

	// Destination filter
	if result.DestinationFilter != nil {
		model.DestinationFilterType = types.StringValue(result.DestinationFilter.Type)
		dstValues, dstPorts, d := aclFilterAPIToModel(ctx, result.DestinationFilter)
		diags.Append(d...)
		model.DestinationFilterValues = dstValues
		model.DestinationFilterPorts = dstPorts
	} else {
		model.DestinationFilterType = types.StringNull()
		model.DestinationFilterValues = types.ListNull(types.StringType)
		model.DestinationFilterPorts = types.ListNull(types.Int64Type)
	}

	// Protocol filter
	if len(result.ProtocolFilter) > 0 {
		pf, d := types.ListValueFrom(ctx, types.StringType, result.ProtocolFilter)
		diags.Append(d...)
		model.ProtocolFilter = pf
	} else {
		model.ProtocolFilter = types.ListNull(types.StringType)
	}

	// Enforcing device filter
	if result.EnforcingDeviceFilter != nil && len(result.EnforcingDeviceFilter.DeviceIDs) > 0 {
		ids, d := types.ListValueFrom(ctx, types.StringType, result.EnforcingDeviceFilter.DeviceIDs)
		diags.Append(d...)
		model.EnforcingDeviceIDs = ids
	} else {
		model.EnforcingDeviceIDs = types.ListNull(types.StringType)
	}

	// Network ID filter
	if result.NetworkIDFilter != "" {
		model.NetworkIDFilter = types.StringValue(result.NetworkIDFilter)
	} else {
		model.NetworkIDFilter = types.StringNull()
	}

	return diags
}

// aclFilterAPIToModel converts an AclFilter to list values for the Terraform model.
func aclFilterAPIToModel(ctx context.Context, filter *client.AclFilter) (values types.List, ports types.List, diags diag.Diagnostics) {

	// Collect string values from all possible string fields.
	var stringValues []string
	stringValues = append(stringValues, filter.IPAddressesOrSubnets...)
	stringValues = append(stringValues, filter.NetworkIDs...)
	stringValues = append(stringValues, filter.MacAddresses...)

	if len(stringValues) > 0 {
		v, d := types.ListValueFrom(ctx, types.StringType, stringValues)
		diags.Append(d...)
		values = v
	} else {
		values = types.ListNull(types.StringType)
	}

	// Collect port numbers.
	allPorts := filter.PortNumbers
	if len(allPorts) == 0 {
		allPorts = filter.PortFilter
	}
	if len(allPorts) > 0 {
		int64Ports := make([]int64, len(allPorts))
		for i, p := range allPorts {
			int64Ports[i] = int64(p)
		}
		p, d := types.ListValueFrom(ctx, types.Int64Type, int64Ports)
		diags.Append(d...)
		ports = p
	} else {
		ports = types.ListNull(types.Int64Type)
	}

	return
}

// aclRuleModelToAPI converts an AclRuleResourceModel into a client.AclRule.
func aclRuleModelToAPI(ctx context.Context, model AclRuleResourceModel) (*client.AclRule, diag.Diagnostics) {
	var diags diag.Diagnostics

	rule := &client.AclRule{
		Type:    model.Type.ValueString(),
		Enabled: model.Enabled.ValueBool(),
		Name:    model.Name.ValueString(),
		Action:  model.Action.ValueString(),
	}

	if !model.Description.IsNull() {
		rule.Description = model.Description.ValueString()
	}

	// Source filter
	if !model.SourceFilterType.IsNull() {
		rule.SourceFilter = &client.AclFilter{
			Type: model.SourceFilterType.ValueString(),
		}
		diags.Append(populateAclFilter(ctx, rule.SourceFilter, model.SourceFilterType.ValueString(), model.SourceFilterValues, model.SourceFilterPorts)...)
	}

	// Destination filter
	if !model.DestinationFilterType.IsNull() {
		rule.DestinationFilter = &client.AclFilter{
			Type: model.DestinationFilterType.ValueString(),
		}
		diags.Append(populateAclFilter(ctx, rule.DestinationFilter, model.DestinationFilterType.ValueString(), model.DestinationFilterValues, model.DestinationFilterPorts)...)
	}

	// Protocol filter
	if !model.ProtocolFilter.IsNull() {
		var protocols []string
		diags.Append(model.ProtocolFilter.ElementsAs(ctx, &protocols, false)...)
		rule.ProtocolFilter = protocols
	}

	// Enforcing device filter
	if !model.EnforcingDeviceIDs.IsNull() {
		var ids []string
		diags.Append(model.EnforcingDeviceIDs.ElementsAs(ctx, &ids, false)...)
		rule.EnforcingDeviceFilter = &client.AclDeviceFilter{
			Type:      "SPECIFIC",
			DeviceIDs: ids,
		}
	}

	// Network ID filter
	if !model.NetworkIDFilter.IsNull() {
		rule.NetworkIDFilter = model.NetworkIDFilter.ValueString()
	}

	return rule, diags
}

// populateAclFilter sets the appropriate fields on an AclFilter based on the filter type.
func populateAclFilter(ctx context.Context, filter *client.AclFilter, filterType string, values types.List, ports types.List) diag.Diagnostics {
	var diags diag.Diagnostics

	if !values.IsNull() {
		var vals []string
		diags.Append(values.ElementsAs(ctx, &vals, false)...)

		switch filterType {
		case client.AclFilterIPAddressesOrSubnets:
			filter.IPAddressesOrSubnets = vals
		case client.AclFilterNetworks:
			filter.NetworkIDs = vals
		case client.AclFilterMacAddresses:
			filter.MacAddresses = vals
		}
	}

	if !ports.IsNull() {
		var portValues []int64
		diags.Append(ports.ElementsAs(ctx, &portValues, false)...)
		intPorts := make([]int, len(portValues))
		for i, p := range portValues {
			intPorts[i] = int(p)
		}
		filter.PortNumbers = intPorts
	}

	return diags
}

// ============================================================================
// Firewall Policy helpers
// ============================================================================

// firewallPolicyAPIToModel maps a client.FirewallPolicy to the Terraform resource model.
func firewallPolicyAPIToModel(ctx context.Context, result *client.FirewallPolicy, model *FirewallPolicyResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.Enabled = types.BoolValue(result.Enabled)
	model.Name = types.StringValue(result.Name)
	model.LoggingEnabled = types.BoolValue(result.LoggingEnabled)

	if result.Description != "" {
		model.Description = types.StringValue(result.Description)
	} else {
		model.Description = types.StringNull()
	}

	// Action
	if result.Action != nil {
		model.ActionType = types.StringValue(result.Action.Type)
		if result.Action.AllowReturnTraffic != nil {
			model.AllowReturnTraffic = types.BoolValue(*result.Action.AllowReturnTraffic)
		} else {
			model.AllowReturnTraffic = types.BoolNull()
		}
	}

	// Source
	if result.Source != nil {
		model.SourceZoneID = types.StringValue(result.Source.ZoneID)
		if result.Source.TrafficFilter != nil {
			model.SourceTrafficFilterType = types.StringValue(result.Source.TrafficFilter.Type)
			model.SourceTrafficFilterValues = firewallTrafficFilterValues(ctx, result.Source.TrafficFilter, &diags)
		} else {
			model.SourceTrafficFilterType = types.StringNull()
			model.SourceTrafficFilterValues = types.ListNull(types.StringType)
		}
	}

	// Destination
	if result.Destination != nil {
		model.DestinationZoneID = types.StringValue(result.Destination.ZoneID)
		if result.Destination.TrafficFilter != nil {
			model.DestinationTrafficFilterType = types.StringValue(result.Destination.TrafficFilter.Type)
			model.DestinationTrafficFilterValues = firewallTrafficFilterValues(ctx, result.Destination.TrafficFilter, &diags)
		} else {
			model.DestinationTrafficFilterType = types.StringNull()
			model.DestinationTrafficFilterValues = types.ListNull(types.StringType)
		}
	}

	// IP Protocol Scope
	if result.IPProtocolScope != nil {
		model.IPVersion = types.StringValue(result.IPProtocolScope.IPVersion)
	}

	// Connection state filter
	if len(result.ConnectionStateFilter) > 0 {
		csf, d := types.ListValueFrom(ctx, types.StringType, result.ConnectionStateFilter)
		diags.Append(d...)
		model.ConnectionStateFilter = csf
	} else {
		model.ConnectionStateFilter = types.ListNull(types.StringType)
	}

	// IPsec filter
	if result.IpsecFilter != "" {
		model.IpsecFilter = types.StringValue(result.IpsecFilter)
	} else {
		model.IpsecFilter = types.StringNull()
	}

	return diags
}

// firewallTrafficFilterValues extracts string values from a FirewallTrafficFilter.
func firewallTrafficFilterValues(ctx context.Context, filter *client.FirewallTrafficFilter, diags *diag.Diagnostics) types.List {
	var values []string

	switch filter.Type {
	case client.TrafficFilterNetwork:
		if filter.NetworkID != "" {
			values = append(values, filter.NetworkID)
		}
	case client.TrafficFilterIPAddress:
		values = append(values, filter.IPAddresses...)
	case client.TrafficFilterMacAddress:
		values = append(values, filter.MacAddresses...)
	}

	if len(values) > 0 {
		v, d := types.ListValueFrom(ctx, types.StringType, values)
		diags.Append(d...)
		return v
	}
	return types.ListNull(types.StringType)
}

// firewallPolicyModelToAPI converts a FirewallPolicyResourceModel into a client.FirewallPolicy.
func firewallPolicyModelToAPI(ctx context.Context, model FirewallPolicyResourceModel) (*client.FirewallPolicy, diag.Diagnostics) {
	var diags diag.Diagnostics

	policy := &client.FirewallPolicy{
		Enabled:        model.Enabled.ValueBool(),
		Name:           model.Name.ValueString(),
		LoggingEnabled: model.LoggingEnabled.ValueBool(),
	}

	if !model.Description.IsNull() {
		policy.Description = model.Description.ValueString()
	}

	// Action
	policy.Action = &client.FirewallAction{
		Type: model.ActionType.ValueString(),
	}
	if !model.AllowReturnTraffic.IsNull() {
		v := model.AllowReturnTraffic.ValueBool()
		policy.Action.AllowReturnTraffic = &v
	}

	// Source
	policy.Source = &client.FirewallEndpoint{
		ZoneID: model.SourceZoneID.ValueString(),
	}
	if !model.SourceTrafficFilterType.IsNull() {
		policy.Source.TrafficFilter = buildFirewallTrafficFilter(ctx, model.SourceTrafficFilterType.ValueString(), model.SourceTrafficFilterValues, &diags)
	}

	// Destination
	policy.Destination = &client.FirewallEndpoint{
		ZoneID: model.DestinationZoneID.ValueString(),
	}
	if !model.DestinationTrafficFilterType.IsNull() {
		policy.Destination.TrafficFilter = buildFirewallTrafficFilter(ctx, model.DestinationTrafficFilterType.ValueString(), model.DestinationTrafficFilterValues, &diags)
	}

	// IP Protocol Scope
	policy.IPProtocolScope = &client.IPProtocolScope{
		IPVersion: model.IPVersion.ValueString(),
	}

	// Connection state filter
	if !model.ConnectionStateFilter.IsNull() {
		var states []string
		diags.Append(model.ConnectionStateFilter.ElementsAs(ctx, &states, false)...)
		policy.ConnectionStateFilter = states
	}

	// IPsec filter
	if !model.IpsecFilter.IsNull() {
		policy.IpsecFilter = model.IpsecFilter.ValueString()
	}

	return policy, diags
}

// buildFirewallTrafficFilter constructs a FirewallTrafficFilter from model values.
func buildFirewallTrafficFilter(ctx context.Context, filterType string, values types.List, diags *diag.Diagnostics) *client.FirewallTrafficFilter {
	filter := &client.FirewallTrafficFilter{
		Type: filterType,
	}

	if !values.IsNull() {
		var vals []string
		diags.Append(values.ElementsAs(ctx, &vals, false)...)

		switch filterType {
		case client.TrafficFilterNetwork:
			if len(vals) > 0 {
				filter.NetworkID = vals[0]
			}
		case client.TrafficFilterIPAddress:
			filter.IPAddresses = vals
		case client.TrafficFilterMacAddress:
			filter.MacAddresses = vals
		}
	}

	return filter
}

// ============================================================================
// Traffic Matching List helpers
// ============================================================================

// trafficMatchingListAPIToModel maps a client.TrafficMatchingList to the Terraform resource model.
func trafficMatchingListAPIToModel(ctx context.Context, result *client.TrafficMatchingList, model *TrafficMatchingListResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.Type = types.StringValue(result.Type)
	model.Name = types.StringValue(result.Name)

	items := make([]TrafficMatchingItemModel, len(result.Items))
	for i, item := range result.Items {
		items[i] = TrafficMatchingItemModel{
			Type: types.StringValue(item.Type),
		}

		if item.Value != "" {
			items[i].Value = types.StringValue(item.Value)
		} else {
			items[i].Value = types.StringNull()
		}

		if item.Subnet != "" {
			items[i].Subnet = types.StringValue(item.Subnet)
		} else {
			items[i].Subnet = types.StringNull()
		}

		if item.StartIP != "" {
			items[i].Start = types.StringValue(item.StartIP)
		} else {
			items[i].Start = types.StringNull()
		}

		if item.EndIP != "" {
			items[i].End = types.StringValue(item.EndIP)
		} else {
			items[i].End = types.StringNull()
		}

		if item.PortNumber != nil {
			items[i].PortNumber = types.Int64Value(int64(*item.PortNumber))
		} else {
			items[i].PortNumber = types.Int64Null()
		}

		if item.StartPort != nil {
			items[i].StartPort = types.Int64Value(int64(*item.StartPort))
		} else {
			items[i].StartPort = types.Int64Null()
		}

		if item.EndPort != nil {
			items[i].EndPort = types.Int64Value(int64(*item.EndPort))
		} else {
			items[i].EndPort = types.Int64Null()
		}
	}

	itemsList, d := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: trafficMatchingItemAttrTypes()}, items)
	diags.Append(d...)
	model.Items = itemsList

	return diags
}

// trafficMatchingListModelToAPI converts a TrafficMatchingListResourceModel into a client.TrafficMatchingList.
func trafficMatchingListModelToAPI(ctx context.Context, model TrafficMatchingListResourceModel) (*client.TrafficMatchingList, diag.Diagnostics) {
	var diags diag.Diagnostics

	list := &client.TrafficMatchingList{
		Type: model.Type.ValueString(),
		Name: model.Name.ValueString(),
	}

	var items []TrafficMatchingItemModel
	diags.Append(model.Items.ElementsAs(ctx, &items, false)...)
	if diags.HasError() {
		return nil, diags
	}

	list.Items = make([]client.TrafficMatchingItem, len(items))
	for i, item := range items {
		list.Items[i] = client.TrafficMatchingItem{
			Type: item.Type.ValueString(),
		}

		if !item.Value.IsNull() {
			list.Items[i].Value = item.Value.ValueString()
		}
		if !item.Subnet.IsNull() {
			list.Items[i].Subnet = item.Subnet.ValueString()
		}
		if !item.Start.IsNull() {
			list.Items[i].StartIP = item.Start.ValueString()
		}
		if !item.End.IsNull() {
			list.Items[i].EndIP = item.End.ValueString()
		}
		if !item.PortNumber.IsNull() {
			v := int(item.PortNumber.ValueInt64())
			list.Items[i].PortNumber = &v
		}
		if !item.StartPort.IsNull() {
			v := int(item.StartPort.ValueInt64())
			list.Items[i].StartPort = &v
		}
		if !item.EndPort.IsNull() {
			v := int(item.EndPort.ValueInt64())
			list.Items[i].EndPort = &v
		}
	}

	return list, diags
}

// ============================================================================
// DNS Policy helpers
// ============================================================================

// dnsPolicyAPIToModel maps a client.DnsPolicy to the Terraform resource model.
func dnsPolicyAPIToModel(ctx context.Context, result *client.DnsPolicy, model *DnsPolicyResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.Type = types.StringValue(result.Type)
	model.Enabled = types.BoolValue(result.Enabled)
	model.Name = types.StringValue(result.Name)

	if result.Domain != "" {
		model.Domain = types.StringValue(result.Domain)
	} else {
		model.Domain = types.StringNull()
	}

	if result.IPv4Address != "" {
		model.IPv4Address = types.StringValue(result.IPv4Address)
	} else {
		model.IPv4Address = types.StringNull()
	}

	if result.IPv6Address != "" {
		model.IPv6Address = types.StringValue(result.IPv6Address)
	} else {
		model.IPv6Address = types.StringNull()
	}

	if result.Target != "" {
		model.Target = types.StringValue(result.Target)
	} else {
		model.Target = types.StringNull()
	}

	if result.TTLSeconds != nil {
		model.TTLSeconds = types.Int64Value(int64(*result.TTLSeconds))
	} else {
		model.TTLSeconds = types.Int64Null()
	}

	if result.Priority != nil {
		model.Priority = types.Int64Value(int64(*result.Priority))
	} else {
		model.Priority = types.Int64Null()
	}

	if result.Weight != nil {
		model.Weight = types.Int64Value(int64(*result.Weight))
	} else {
		model.Weight = types.Int64Null()
	}

	if result.Port != nil {
		model.Port = types.Int64Value(int64(*result.Port))
	} else {
		model.Port = types.Int64Null()
	}

	if result.TxtValue != "" {
		model.TxtValue = types.StringValue(result.TxtValue)
	} else {
		model.TxtValue = types.StringNull()
	}

	if len(result.ForwardTo) > 0 {
		ft, d := types.ListValueFrom(ctx, types.StringType, result.ForwardTo)
		diags.Append(d...)
		model.ForwardTo = ft
	} else {
		model.ForwardTo = types.ListNull(types.StringType)
	}

	return diags
}

// dnsPolicyModelToAPI converts a DnsPolicyResourceModel into a client.DnsPolicy.
func dnsPolicyModelToAPI(ctx context.Context, model DnsPolicyResourceModel) (*client.DnsPolicy, diag.Diagnostics) {
	var diags diag.Diagnostics

	policy := &client.DnsPolicy{
		Type:    model.Type.ValueString(),
		Enabled: model.Enabled.ValueBool(),
		Name:    model.Name.ValueString(),
	}

	if !model.Domain.IsNull() {
		policy.Domain = model.Domain.ValueString()
	}
	if !model.IPv4Address.IsNull() {
		policy.IPv4Address = model.IPv4Address.ValueString()
	}
	if !model.IPv6Address.IsNull() {
		policy.IPv6Address = model.IPv6Address.ValueString()
	}
	if !model.Target.IsNull() {
		policy.Target = model.Target.ValueString()
	}
	if !model.TTLSeconds.IsNull() {
		v := int(model.TTLSeconds.ValueInt64())
		policy.TTLSeconds = &v
	}
	if !model.Priority.IsNull() {
		v := int(model.Priority.ValueInt64())
		policy.Priority = &v
	}
	if !model.Weight.IsNull() {
		v := int(model.Weight.ValueInt64())
		policy.Weight = &v
	}
	if !model.Port.IsNull() {
		v := int(model.Port.ValueInt64())
		policy.Port = &v
	}
	if !model.TxtValue.IsNull() {
		policy.TxtValue = model.TxtValue.ValueString()
	}
	if !model.ForwardTo.IsNull() {
		var servers []string
		diags.Append(model.ForwardTo.ElementsAs(ctx, &servers, false)...)
		policy.ForwardTo = servers
	}

	return policy, diags
}

// ============================================================================
// Hotspot Voucher helpers
// ============================================================================

// hotspotVoucherAPIToModel maps a client.HotspotVoucher to the Terraform resource model.
func hotspotVoucherAPIToModel(result *client.HotspotVoucher, model *HotspotVoucherResourceModel) {
	model.Name = types.StringValue(result.Name)
	model.TimeLimitMinutes = types.Int64Value(int64(result.TimeLimitMinutes))

	if result.Code != "" {
		model.Code = types.StringValue(result.Code)
	} else {
		model.Code = types.StringNull()
	}

	if result.AuthorizedGuestLimit != nil {
		model.AuthorizedGuestLimit = types.Int64Value(int64(*result.AuthorizedGuestLimit))
	} else {
		model.AuthorizedGuestLimit = types.Int64Null()
	}

	if result.DataUsageLimitMBytes != nil {
		model.DataUsageLimitMBytes = types.Int64Value(int64(*result.DataUsageLimitMBytes))
	} else {
		model.DataUsageLimitMBytes = types.Int64Null()
	}

	if result.RxRateLimitKbps != nil {
		model.RxRateLimitKbps = types.Int64Value(int64(*result.RxRateLimitKbps))
	} else {
		model.RxRateLimitKbps = types.Int64Null()
	}

	if result.TxRateLimitKbps != nil {
		model.TxRateLimitKbps = types.Int64Value(int64(*result.TxRateLimitKbps))
	} else {
		model.TxRateLimitKbps = types.Int64Null()
	}
}
