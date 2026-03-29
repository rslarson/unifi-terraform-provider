package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/rslarson/terraform-provider-unifi/internal/client"
)

// --- Network resource ---

func TestNetworkResourceMetadata(t *testing.T) {
	r := NewNetworkResource()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "unifi"}, resp)

	if resp.TypeName != "unifi_network" {
		t.Errorf("expected type name 'unifi_network', got %q", resp.TypeName)
	}
}

func TestNetworkResourceSchema(t *testing.T) {
	r := NewNetworkResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	requiredAttrs := []string{"site_id", "name", "management", "enabled", "vlan_id"}
	for _, attr := range requiredAttrs {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Errorf("expected attribute %q in network schema", attr)
			continue
		}
		if !a.IsRequired() {
			t.Errorf("expected attribute %q to be required", attr)
		}
	}

	idAttr, ok := resp.Schema.Attributes["id"]
	if !ok {
		t.Fatal("expected 'id' attribute")
	}
	if !idAttr.IsComputed() {
		t.Error("expected 'id' to be computed")
	}

	dhcpAttr, ok := resp.Schema.Attributes["trusted_dhcp_server_ip_addresses"]
	if !ok {
		t.Fatal("expected 'trusted_dhcp_server_ip_addresses' attribute")
	}
	if !dhcpAttr.IsOptional() {
		t.Error("expected 'trusted_dhcp_server_ip_addresses' to be optional")
	}
	if dhcpAttr.IsRequired() {
		t.Error("expected 'trusted_dhcp_server_ip_addresses' to not be required")
	}
}

func TestNetworkResourceSchemaAttributeCount(t *testing.T) {
	r := NewNetworkResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	// id, site_id, name, management, enabled, vlan_id, trusted_dhcp_server_ip_addresses,
	// isolation_enabled, cellular_backup_enabled, internet_access_enabled,
	// mdns_forwarding_enabled, zone_id, device_id, ipv4_configuration
	expectedCount := 14
	if len(resp.Schema.Attributes) != expectedCount {
		t.Errorf("expected %d attributes, got %d", expectedCount, len(resp.Schema.Attributes))
	}
}

func TestNetworkResourceConfigure(t *testing.T) {
	r := &NetworkResource{}

	// Nil provider data — should not error (provider not yet configured)
	resp := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: nil}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected error for nil provider data: %v", resp.Diagnostics)
	}

	// Wrong type — should error
	resp2 := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: "wrong"}, resp2)
	if !resp2.Diagnostics.HasError() {
		t.Error("expected error for wrong provider data type")
	}

	// Correct type — should succeed
	c := client.NewClient("key", "host")
	resp3 := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: c}, resp3)
	if resp3.Diagnostics.HasError() {
		t.Errorf("unexpected error for correct provider data: %v", resp3.Diagnostics)
	}
}

// --- WiFi Broadcast resource ---

func TestWifiBroadcastResourceMetadata(t *testing.T) {
	r := NewWifiBroadcastResource()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "unifi"}, resp)

	if resp.TypeName != "unifi_wifi_broadcast" {
		t.Errorf("expected type name 'unifi_wifi_broadcast', got %q", resp.TypeName)
	}
}

func TestWifiBroadcastResourceSchema(t *testing.T) {
	r := NewWifiBroadcastResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	requiredAttrs := []string{"site_id", "type", "name", "enabled", "security_type", "network_type"}
	for _, attr := range requiredAttrs {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Errorf("expected attribute %q in wifi broadcast schema", attr)
			continue
		}
		if !a.IsRequired() {
			t.Errorf("expected attribute %q to be required", attr)
		}
	}

	optionalAttrs := []string{
		"passphrase", "passphrase_wo", "passphrase_wo_version",
		"network_id", "client_isolation_enabled", "hide_name",
		"multicast_to_unicast_conversion_enabled", "uapsd_enabled",
		"basic_data_rate_24ghz", "basic_data_rate_5ghz",
		"client_filter_action", "client_filter_mac_addresses",
		"blackout_schedule_days", "broadcasting_frequencies_ghz",
		"broadcasting_device_filter_type", "broadcasting_device_filter_ids",
		"multicast_filter_action", "mdns_proxy_mode",
		"band_steering_enabled", "mlo_enabled", "arp_proxy_enabled",
		"bss_transition_enabled", "advertise_device_name",
	}
	for _, attr := range optionalAttrs {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Errorf("expected optional attribute %q in wifi broadcast schema", attr)
			continue
		}
		if a.IsRequired() {
			t.Errorf("expected attribute %q to NOT be required", attr)
		}
	}

	// Passphrase sensitivity
	passphrase, ok := resp.Schema.Attributes["passphrase"]
	if !ok {
		t.Fatal("expected 'passphrase' attribute")
	}
	if !passphrase.IsSensitive() {
		t.Error("expected 'passphrase' to be sensitive")
	}

	// Write-only passphrase
	passphraseWO, ok := resp.Schema.Attributes["passphrase_wo"]
	if !ok {
		t.Fatal("expected 'passphrase_wo' attribute")
	}
	if !passphraseWO.IsWriteOnly() {
		t.Error("expected 'passphrase_wo' to be write-only")
	}

	// Version tracker
	woVersion, ok := resp.Schema.Attributes["passphrase_wo_version"]
	if !ok {
		t.Fatal("expected 'passphrase_wo_version' attribute")
	}
	if !woVersion.IsOptional() {
		t.Error("expected 'passphrase_wo_version' to be optional")
	}
}

func TestWifiBroadcastResourceConfigure(t *testing.T) {
	r := &WifiBroadcastResource{}

	resp := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: nil}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected error for nil: %v", resp.Diagnostics)
	}

	resp2 := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: 42}, resp2)
	if !resp2.Diagnostics.HasError() {
		t.Error("expected error for wrong type")
	}
}

// wifiBroadcastConfig builds a tfsdk.Config for ValidateConfig tests.
func wifiBroadcastConfig(t *testing.T, attrs map[string]tftypes.Value) tfsdk.Config {
	t.Helper()

	r := NewWifiBroadcastResource()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema errors: %v", schemaResp.Diagnostics)
	}

	defaults := map[string]tftypes.Value{
		"id":            tftypes.NewValue(tftypes.String, nil),
		"site_id":       tftypes.NewValue(tftypes.String, "site-1"),
		"type":          tftypes.NewValue(tftypes.String, "STANDARD"),
		"name":          tftypes.NewValue(tftypes.String, "Test WiFi"),
		"enabled":       tftypes.NewValue(tftypes.Bool, true),
		"security_type": tftypes.NewValue(tftypes.String, nil),
		"passphrase":    tftypes.NewValue(tftypes.String, nil),
		"passphrase_wo": tftypes.NewValue(tftypes.String, nil),
		"passphrase_wo_version":                    tftypes.NewValue(tftypes.Number, nil),
		"network_type":                             tftypes.NewValue(tftypes.String, "NATIVE"),
		"network_id":                               tftypes.NewValue(tftypes.String, nil),
		"client_isolation_enabled":                 tftypes.NewValue(tftypes.Bool, nil),
		"hide_name":                                tftypes.NewValue(tftypes.Bool, nil),
		"multicast_to_unicast_conversion_enabled":  tftypes.NewValue(tftypes.Bool, nil),
		"uapsd_enabled":                            tftypes.NewValue(tftypes.Bool, nil),
		"basic_data_rate_24ghz":                    tftypes.NewValue(tftypes.Number, nil),
		"basic_data_rate_5ghz":                     tftypes.NewValue(tftypes.Number, nil),
		"client_filter_action":                     tftypes.NewValue(tftypes.String, nil),
		"client_filter_mac_addresses":              tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
		"blackout_schedule_days": tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"type": tftypes.String, "day": tftypes.String, "start_time": tftypes.String, "end_time": tftypes.String,
		}}}, nil),
		"broadcasting_frequencies_ghz":             tftypes.NewValue(tftypes.List{ElementType: tftypes.Number}, nil),
		"broadcasting_device_filter_type":          tftypes.NewValue(tftypes.String, nil),
		"broadcasting_device_filter_ids":           tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
		"multicast_filter_action":                  tftypes.NewValue(tftypes.String, nil),
		"mdns_proxy_mode":                          tftypes.NewValue(tftypes.String, nil),
		"band_steering_enabled":                    tftypes.NewValue(tftypes.Bool, nil),
		"mlo_enabled":                              tftypes.NewValue(tftypes.Bool, nil),
		"arp_proxy_enabled":                        tftypes.NewValue(tftypes.Bool, nil),
		"bss_transition_enabled":                   tftypes.NewValue(tftypes.Bool, nil),
		"advertise_device_name":                    tftypes.NewValue(tftypes.Bool, nil),
	}
	for k, v := range attrs {
		defaults[k] = v
	}

	tfType := schemaResp.Schema.Type().TerraformType(context.Background())
	rawVal, err := schemaResp.Schema.Type().ValueFromTerraform(context.Background(),
		tftypes.NewValue(tfType, defaults),
	)
	if err != nil {
		t.Fatalf("failed to create config value: %v", err)
	}
	raw, err2 := rawVal.ToTerraformValue(context.Background())
	if err2 != nil {
		t.Fatalf("failed to convert to terraform value: %v", err2)
	}
	return tfsdk.Config{Schema: schemaResp.Schema, Raw: raw}
}

func TestWifiBroadcastValidateConfig(t *testing.T) {
	tests := []struct {
		name         string
		attrs        map[string]tftypes.Value
		wantErrors   []string
		wantWarnings []string
	}{
		{
			name: "personal security with passphrase succeeds",
			attrs: map[string]tftypes.Value{
				"security_type": tftypes.NewValue(tftypes.String, client.SecurityWPA2Personal),
				"passphrase":    tftypes.NewValue(tftypes.String, "mypassword123"),
			},
		},
		{
			name: "personal security with passphrase_wo and version succeeds",
			attrs: map[string]tftypes.Value{
				"security_type":         tftypes.NewValue(tftypes.String, client.SecurityWPA2WPA3Personal),
				"passphrase_wo":         tftypes.NewValue(tftypes.String, "mypassword123"),
				"passphrase_wo_version": tftypes.NewValue(tftypes.Number, 1),
			},
		},
		{
			name: "WPA3 personal with passphrase succeeds",
			attrs: map[string]tftypes.Value{
				"security_type": tftypes.NewValue(tftypes.String, client.SecurityWPA3Personal),
				"passphrase":    tftypes.NewValue(tftypes.String, "wpa3pass123"),
			},
		},
		{
			name: "personal security without passphrase errors",
			attrs: map[string]tftypes.Value{
				"security_type": tftypes.NewValue(tftypes.String, client.SecurityWPA3Personal),
			},
			wantErrors: []string{"Missing passphrase"},
		},
		{
			name: "WPA2 personal without passphrase errors",
			attrs: map[string]tftypes.Value{
				"security_type": tftypes.NewValue(tftypes.String, client.SecurityWPA2Personal),
			},
			wantErrors: []string{"Missing passphrase"},
		},
		{
			name: "open security without passphrase succeeds",
			attrs: map[string]tftypes.Value{
				"security_type": tftypes.NewValue(tftypes.String, client.SecurityOpen),
			},
		},
		{
			name: "open security with passphrase errors",
			attrs: map[string]tftypes.Value{
				"security_type": tftypes.NewValue(tftypes.String, client.SecurityOpen),
				"passphrase":    tftypes.NewValue(tftypes.String, "mypassword123"),
			},
			wantErrors: []string{"Unexpected passphrase"},
		},
		{
			name: "open security with passphrase_wo errors",
			attrs: map[string]tftypes.Value{
				"security_type":         tftypes.NewValue(tftypes.String, client.SecurityOpen),
				"passphrase_wo":         tftypes.NewValue(tftypes.String, "mypassword123"),
				"passphrase_wo_version": tftypes.NewValue(tftypes.Number, 1),
			},
			wantErrors: []string{"Unexpected passphrase"},
		},
		{
			name: "WPA2 enterprise with passphrase errors",
			attrs: map[string]tftypes.Value{
				"security_type": tftypes.NewValue(tftypes.String, client.SecurityWPA2Enterprise),
				"passphrase":    tftypes.NewValue(tftypes.String, "mypassword123"),
			},
			wantErrors: []string{"Unexpected passphrase"},
		},
		{
			name: "WPA3 enterprise with passphrase_wo errors",
			attrs: map[string]tftypes.Value{
				"security_type":         tftypes.NewValue(tftypes.String, client.SecurityWPA3Enterprise),
				"passphrase_wo":         tftypes.NewValue(tftypes.String, "mypassword123"),
				"passphrase_wo_version": tftypes.NewValue(tftypes.Number, 1),
			},
			wantErrors: []string{"Unexpected passphrase"},
		},
		{
			name: "WPA2/WPA3 enterprise without passphrase succeeds",
			attrs: map[string]tftypes.Value{
				"security_type": tftypes.NewValue(tftypes.String, client.SecurityWPA2WPA3Enterprise),
			},
		},
		{
			name: "passphrase_wo without version warns",
			attrs: map[string]tftypes.Value{
				"security_type": tftypes.NewValue(tftypes.String, client.SecurityWPA2Personal),
				"passphrase_wo": tftypes.NewValue(tftypes.String, "mypassword123"),
			},
			wantWarnings: []string{"Missing passphrase_wo_version"},
		},
		{
			name: "passphrase_wo with version does not warn",
			attrs: map[string]tftypes.Value{
				"security_type":         tftypes.NewValue(tftypes.String, client.SecurityWPA2Personal),
				"passphrase_wo":         tftypes.NewValue(tftypes.String, "mypassword123"),
				"passphrase_wo_version": tftypes.NewValue(tftypes.Number, 1),
			},
		},
		{
			name: "null security type skips validation",
			attrs: map[string]tftypes.Value{
				"security_type": tftypes.NewValue(tftypes.String, nil),
				"passphrase":    tftypes.NewValue(tftypes.String, "mypassword123"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &WifiBroadcastResource{}
			resp := &resource.ValidateConfigResponse{}
			r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{
				Config: wifiBroadcastConfig(t, tt.attrs),
			}, resp)

			if len(tt.wantErrors) == 0 && resp.Diagnostics.HasError() {
				t.Fatalf("unexpected errors: %v", resp.Diagnostics)
			}
			for _, want := range tt.wantErrors {
				found := false
				for _, d := range resp.Diagnostics.Errors() {
					if d.Summary() == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error %q, got: %v", want, resp.Diagnostics)
				}
			}
			for _, want := range tt.wantWarnings {
				found := false
				for _, d := range resp.Diagnostics.Warnings() {
					if d.Summary() == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected warning %q, got: %v", want, resp.Diagnostics)
				}
			}
			if len(tt.wantWarnings) == 0 && len(resp.Diagnostics.Warnings()) > 0 {
				t.Errorf("unexpected warnings: %v", resp.Diagnostics.Warnings())
			}
		})
	}
}

// --- Firewall Zone resource ---

func TestFirewallZoneResourceMetadata(t *testing.T) {
	r := NewFirewallZoneResource()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "unifi"}, resp)

	if resp.TypeName != "unifi_firewall_zone" {
		t.Errorf("expected type name 'unifi_firewall_zone', got %q", resp.TypeName)
	}
}

func TestFirewallZoneResourceSchema(t *testing.T) {
	r := NewFirewallZoneResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	requiredAttrs := []string{"site_id", "name", "network_ids"}
	for _, attr := range requiredAttrs {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Errorf("expected attribute %q in firewall zone schema", attr)
			continue
		}
		if !a.IsRequired() {
			t.Errorf("expected attribute %q to be required", attr)
		}
	}

	idAttr, ok := resp.Schema.Attributes["id"]
	if !ok {
		t.Fatal("expected 'id' attribute")
	}
	if !idAttr.IsComputed() {
		t.Error("expected 'id' to be computed")
	}
}

func TestFirewallZoneResourceConfigure(t *testing.T) {
	r := &FirewallZoneResource{}

	resp := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: nil}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected error for nil: %v", resp.Diagnostics)
	}

	resp2 := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: true}, resp2)
	if !resp2.Diagnostics.HasError() {
		t.Error("expected error for wrong type")
	}

	c := client.NewClient("key", "host")
	resp3 := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: c}, resp3)
	if resp3.Diagnostics.HasError() {
		t.Errorf("unexpected error for correct type: %v", resp3.Diagnostics)
	}
}

// --- ACL Rule resource ---

func TestAclRuleResourceMetadata(t *testing.T) {
	r := NewAclRuleResource()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "unifi"}, resp)

	if resp.TypeName != "unifi_acl_rule" {
		t.Errorf("expected type name 'unifi_acl_rule', got %q", resp.TypeName)
	}
}

func TestAclRuleResourceSchema(t *testing.T) {
	r := NewAclRuleResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	requiredAttrs := []string{"site_id", "type", "enabled", "name", "action"}
	for _, attr := range requiredAttrs {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Errorf("expected attribute %q in acl rule schema", attr)
			continue
		}
		if !a.IsRequired() {
			t.Errorf("expected attribute %q to be required", attr)
		}
	}

	idAttr, ok := resp.Schema.Attributes["id"]
	if !ok {
		t.Fatal("expected 'id' attribute")
	}
	if !idAttr.IsComputed() {
		t.Error("expected 'id' to be computed")
	}

	optionalAttrs := []string{
		"description", "source_filter_type", "source_filter_values", "source_filter_ports",
		"destination_filter_type", "destination_filter_values", "destination_filter_ports",
		"protocol_filter", "enforcing_device_ids", "network_id_filter",
	}
	for _, attr := range optionalAttrs {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Errorf("expected optional attribute %q in acl rule schema", attr)
			continue
		}
		if a.IsRequired() {
			t.Errorf("expected attribute %q to NOT be required", attr)
		}
	}
}

func TestAclRuleResourceConfigure(t *testing.T) {
	r := &AclRuleResource{}

	resp := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: nil}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected error for nil: %v", resp.Diagnostics)
	}

	resp2 := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: true}, resp2)
	if !resp2.Diagnostics.HasError() {
		t.Error("expected error for wrong type")
	}

	c := client.NewClient("key", "host")
	resp3 := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: c}, resp3)
	if resp3.Diagnostics.HasError() {
		t.Errorf("unexpected error for correct type: %v", resp3.Diagnostics)
	}
}

// --- Firewall Policy resource ---

func TestFirewallPolicyResourceMetadata(t *testing.T) {
	r := NewFirewallPolicyResource()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "unifi"}, resp)

	if resp.TypeName != "unifi_firewall_policy" {
		t.Errorf("expected type name 'unifi_firewall_policy', got %q", resp.TypeName)
	}
}

func TestFirewallPolicyResourceSchema(t *testing.T) {
	r := NewFirewallPolicyResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	requiredAttrs := []string{
		"site_id", "enabled", "name", "action_type",
		"source_zone_id", "destination_zone_id", "ip_version", "logging_enabled",
	}
	for _, attr := range requiredAttrs {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Errorf("expected attribute %q in firewall policy schema", attr)
			continue
		}
		if !a.IsRequired() {
			t.Errorf("expected attribute %q to be required", attr)
		}
	}

	idAttr, ok := resp.Schema.Attributes["id"]
	if !ok {
		t.Fatal("expected 'id' attribute")
	}
	if !idAttr.IsComputed() {
		t.Error("expected 'id' to be computed")
	}

	optionalAttrs := []string{
		"description", "allow_return_traffic",
		"source_traffic_filter_type", "source_traffic_filter_values",
		"destination_traffic_filter_type", "destination_traffic_filter_values",
		"connection_state_filter", "ipsec_filter",
	}
	for _, attr := range optionalAttrs {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Errorf("expected optional attribute %q in firewall policy schema", attr)
			continue
		}
		if a.IsRequired() {
			t.Errorf("expected attribute %q to NOT be required", attr)
		}
	}
}

func TestFirewallPolicyResourceConfigure(t *testing.T) {
	r := &FirewallPolicyResource{}

	resp := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: nil}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected error for nil: %v", resp.Diagnostics)
	}

	resp2 := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: "wrong"}, resp2)
	if !resp2.Diagnostics.HasError() {
		t.Error("expected error for wrong type")
	}
}

// --- Traffic Matching List resource ---

func TestTrafficMatchingListResourceMetadata(t *testing.T) {
	r := NewTrafficMatchingListResource()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "unifi"}, resp)

	if resp.TypeName != "unifi_traffic_matching_list" {
		t.Errorf("expected type name 'unifi_traffic_matching_list', got %q", resp.TypeName)
	}
}

func TestTrafficMatchingListResourceSchema(t *testing.T) {
	r := NewTrafficMatchingListResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	requiredAttrs := []string{"site_id", "type", "name", "items"}
	for _, attr := range requiredAttrs {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Errorf("expected attribute %q in traffic matching list schema", attr)
			continue
		}
		if !a.IsRequired() {
			t.Errorf("expected attribute %q to be required", attr)
		}
	}

	idAttr, ok := resp.Schema.Attributes["id"]
	if !ok {
		t.Fatal("expected 'id' attribute")
	}
	if !idAttr.IsComputed() {
		t.Error("expected 'id' to be computed")
	}
}

func TestTrafficMatchingListResourceConfigure(t *testing.T) {
	r := &TrafficMatchingListResource{}

	resp := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: nil}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected error for nil: %v", resp.Diagnostics)
	}

	resp2 := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: 42}, resp2)
	if !resp2.Diagnostics.HasError() {
		t.Error("expected error for wrong type")
	}
}

// --- DNS Policy resource ---

func TestDnsPolicyResourceMetadata(t *testing.T) {
	r := NewDnsPolicyResource()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "unifi"}, resp)

	if resp.TypeName != "unifi_dns_policy" {
		t.Errorf("expected type name 'unifi_dns_policy', got %q", resp.TypeName)
	}
}

func TestDnsPolicyResourceSchema(t *testing.T) {
	r := NewDnsPolicyResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	requiredAttrs := []string{"site_id", "type", "enabled"}
	for _, attr := range requiredAttrs {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Errorf("expected attribute %q in dns policy schema", attr)
			continue
		}
		if !a.IsRequired() {
			t.Errorf("expected attribute %q to be required", attr)
		}
	}

	idAttr, ok := resp.Schema.Attributes["id"]
	if !ok {
		t.Fatal("expected 'id' attribute")
	}
	if !idAttr.IsComputed() {
		t.Error("expected 'id' to be computed")
	}

	optionalAttrs := []string{
		"domain", "ipv4_address", "ipv6_address", "target",
		"ttl_seconds", "priority", "weight", "port", "txt_value", "forward_to",
	}
	for _, attr := range optionalAttrs {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Errorf("expected optional attribute %q in dns policy schema", attr)
			continue
		}
		if a.IsRequired() {
			t.Errorf("expected attribute %q to NOT be required", attr)
		}
	}
}

func TestDnsPolicyResourceConfigure(t *testing.T) {
	r := &DnsPolicyResource{}

	resp := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: nil}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected error for nil: %v", resp.Diagnostics)
	}

	resp2 := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: false}, resp2)
	if !resp2.Diagnostics.HasError() {
		t.Error("expected error for wrong type")
	}
}

// --- Hotspot Voucher resource ---

func TestHotspotVoucherResourceMetadata(t *testing.T) {
	r := NewHotspotVoucherResource()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "unifi"}, resp)

	if resp.TypeName != "unifi_hotspot_voucher" {
		t.Errorf("expected type name 'unifi_hotspot_voucher', got %q", resp.TypeName)
	}
}

func TestHotspotVoucherResourceSchema(t *testing.T) {
	r := NewHotspotVoucherResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	requiredAttrs := []string{"site_id", "name", "time_limit_minutes"}
	for _, attr := range requiredAttrs {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Errorf("expected attribute %q in hotspot voucher schema", attr)
			continue
		}
		if !a.IsRequired() {
			t.Errorf("expected attribute %q to be required", attr)
		}
	}

	idAttr, ok := resp.Schema.Attributes["id"]
	if !ok {
		t.Fatal("expected 'id' attribute")
	}
	if !idAttr.IsComputed() {
		t.Error("expected 'id' to be computed")
	}

	codeAttr, ok := resp.Schema.Attributes["code"]
	if !ok {
		t.Fatal("expected 'code' attribute")
	}
	if !codeAttr.IsComputed() {
		t.Error("expected 'code' to be computed")
	}
	if !codeAttr.IsSensitive() {
		t.Error("expected 'code' to be sensitive")
	}

	optionalAttrs := []string{
		"authorized_guest_limit", "data_usage_limit_mbytes",
		"rx_rate_limit_kbps", "tx_rate_limit_kbps",
	}
	for _, attr := range optionalAttrs {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Errorf("expected optional attribute %q in hotspot voucher schema", attr)
			continue
		}
		if a.IsRequired() {
			t.Errorf("expected attribute %q to NOT be required", attr)
		}
	}
}

func TestHotspotVoucherResourceConfigure(t *testing.T) {
	r := &HotspotVoucherResource{}

	resp := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: nil}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected error for nil: %v", resp.Diagnostics)
	}

	resp2 := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: 123}, resp2)
	if !resp2.Diagnostics.HasError() {
		t.Error("expected error for wrong type")
	}
}
