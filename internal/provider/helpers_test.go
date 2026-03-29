package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rslarson/terraform-provider-unifi/internal/client"
)

// --- parseCompositeID ---

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

// --- extractClient ---

func TestExtractClientSuccess(t *testing.T) {
	c := client.NewClient("key", "host")
	result, diags := extractClient(c, "Resource")
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}
	if result != c {
		t.Error("expected returned client to be the same as provided")
	}
}

func TestExtractClientNil(t *testing.T) {
	result, diags := extractClient(nil, "Resource")
	if diags.HasError() {
		t.Fatalf("unexpected errors for nil provider data: %v", diags)
	}
	if result != nil {
		t.Error("expected nil client for nil provider data")
	}
}

func TestExtractClientWrongType(t *testing.T) {
	_, diags := extractClient("not-a-client", "Resource")
	if !diags.HasError() {
		t.Error("expected error for wrong provider data type")
	}
}

// --- networkAPIToModelFull ---

func TestNetworkAPIToModelBasic(t *testing.T) {
	n := &client.Network{
		ID:         "net-1",
		Name:       "My Network",
		Management: client.ManagementGateway,
		Enabled:    true,
		VlanID:     100,
	}
	model := &NetworkResourceModel{}
	diags := networkAPIToModelFull(context.Background(), n, model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}
	if model.Name.ValueString() != "My Network" {
		t.Errorf("expected name 'My Network', got %q", model.Name.ValueString())
	}
	if model.Management.ValueString() != client.ManagementGateway {
		t.Errorf("expected management %q, got %q", client.ManagementGateway, model.Management.ValueString())
	}
	if !model.Enabled.ValueBool() {
		t.Error("expected enabled to be true")
	}
	if model.VlanID.ValueInt64() != 100 {
		t.Errorf("expected vlanId 100, got %d", model.VlanID.ValueInt64())
	}
	if !model.TrustedDhcpServerIPAddresses.IsNull() {
		t.Error("expected dhcpIPs to be null when no dhcpGuarding")
	}
}

func TestNetworkAPIToModelWithDhcpGuarding(t *testing.T) {
	n := &client.Network{
		Name:       "Guarded",
		Management: client.ManagementSwitch,
		VlanID:     200,
		DhcpGuarding: &client.DhcpGuarding{
			TrustedDhcpServerIPAddresses: []string{"192.168.1.1", "10.0.0.1"},
		},
	}
	model := &NetworkResourceModel{}
	diags := networkAPIToModelFull(context.Background(), n, model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}
	if model.TrustedDhcpServerIPAddresses.IsNull() {
		t.Fatal("expected dhcpIPs to be non-null")
	}
	var ips []string
	diags = model.TrustedDhcpServerIPAddresses.ElementsAs(context.Background(), &ips, false)
	if diags.HasError() {
		t.Fatalf("ElementsAs error: %v", diags)
	}
	if len(ips) != 2 {
		t.Errorf("expected 2 IPs, got %d", len(ips))
	}
	if ips[0] != "192.168.1.1" {
		t.Errorf("expected first IP '192.168.1.1', got %q", ips[0])
	}
}

func TestNetworkAPIToModelEmptyDhcpGuarding(t *testing.T) {
	n := &client.Network{
		Name:       "Empty Guard",
		Management: client.ManagementUnmanaged,
		VlanID:     10,
		DhcpGuarding: &client.DhcpGuarding{
			TrustedDhcpServerIPAddresses: []string{},
		},
	}
	model := &NetworkResourceModel{}
	diags := networkAPIToModelFull(context.Background(), n, model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}
	if !model.TrustedDhcpServerIPAddresses.IsNull() {
		t.Error("expected dhcpIPs to be null for empty trusted IPs")
	}
}

// --- networkModelToAPI ---

func TestNetworkModelToAPIBasic(t *testing.T) {
	model := NetworkResourceModel{
		Name:                         types.StringValue("Test Net"),
		Management:                   types.StringValue(client.ManagementUnmanaged),
		Enabled:                      types.BoolValue(true),
		VlanID:                       types.Int64Value(50),
		TrustedDhcpServerIPAddresses: types.ListNull(types.StringType),
		IsolationEnabled:             types.BoolNull(),
		CellularBackupEnabled:        types.BoolNull(),
		InternetAccessEnabled:        types.BoolNull(),
		MdnsForwardingEnabled:        types.BoolNull(),
		ZoneID:                       types.StringNull(),
		DeviceID:                     types.StringNull(),
		IPv4Configuration:            types.ObjectNull(ipv4ConfigurationAttrTypes()),
	}
	network, diags := networkModelToAPI(context.Background(), model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}
	if network.Name != "Test Net" {
		t.Errorf("expected name 'Test Net', got %q", network.Name)
	}
	if network.Management != client.ManagementUnmanaged {
		t.Errorf("expected management %q, got %q", client.ManagementUnmanaged, network.Management)
	}
	if !network.Enabled {
		t.Error("expected enabled to be true")
	}
	if network.VlanID != 50 {
		t.Errorf("expected vlanId 50, got %d", network.VlanID)
	}
	if network.DhcpGuarding != nil {
		t.Error("expected DhcpGuarding to be nil when no trusted IPs")
	}
}

func TestNetworkModelToAPIWithTrustedIPs(t *testing.T) {
	ips, diags := types.ListValueFrom(context.Background(), types.StringType, []string{"10.1.1.1", "10.2.2.2"})
	if diags.HasError() {
		t.Fatalf("ListValueFrom error: %v", diags)
	}
	model := NetworkResourceModel{
		Name:                         types.StringValue("Net"),
		Management:                   types.StringValue(client.ManagementGateway),
		Enabled:                      types.BoolValue(false),
		VlanID:                       types.Int64Value(300),
		TrustedDhcpServerIPAddresses: ips,
		IsolationEnabled:             types.BoolNull(),
		CellularBackupEnabled:        types.BoolNull(),
		InternetAccessEnabled:        types.BoolNull(),
		MdnsForwardingEnabled:        types.BoolNull(),
		ZoneID:                       types.StringNull(),
		DeviceID:                     types.StringNull(),
		IPv4Configuration:            types.ObjectNull(ipv4ConfigurationAttrTypes()),
	}
	network, diags := networkModelToAPI(context.Background(), model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}
	if network.DhcpGuarding == nil {
		t.Fatal("expected DhcpGuarding to be non-nil")
	}
	if len(network.DhcpGuarding.TrustedDhcpServerIPAddresses) != 2 {
		t.Errorf("expected 2 trusted IPs, got %d", len(network.DhcpGuarding.TrustedDhcpServerIPAddresses))
	}
	if network.DhcpGuarding.TrustedDhcpServerIPAddresses[0] != "10.1.1.1" {
		t.Errorf("expected first IP '10.1.1.1', got %q", network.DhcpGuarding.TrustedDhcpServerIPAddresses[0])
	}
}

// --- wifiBroadcastAPIToModel ---

func TestWifiBroadcastAPIToModelComplete(t *testing.T) {
	wb := &client.WifiBroadcast{
		ID:      "wifi-1",
		Type:    client.BroadcastTypeStandard,
		Name:    "HomeSSID",
		Enabled: true,
		SecurityConfiguration: &client.SecurityConfiguration{
			Type: client.SecurityWPA2Personal,
		},
		Network: &client.BroadcastNetwork{
			Type:      client.NetworkTypeSpecific,
			NetworkID: "net-abc",
		},
		ClientIsolationEnabled:              true,
		HideName:                            true,
		MulticastToUnicastConversionEnabled: true,
		UapsdEnabled:                        true,
	}
	model := &WifiBroadcastResourceModel{}
	diags := wifiBroadcastAPIToModel(context.Background(), model, wb)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}

	if model.Type.ValueString() != client.BroadcastTypeStandard {
		t.Errorf("expected type %q, got %q", client.BroadcastTypeStandard, model.Type.ValueString())
	}
	if model.Name.ValueString() != "HomeSSID" {
		t.Errorf("expected name 'HomeSSID', got %q", model.Name.ValueString())
	}
	if !model.Enabled.ValueBool() {
		t.Error("expected enabled to be true")
	}
	if !model.ClientIsolationEnabled.ValueBool() {
		t.Error("expected clientIsolationEnabled to be true")
	}
	if !model.HideName.ValueBool() {
		t.Error("expected hideName to be true")
	}
	if !model.MulticastToUnicastConversionEnabled.ValueBool() {
		t.Error("expected multicastToUnicastConversionEnabled to be true")
	}
	if !model.UapsdEnabled.ValueBool() {
		t.Error("expected uapsdEnabled to be true")
	}
	if model.SecurityType.ValueString() != client.SecurityWPA2Personal {
		t.Errorf("expected securityType %q, got %q", client.SecurityWPA2Personal, model.SecurityType.ValueString())
	}
	if model.NetworkType.ValueString() != client.NetworkTypeSpecific {
		t.Errorf("expected networkType %q, got %q", client.NetworkTypeSpecific, model.NetworkType.ValueString())
	}
	if model.NetworkID.ValueString() != "net-abc" {
		t.Errorf("expected networkId 'net-abc', got %q", model.NetworkID.ValueString())
	}
}

func TestWifiBroadcastAPIToModelNativeNetwork(t *testing.T) {
	wb := &client.WifiBroadcast{
		Type:    client.BroadcastTypeIoTOptimized,
		Name:    "IoT",
		Enabled: false,
		SecurityConfiguration: &client.SecurityConfiguration{Type: client.SecurityOpen},
		Network:               &client.BroadcastNetwork{Type: client.NetworkTypeNative},
	}
	model := &WifiBroadcastResourceModel{}
	diags := wifiBroadcastAPIToModel(context.Background(), model, wb)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}

	if model.NetworkType.ValueString() != client.NetworkTypeNative {
		t.Errorf("expected networkType %q, got %q", client.NetworkTypeNative, model.NetworkType.ValueString())
	}
	// networkID should be null for NATIVE network (no networkId)
	if !model.NetworkID.IsNull() {
		t.Errorf("expected null networkId for NATIVE type, got %q", model.NetworkID.ValueString())
	}
}

func TestWifiBroadcastAPIToModelNilSecurityConfig(t *testing.T) {
	wb := &client.WifiBroadcast{
		Type:    client.BroadcastTypeStandard,
		Name:    "Test",
		Enabled: true,
		Network: &client.BroadcastNetwork{Type: client.NetworkTypeNative},
		// SecurityConfiguration intentionally nil (defensive test)
	}
	model := &WifiBroadcastResourceModel{}
	diags := wifiBroadcastAPIToModel(context.Background(), model, wb)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}
	// Should not panic; securityType will be zero-value (empty string)
	_ = model.SecurityType
}

// --- wifiBroadcastModelToAPI ---

func TestWifiBroadcastModelToAPIWithPassphrase(t *testing.T) {
	plan := WifiBroadcastResourceModel{
		Type:                                types.StringValue(client.BroadcastTypeStandard),
		Name:                                types.StringValue("WiFi"),
		Enabled:                             types.BoolValue(true),
		SecurityType:                        types.StringValue(client.SecurityWPA2Personal),
		Passphrase:                          types.StringValue("mypassword12"),
		PassphraseWO:                        types.StringNull(),
		NetworkType:                         types.StringValue(client.NetworkTypeNative),
		NetworkID:                           types.StringNull(),
		ClientIsolationEnabled:              types.BoolValue(false),
		HideName:                            types.BoolValue(false),
		MulticastToUnicastConversionEnabled: types.BoolValue(false),
		UapsdEnabled:                        types.BoolValue(false),
		BasicDataRateGHz24:                  types.Int64Null(),
		BasicDataRateGHz5:                   types.Int64Null(),
		ClientFilterAction:                  types.StringNull(),
		ClientFilterMacAddresses:            types.ListNull(types.StringType),
		BlackoutScheduleDays:                types.ListNull(types.ObjectType{AttrTypes: blackoutScheduleDayAttrTypes()}),
		BroadcastingFrequenciesGHz:          types.ListNull(types.Float64Type),
		BroadcastingDeviceFilterType:        types.StringNull(),
		BroadcastingDeviceFilterIds:         types.ListNull(types.StringType),
		MulticastFilterAction:               types.StringNull(),
		MdnsProxyMode:                       types.StringNull(),
		BandSteeringEnabled:                 types.BoolNull(),
		MloEnabled:                          types.BoolNull(),
		ArpProxyEnabled:                     types.BoolNull(),
		BssTransitionEnabled:                types.BoolNull(),
		AdvertiseDeviceName:                 types.BoolNull(),
	}
	wb, diags := wifiBroadcastModelToAPI(context.Background(), plan, types.StringNull())
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}
	if wb.SecurityConfiguration.Passphrase != "mypassword12" {
		t.Errorf("expected passphrase 'mypassword12', got %q", wb.SecurityConfiguration.Passphrase)
	}
	if wb.Name != "WiFi" {
		t.Errorf("expected name 'WiFi', got %q", wb.Name)
	}
}

func TestWifiBroadcastModelToAPIPassphraseWOTakesPriority(t *testing.T) {
	plan := WifiBroadcastResourceModel{
		Type:         types.StringValue(client.BroadcastTypeStandard),
		Name:         types.StringValue("WiFi"),
		Enabled:      types.BoolValue(true),
		SecurityType: types.StringValue(client.SecurityWPA3Personal),
		Passphrase:   types.StringValue("legacy-pass"),
		PassphraseWO: types.StringNull(), // WO is null in state (write-only)
		NetworkType:  types.StringValue(client.NetworkTypeNative),
		NetworkID:    types.StringNull(),
	}
	// passphraseWO read from config takes priority over state passphrase
	wb, diags := wifiBroadcastModelToAPI(context.Background(), plan, types.StringValue("write-only-pass"))
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}
	if wb.SecurityConfiguration.Passphrase != "write-only-pass" {
		t.Errorf("expected write-only passphrase to take priority, got %q",
			wb.SecurityConfiguration.Passphrase)
	}
}

func TestWifiBroadcastModelToAPIOpenSecurity(t *testing.T) {
	plan := WifiBroadcastResourceModel{
		Type:         types.StringValue(client.BroadcastTypeStandard),
		Name:         types.StringValue("Open WiFi"),
		Enabled:      types.BoolValue(true),
		SecurityType: types.StringValue(client.SecurityOpen),
		Passphrase:   types.StringNull(),
		PassphraseWO: types.StringNull(),
		NetworkType:  types.StringValue(client.NetworkTypeNative),
		NetworkID:    types.StringNull(),
	}
	wb, diags := wifiBroadcastModelToAPI(context.Background(), plan, types.StringNull())
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}
	if wb.SecurityConfiguration.Passphrase != "" {
		t.Errorf("expected empty passphrase for OPEN, got %q", wb.SecurityConfiguration.Passphrase)
	}
	if wb.SecurityConfiguration.Type != client.SecurityOpen {
		t.Errorf("expected security type %q, got %q", client.SecurityOpen, wb.SecurityConfiguration.Type)
	}
}

func TestWifiBroadcastModelToAPIWithNetworkID(t *testing.T) {
	plan := WifiBroadcastResourceModel{
		Type:         types.StringValue(client.BroadcastTypeStandard),
		Name:         types.StringValue("Vlan WiFi"),
		Enabled:      types.BoolValue(true),
		SecurityType: types.StringValue(client.SecurityOpen),
		Passphrase:   types.StringNull(),
		PassphraseWO: types.StringNull(),
		NetworkType:  types.StringValue(client.NetworkTypeSpecific),
		NetworkID:    types.StringValue("net-vlan42"),
	}
	wb, diags := wifiBroadcastModelToAPI(context.Background(), plan, types.StringNull())
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}
	if wb.Network.NetworkID != "net-vlan42" {
		t.Errorf("expected networkId 'net-vlan42', got %q", wb.Network.NetworkID)
	}
	if wb.Network.Type != client.NetworkTypeSpecific {
		t.Errorf("expected networkType %q, got %q", client.NetworkTypeSpecific, wb.Network.Type)
	}
}

func TestWifiBroadcastModelToAPIAllBoolFields(t *testing.T) {
	plan := WifiBroadcastResourceModel{
		Type:                                types.StringValue(client.BroadcastTypeIoTOptimized),
		Name:                                types.StringValue("IoT"),
		Enabled:                             types.BoolValue(true),
		SecurityType:                        types.StringValue(client.SecurityWPA2WPA3Personal),
		Passphrase:                          types.StringValue("iotpassword"),
		PassphraseWO:                        types.StringNull(),
		NetworkType:                         types.StringValue(client.NetworkTypeNative),
		NetworkID:                           types.StringNull(),
		ClientIsolationEnabled:              types.BoolValue(true),
		HideName:                            types.BoolValue(true),
		MulticastToUnicastConversionEnabled: types.BoolValue(true),
		UapsdEnabled:                        types.BoolValue(true),
		BasicDataRateGHz24:                  types.Int64Null(),
		BasicDataRateGHz5:                   types.Int64Null(),
		ClientFilterAction:                  types.StringNull(),
		ClientFilterMacAddresses:            types.ListNull(types.StringType),
		BlackoutScheduleDays:                types.ListNull(types.ObjectType{AttrTypes: blackoutScheduleDayAttrTypes()}),
		BroadcastingFrequenciesGHz:          types.ListNull(types.Float64Type),
		BroadcastingDeviceFilterType:        types.StringNull(),
		BroadcastingDeviceFilterIds:         types.ListNull(types.StringType),
		MulticastFilterAction:               types.StringNull(),
		MdnsProxyMode:                       types.StringNull(),
		BandSteeringEnabled:                 types.BoolNull(),
		MloEnabled:                          types.BoolNull(),
		ArpProxyEnabled:                     types.BoolNull(),
		BssTransitionEnabled:                types.BoolNull(),
		AdvertiseDeviceName:                 types.BoolNull(),
	}
	wb, diags := wifiBroadcastModelToAPI(context.Background(), plan, types.StringNull())
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}
	if !wb.ClientIsolationEnabled {
		t.Error("expected clientIsolationEnabled to be true")
	}
	if !wb.HideName {
		t.Error("expected hideName to be true")
	}
	if !wb.MulticastToUnicastConversionEnabled {
		t.Error("expected multicastToUnicastConversionEnabled to be true")
	}
	if !wb.UapsdEnabled {
		t.Error("expected uapsdEnabled to be true")
	}
}

// --- firewallZoneAPIToModel ---

func TestFirewallZoneAPIToModel(t *testing.T) {
	z := &client.FirewallZone{
		ID:         "zone-1",
		Name:       "LAN Zone",
		NetworkIDs: []string{"net-a", "net-b", "net-c"},
	}
	name, networkIDs, diags := firewallZoneAPIToModel(context.Background(), z)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}
	if name.ValueString() != "LAN Zone" {
		t.Errorf("expected name 'LAN Zone', got %q", name.ValueString())
	}
	var ids []string
	diags = networkIDs.ElementsAs(context.Background(), &ids, false)
	if diags.HasError() {
		t.Fatalf("ElementsAs error: %v", diags)
	}
	if len(ids) != 3 {
		t.Errorf("expected 3 network IDs, got %d", len(ids))
	}
	if ids[0] != "net-a" || ids[2] != "net-c" {
		t.Errorf("unexpected network IDs: %v", ids)
	}
}

func TestFirewallZoneAPIToModelEmptyNetworks(t *testing.T) {
	z := &client.FirewallZone{
		Name:       "Empty Zone",
		NetworkIDs: []string{},
	}
	name, networkIDs, diags := firewallZoneAPIToModel(context.Background(), z)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}
	if name.ValueString() != "Empty Zone" {
		t.Errorf("expected name 'Empty Zone', got %q", name.ValueString())
	}
	if networkIDs.IsNull() {
		t.Error("expected non-null list for empty networkIds")
	}
	var ids []string
	networkIDs.ElementsAs(context.Background(), &ids, false)
	if len(ids) != 0 {
		t.Errorf("expected 0 network IDs, got %d", len(ids))
	}
}

// --- firewallZoneModelToAPI ---

func TestFirewallZoneModelToAPI(t *testing.T) {
	ids, diags := types.ListValueFrom(context.Background(), types.StringType, []string{"net-1", "net-2"})
	if diags.HasError() {
		t.Fatalf("ListValueFrom error: %v", diags)
	}
	model := FirewallZoneResourceModel{
		Name:       types.StringValue("WAN Zone"),
		NetworkIDs: ids,
	}
	zone, diags := firewallZoneModelToAPI(context.Background(), model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}
	if zone.Name != "WAN Zone" {
		t.Errorf("expected name 'WAN Zone', got %q", zone.Name)
	}
	if len(zone.NetworkIDs) != 2 {
		t.Errorf("expected 2 network IDs, got %d", len(zone.NetworkIDs))
	}
	if zone.NetworkIDs[0] != "net-1" {
		t.Errorf("expected first ID 'net-1', got %q", zone.NetworkIDs[0])
	}
}

// --- deviceAPIToModel ---

func TestDeviceAPIToModelWithUplink(t *testing.T) {
	d := &client.Device{
		ID:                "dev-1",
		MacAddress:        "aa:bb:cc:dd:ee:ff",
		IPAddress:         "10.0.0.5",
		Name:              "My AP",
		Model:             "U6-LR",
		Supported:         true,
		State:             "ONLINE",
		FirmwareVersion:   "6.6.1",
		FirmwareUpdatable: true,
		AdoptedAt:         "2024-01-01T00:00:00Z",
		ProvisionedAt:     "2024-01-01T00:01:00Z",
		ConfigurationID:   "cfg-xyz",
		Uplink:            &client.DeviceUplink{DeviceID: "dev-parent"},
	}
	model := &DeviceDataSourceModel{}
	deviceAPIToModel(model, d)

	if model.MacAddress.ValueString() != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("expected mac 'aa:bb:cc:dd:ee:ff', got %q", model.MacAddress.ValueString())
	}
	if model.IPAddress.ValueString() != "10.0.0.5" {
		t.Errorf("expected ip '10.0.0.5', got %q", model.IPAddress.ValueString())
	}
	if model.Name.ValueString() != "My AP" {
		t.Errorf("expected name 'My AP', got %q", model.Name.ValueString())
	}
	if model.Model.ValueString() != "U6-LR" {
		t.Errorf("expected model 'U6-LR', got %q", model.Model.ValueString())
	}
	if !model.Supported.ValueBool() {
		t.Error("expected supported to be true")
	}
	if model.State.ValueString() != "ONLINE" {
		t.Errorf("expected state 'ONLINE', got %q", model.State.ValueString())
	}
	if model.FirmwareVersion.ValueString() != "6.6.1" {
		t.Errorf("expected firmwareVersion '6.6.1', got %q", model.FirmwareVersion.ValueString())
	}
	if !model.FirmwareUpdatable.ValueBool() {
		t.Error("expected firmwareUpdatable to be true")
	}
	if model.AdoptedAt.ValueString() != "2024-01-01T00:00:00Z" {
		t.Errorf("unexpected adoptedAt: %q", model.AdoptedAt.ValueString())
	}
	if model.ConfigurationID.ValueString() != "cfg-xyz" {
		t.Errorf("expected configurationId 'cfg-xyz', got %q", model.ConfigurationID.ValueString())
	}
	if model.UplinkDeviceID.ValueString() != "dev-parent" {
		t.Errorf("expected uplinkDeviceId 'dev-parent', got %q", model.UplinkDeviceID.ValueString())
	}
}

func TestDeviceAPIToModelWithoutUplink(t *testing.T) {
	d := &client.Device{
		ID:         "dev-2",
		MacAddress: "11:22:33:44:55:66",
		Name:       "Root Switch",
		State:      "ONLINE",
		Uplink:     nil,
	}
	model := &DeviceDataSourceModel{}
	deviceAPIToModel(model, d)

	if !model.UplinkDeviceID.IsNull() {
		t.Errorf("expected null uplinkDeviceId when uplink is nil, got %q", model.UplinkDeviceID.ValueString())
	}
}

func TestDeviceAPIToModelOfflineDevice(t *testing.T) {
	d := &client.Device{
		ID:                "dev-3",
		MacAddress:        "ff:ee:dd:cc:bb:aa",
		State:             "OFFLINE",
		Supported:         false,
		FirmwareUpdatable: false,
	}
	model := &DeviceDataSourceModel{}
	deviceAPIToModel(model, d)

	if model.State.ValueString() != "OFFLINE" {
		t.Errorf("expected state 'OFFLINE', got %q", model.State.ValueString())
	}
	if model.Supported.ValueBool() {
		t.Error("expected supported to be false")
	}
}
