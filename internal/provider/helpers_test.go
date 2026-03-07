package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rslarson/terraform-provider-unifi/internal/client"
)

func TestParseCompositeID(t *testing.T) {
	tests := []struct {
		input        string
		wantSite     string
		wantResource string
		wantErr      bool
	}{
		{"site-123/resource-456", "site-123", "resource-456", false},
		{"abc/def", "abc", "def", false},
		{"site/with/slash", "site", "with/slash", false},
		{"noslash", "", "", true},
		{"/missingsite", "", "", true},
		{"missingresource/", "", "", true},
		{"", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			siteID, resourceID, err := parseCompositeID(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for input %q", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if siteID != tt.wantSite {
				t.Errorf("expected siteID %q, got %q", tt.wantSite, siteID)
			}
			if resourceID != tt.wantResource {
				t.Errorf("expected resourceID %q, got %q", tt.wantResource, resourceID)
			}
		})
	}
}

func TestExtractClient(t *testing.T) {
	t.Run("nil provider data", func(t *testing.T) {
		c, diags := extractClient(nil, "Resource")
		if c != nil {
			t.Error("expected nil client for nil provider data")
		}
		if diags.HasError() {
			t.Errorf("unexpected errors: %v", diags)
		}
	})

	t.Run("correct type", func(t *testing.T) {
		expected := client.NewClient("key", "host")
		c, diags := extractClient(expected, "Resource")
		if diags.HasError() {
			t.Fatalf("unexpected errors: %v", diags)
		}
		if c != expected {
			t.Error("expected returned client to match input")
		}
	})

	t.Run("wrong type", func(t *testing.T) {
		c, diags := extractClient("not-a-client", "Data Source")
		if c != nil {
			t.Error("expected nil client for wrong type")
		}
		if !diags.HasError() {
			t.Fatal("expected error for wrong type")
		}
		found := false
		for _, d := range diags.Errors() {
			if d.Summary() == "Unexpected Data Source Configure Type" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected 'Unexpected Data Source Configure Type' error, got: %v", diags)
		}
	})
}

func TestDeviceAPIToModel(t *testing.T) {
	t.Run("with uplink", func(t *testing.T) {
		device := &client.Device{
			MacAddress:      "aa:bb:cc:dd:ee:ff",
			IPAddress:       "192.168.1.1",
			Name:            "Switch",
			Model:           "USW-24",
			Supported:       true,
			State:           "ONLINE",
			FirmwareVersion: "7.0.0",
			FirmwareUpdatable: true,
			AdoptedAt:       "2024-01-01T00:00:00Z",
			ProvisionedAt:   "2024-01-02T00:00:00Z",
			ConfigurationID: "config-123",
			Uplink:          &client.DeviceUplink{DeviceID: "dev-parent"},
		}

		model := &DeviceDataSourceModel{}
		deviceAPIToModel(model, device)

		if model.MacAddress.ValueString() != "aa:bb:cc:dd:ee:ff" {
			t.Errorf("expected mac 'aa:bb:cc:dd:ee:ff', got %q", model.MacAddress.ValueString())
		}
		if model.IPAddress.ValueString() != "192.168.1.1" {
			t.Errorf("expected ip '192.168.1.1', got %q", model.IPAddress.ValueString())
		}
		if model.Name.ValueString() != "Switch" {
			t.Errorf("expected name 'Switch', got %q", model.Name.ValueString())
		}
		if model.Model.ValueString() != "USW-24" {
			t.Errorf("expected model 'USW-24', got %q", model.Model.ValueString())
		}
		if !model.Supported.ValueBool() {
			t.Error("expected supported to be true")
		}
		if model.State.ValueString() != "ONLINE" {
			t.Errorf("expected state 'ONLINE', got %q", model.State.ValueString())
		}
		if model.FirmwareVersion.ValueString() != "7.0.0" {
			t.Errorf("expected firmware '7.0.0', got %q", model.FirmwareVersion.ValueString())
		}
		if !model.FirmwareUpdatable.ValueBool() {
			t.Error("expected firmware updatable to be true")
		}
		if model.AdoptedAt.ValueString() != "2024-01-01T00:00:00Z" {
			t.Errorf("expected adopted at '2024-01-01T00:00:00Z', got %q", model.AdoptedAt.ValueString())
		}
		if model.ProvisionedAt.ValueString() != "2024-01-02T00:00:00Z" {
			t.Errorf("expected provisioned at '2024-01-02T00:00:00Z', got %q", model.ProvisionedAt.ValueString())
		}
		if model.ConfigurationID.ValueString() != "config-123" {
			t.Errorf("expected config ID 'config-123', got %q", model.ConfigurationID.ValueString())
		}
		if model.UplinkDeviceID.ValueString() != "dev-parent" {
			t.Errorf("expected uplink device ID 'dev-parent', got %q", model.UplinkDeviceID.ValueString())
		}
	})

	t.Run("without uplink", func(t *testing.T) {
		device := &client.Device{
			MacAddress: "aa:bb:cc:dd:ee:ff",
			Name:       "Switch",
			Uplink:     nil,
		}

		model := &DeviceDataSourceModel{}
		deviceAPIToModel(model, device)

		if !model.UplinkDeviceID.IsNull() {
			t.Error("expected uplink device ID to be null when uplink is nil")
		}
	})
}

func TestNetworkAPIToModel(t *testing.T) {
	ctx := context.Background()

	t.Run("with DHCP guarding", func(t *testing.T) {
		network := &client.Network{
			Name:       "Test Net",
			Management: client.ManagementGateway,
			Enabled:    true,
			VlanID:     100,
			DhcpGuarding: &client.DhcpGuarding{
				TrustedDhcpServerIPAddresses: []string{"10.0.0.1", "10.0.0.2"},
			},
		}

		name, management, enabled, vlanID, dhcpIPs, diags := networkAPIToModel(ctx, network)
		if diags.HasError() {
			t.Fatalf("unexpected errors: %v", diags)
		}
		if name.ValueString() != "Test Net" {
			t.Errorf("expected name 'Test Net', got %q", name.ValueString())
		}
		if management.ValueString() != client.ManagementGateway {
			t.Errorf("expected management %q, got %q", client.ManagementGateway, management.ValueString())
		}
		if !enabled.ValueBool() {
			t.Error("expected enabled to be true")
		}
		if vlanID.ValueInt64() != 100 {
			t.Errorf("expected vlanID 100, got %d", vlanID.ValueInt64())
		}
		if dhcpIPs.IsNull() {
			t.Error("expected dhcpIPs to not be null")
		}
		var ips []string
		diags = dhcpIPs.ElementsAs(ctx, &ips, false)
		if diags.HasError() {
			t.Fatalf("unexpected errors: %v", diags)
		}
		if len(ips) != 2 || ips[0] != "10.0.0.1" {
			t.Errorf("expected 2 IPs starting with '10.0.0.1', got %v", ips)
		}
	})

	t.Run("without DHCP guarding", func(t *testing.T) {
		network := &client.Network{
			Name:       "Simple Net",
			Management: client.ManagementUnmanaged,
			Enabled:    false,
			VlanID:     200,
		}

		_, _, _, _, dhcpIPs, diags := networkAPIToModel(ctx, network)
		if diags.HasError() {
			t.Fatalf("unexpected errors: %v", diags)
		}
		if !dhcpIPs.IsNull() {
			t.Error("expected dhcpIPs to be null when DHCP guarding is nil")
		}
	})

	t.Run("empty DHCP IP list", func(t *testing.T) {
		network := &client.Network{
			Name:       "Net",
			Management: client.ManagementUnmanaged,
			DhcpGuarding: &client.DhcpGuarding{
				TrustedDhcpServerIPAddresses: []string{},
			},
		}

		_, _, _, _, dhcpIPs, diags := networkAPIToModel(ctx, network)
		if diags.HasError() {
			t.Fatalf("unexpected errors: %v", diags)
		}
		if !dhcpIPs.IsNull() {
			t.Error("expected dhcpIPs to be null when DHCP IP list is empty")
		}
	})
}

func TestNetworkModelToAPI(t *testing.T) {
	ctx := context.Background()

	t.Run("with DHCP IPs", func(t *testing.T) {
		ips, diags := types.ListValueFrom(ctx, types.StringType, []string{"10.0.0.1"})
		if diags.HasError() {
			t.Fatalf("unexpected errors: %v", diags)
		}

		model := NetworkResourceModel{
			Name:                         types.StringValue("My Net"),
			Management:                   types.StringValue(client.ManagementSwitch),
			Enabled:                      types.BoolValue(true),
			VlanID:                        types.Int64Value(300),
			TrustedDhcpServerIPAddresses: ips,
		}

		network, d := networkModelToAPI(ctx, model)
		if d.HasError() {
			t.Fatalf("unexpected errors: %v", d)
		}
		if network.Name != "My Net" {
			t.Errorf("expected name 'My Net', got %q", network.Name)
		}
		if network.Management != client.ManagementSwitch {
			t.Errorf("expected management %q, got %q", client.ManagementSwitch, network.Management)
		}
		if !network.Enabled {
			t.Error("expected enabled to be true")
		}
		if network.VlanID != 300 {
			t.Errorf("expected vlanID 300, got %d", network.VlanID)
		}
		if network.DhcpGuarding == nil {
			t.Fatal("expected DhcpGuarding to be set")
		}
		if len(network.DhcpGuarding.TrustedDhcpServerIPAddresses) != 1 {
			t.Errorf("expected 1 DHCP IP, got %d", len(network.DhcpGuarding.TrustedDhcpServerIPAddresses))
		}
	})

	t.Run("without DHCP IPs", func(t *testing.T) {
		model := NetworkResourceModel{
			Name:                         types.StringValue("Simple Net"),
			Management:                   types.StringValue(client.ManagementUnmanaged),
			Enabled:                      types.BoolValue(false),
			VlanID:                        types.Int64Value(100),
			TrustedDhcpServerIPAddresses: types.ListNull(types.StringType),
		}

		network, d := networkModelToAPI(ctx, model)
		if d.HasError() {
			t.Fatalf("unexpected errors: %v", d)
		}
		if network.DhcpGuarding != nil {
			t.Error("expected DhcpGuarding to be nil when IPs are null")
		}
	})
}

func TestWifiBroadcastAPIToModel(t *testing.T) {
	t.Run("full broadcast", func(t *testing.T) {
		broadcast := &client.WifiBroadcast{
			Type:                                client.BroadcastTypeStandard,
			Name:                                "Test WiFi",
			Enabled:                             true,
			ClientIsolationEnabled:              true,
			HideName:                            false,
			MulticastToUnicastConversionEnabled: true,
			UapsdEnabled:                        false,
			SecurityConfiguration:               &client.SecurityConfiguration{Type: client.SecurityWPA2Personal},
			Network:                             &client.BroadcastNetwork{Type: client.NetworkTypeSpecific, NetworkID: "net-123"},
		}

		broadcastType, name, enabled, clientIsolation, hideName, mcastUcast, uapsd, securityType, networkType, networkID := wifiBroadcastAPIToModel(broadcast)

		if broadcastType.ValueString() != client.BroadcastTypeStandard {
			t.Errorf("expected type %q, got %q", client.BroadcastTypeStandard, broadcastType.ValueString())
		}
		if name.ValueString() != "Test WiFi" {
			t.Errorf("expected name 'Test WiFi', got %q", name.ValueString())
		}
		if !enabled.ValueBool() {
			t.Error("expected enabled to be true")
		}
		if !clientIsolation.ValueBool() {
			t.Error("expected client isolation to be true")
		}
		if hideName.ValueBool() {
			t.Error("expected hide name to be false")
		}
		if !mcastUcast.ValueBool() {
			t.Error("expected multicast to unicast to be true")
		}
		if uapsd.ValueBool() {
			t.Error("expected uapsd to be false")
		}
		if securityType.ValueString() != client.SecurityWPA2Personal {
			t.Errorf("expected security type %q, got %q", client.SecurityWPA2Personal, securityType.ValueString())
		}
		if networkType.ValueString() != client.NetworkTypeSpecific {
			t.Errorf("expected network type %q, got %q", client.NetworkTypeSpecific, networkType.ValueString())
		}
		if networkID.ValueString() != "net-123" {
			t.Errorf("expected network ID 'net-123', got %q", networkID.ValueString())
		}
	})

	t.Run("nil security and network", func(t *testing.T) {
		broadcast := &client.WifiBroadcast{
			Name:                  "Open WiFi",
			SecurityConfiguration: nil,
			Network:               nil,
		}

		_, _, _, _, _, _, _, securityType, networkType, networkID := wifiBroadcastAPIToModel(broadcast)

		if securityType.ValueString() != "" {
			t.Errorf("expected empty security type for nil config, got %q", securityType.ValueString())
		}
		if networkType.ValueString() != "" {
			t.Errorf("expected empty network type for nil network, got %q", networkType.ValueString())
		}
		if networkID.ValueString() != "" {
			t.Errorf("expected empty network ID for nil network, got %q", networkID.ValueString())
		}
	})

	t.Run("network with empty network ID", func(t *testing.T) {
		broadcast := &client.WifiBroadcast{
			Name:    "Native WiFi",
			Network: &client.BroadcastNetwork{Type: client.NetworkTypeNative, NetworkID: ""},
		}

		_, _, _, _, _, _, _, _, networkType, networkID := wifiBroadcastAPIToModel(broadcast)

		if networkType.ValueString() != client.NetworkTypeNative {
			t.Errorf("expected network type %q, got %q", client.NetworkTypeNative, networkType.ValueString())
		}
		if !networkID.IsNull() {
			t.Error("expected network ID to be null for empty network ID")
		}
	})
}

func TestWifiBroadcastModelToAPI(t *testing.T) {
	t.Run("with passphrase_wo", func(t *testing.T) {
		plan := WifiBroadcastResourceModel{
			Type:                                types.StringValue(client.BroadcastTypeStandard),
			Name:                                types.StringValue("My WiFi"),
			Enabled:                             types.BoolValue(true),
			SecurityType:                        types.StringValue(client.SecurityWPA2Personal),
			Passphrase:                          types.StringValue("legacy-pass"),
			NetworkType:                         types.StringValue(client.NetworkTypeNative),
			NetworkID:                           types.StringNull(),
			ClientIsolationEnabled:              types.BoolValue(false),
			HideName:                            types.BoolValue(false),
			MulticastToUnicastConversionEnabled: types.BoolValue(false),
			UapsdEnabled:                        types.BoolValue(false),
		}

		passphraseWO := types.StringValue("write-only-pass")
		broadcast := wifiBroadcastModelToAPI(plan, passphraseWO)

		if broadcast.SecurityConfiguration.Passphrase != "write-only-pass" {
			t.Errorf("expected passphrase 'write-only-pass', got %q", broadcast.SecurityConfiguration.Passphrase)
		}
	})

	t.Run("with legacy passphrase", func(t *testing.T) {
		plan := WifiBroadcastResourceModel{
			Type:                                types.StringValue(client.BroadcastTypeStandard),
			Name:                                types.StringValue("My WiFi"),
			Enabled:                             types.BoolValue(true),
			SecurityType:                        types.StringValue(client.SecurityWPA2Personal),
			Passphrase:                          types.StringValue("legacy-pass"),
			NetworkType:                         types.StringValue(client.NetworkTypeNative),
			NetworkID:                           types.StringNull(),
			ClientIsolationEnabled:              types.BoolValue(false),
			HideName:                            types.BoolValue(false),
			MulticastToUnicastConversionEnabled: types.BoolValue(false),
			UapsdEnabled:                        types.BoolValue(false),
		}

		broadcast := wifiBroadcastModelToAPI(plan, types.StringNull())

		if broadcast.SecurityConfiguration.Passphrase != "legacy-pass" {
			t.Errorf("expected passphrase 'legacy-pass', got %q", broadcast.SecurityConfiguration.Passphrase)
		}
	})

	t.Run("no passphrase (open)", func(t *testing.T) {
		plan := WifiBroadcastResourceModel{
			Type:                                types.StringValue(client.BroadcastTypeStandard),
			Name:                                types.StringValue("Open WiFi"),
			Enabled:                             types.BoolValue(true),
			SecurityType:                        types.StringValue(client.SecurityOpen),
			Passphrase:                          types.StringNull(),
			NetworkType:                         types.StringValue(client.NetworkTypeNative),
			NetworkID:                           types.StringNull(),
			ClientIsolationEnabled:              types.BoolValue(false),
			HideName:                            types.BoolValue(false),
			MulticastToUnicastConversionEnabled: types.BoolValue(false),
			UapsdEnabled:                        types.BoolValue(false),
		}

		broadcast := wifiBroadcastModelToAPI(plan, types.StringNull())

		if broadcast.SecurityConfiguration.Passphrase != "" {
			t.Errorf("expected empty passphrase, got %q", broadcast.SecurityConfiguration.Passphrase)
		}
	})

	t.Run("with specific network ID", func(t *testing.T) {
		plan := WifiBroadcastResourceModel{
			Type:                                types.StringValue(client.BroadcastTypeStandard),
			Name:                                types.StringValue("WiFi"),
			Enabled:                             types.BoolValue(true),
			SecurityType:                        types.StringValue(client.SecurityOpen),
			Passphrase:                          types.StringNull(),
			NetworkType:                         types.StringValue(client.NetworkTypeSpecific),
			NetworkID:                           types.StringValue("net-456"),
			ClientIsolationEnabled:              types.BoolValue(false),
			HideName:                            types.BoolValue(false),
			MulticastToUnicastConversionEnabled: types.BoolValue(false),
			UapsdEnabled:                        types.BoolValue(false),
		}

		broadcast := wifiBroadcastModelToAPI(plan, types.StringNull())

		if broadcast.Network.Type != client.NetworkTypeSpecific {
			t.Errorf("expected network type %q, got %q", client.NetworkTypeSpecific, broadcast.Network.Type)
		}
		if broadcast.Network.NetworkID != "net-456" {
			t.Errorf("expected network ID 'net-456', got %q", broadcast.Network.NetworkID)
		}
	})
}

func TestFirewallZoneAPIToModel(t *testing.T) {
	ctx := context.Background()

	zone := &client.FirewallZone{
		Name:       "Test Zone",
		NetworkIDs: []string{"net-1", "net-2", "net-3"},
	}

	name, networkIDs, diags := firewallZoneAPIToModel(ctx, zone)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}
	if name.ValueString() != "Test Zone" {
		t.Errorf("expected name 'Test Zone', got %q", name.ValueString())
	}

	var ids []string
	diags = networkIDs.ElementsAs(ctx, &ids, false)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}
	if len(ids) != 3 {
		t.Errorf("expected 3 network IDs, got %d", len(ids))
	}
}

func TestFirewallZoneModelToAPI(t *testing.T) {
	ctx := context.Background()

	networkIDs, diags := types.ListValueFrom(ctx, types.StringType, []string{"net-a", "net-b"})
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}

	model := FirewallZoneResourceModel{
		Name:       types.StringValue("My Zone"),
		NetworkIDs: networkIDs,
	}

	zone, d := firewallZoneModelToAPI(ctx, model)
	if d.HasError() {
		t.Fatalf("unexpected errors: %v", d)
	}
	if zone.Name != "My Zone" {
		t.Errorf("expected name 'My Zone', got %q", zone.Name)
	}
	if len(zone.NetworkIDs) != 2 {
		t.Errorf("expected 2 network IDs, got %d", len(zone.NetworkIDs))
	}
	if zone.NetworkIDs[0] != "net-a" {
		t.Errorf("expected first ID 'net-a', got %q", zone.NetworkIDs[0])
	}
}
