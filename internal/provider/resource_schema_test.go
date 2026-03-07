package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/rslarson/terraform-provider-unifi/internal/client"
)

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
}

func TestNetworkResourceMetadata(t *testing.T) {
	r := NewNetworkResource()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "unifi"}, resp)

	if resp.TypeName != "unifi_network" {
		t.Errorf("expected type name 'unifi_network', got %q", resp.TypeName)
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

	// Verify passphrase is sensitive
	passphrase, ok := resp.Schema.Attributes["passphrase"]
	if !ok {
		t.Fatal("expected 'passphrase' attribute")
	}
	if !passphrase.IsSensitive() {
		t.Error("expected 'passphrase' to be sensitive")
	}

	// Verify passphrase_wo exists and is write-only
	passphraseWO, ok := resp.Schema.Attributes["passphrase_wo"]
	if !ok {
		t.Fatal("expected 'passphrase_wo' attribute")
	}
	if !passphraseWO.IsWriteOnly() {
		t.Error("expected 'passphrase_wo' to be write-only")
	}

	// Verify passphrase_wo_version exists and is optional
	woVersion, ok := resp.Schema.Attributes["passphrase_wo_version"]
	if !ok {
		t.Fatal("expected 'passphrase_wo_version' attribute")
	}
	if !woVersion.IsOptional() {
		t.Error("expected 'passphrase_wo_version' to be optional")
	}
}

func TestWifiBroadcastResourceMetadata(t *testing.T) {
	r := NewWifiBroadcastResource()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "unifi"}, resp)

	if resp.TypeName != "unifi_wifi_broadcast" {
		t.Errorf("expected type name 'unifi_wifi_broadcast', got %q", resp.TypeName)
	}
}

// wifiBroadcastConfig builds a tfsdk.Config for the wifi broadcast resource
// with the given attribute overrides. Unspecified attributes use sensible
// defaults (null for optional, valid values for required).
func wifiBroadcastConfig(t *testing.T, attrs map[string]tftypes.Value) tfsdk.Config {
	t.Helper()

	r := NewWifiBroadcastResource()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema errors: %v", schemaResp.Diagnostics)
	}

	// Build default values for all attributes.
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

	return tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw:    raw,
	}
}

func TestWifiBroadcastValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		attrs       map[string]tftypes.Value
		wantErrors  []string
		wantWarnings []string
	}{
		{
			name: "personal security type with passphrase",
			attrs: map[string]tftypes.Value{
				"security_type": tftypes.NewValue(tftypes.String, client.SecurityWPA2Personal),
				"passphrase":    tftypes.NewValue(tftypes.String, "mypassword123"),
			},
		},
		{
			name: "personal security type with passphrase_wo and version",
			attrs: map[string]tftypes.Value{
				"security_type":         tftypes.NewValue(tftypes.String, client.SecurityWPA2WPA3Personal),
				"passphrase_wo":         tftypes.NewValue(tftypes.String, "mypassword123"),
				"passphrase_wo_version": tftypes.NewValue(tftypes.Number, 1),
			},
		},
		{
			name: "personal security type without passphrase errors",
			attrs: map[string]tftypes.Value{
				"security_type": tftypes.NewValue(tftypes.String, client.SecurityWPA3Personal),
			},
			wantErrors: []string{"Missing passphrase"},
		},
		{
			name: "open security type without passphrase succeeds",
			attrs: map[string]tftypes.Value{
				"security_type": tftypes.NewValue(tftypes.String, client.SecurityOpen),
			},
		},
		{
			name: "open security type with passphrase errors",
			attrs: map[string]tftypes.Value{
				"security_type": tftypes.NewValue(tftypes.String, client.SecurityOpen),
				"passphrase":    tftypes.NewValue(tftypes.String, "mypassword123"),
			},
			wantErrors: []string{"Unexpected passphrase"},
		},
		{
			name: "enterprise security type with passphrase_wo errors",
			attrs: map[string]tftypes.Value{
				"security_type":         tftypes.NewValue(tftypes.String, client.SecurityWPA2Enterprise),
				"passphrase_wo":         tftypes.NewValue(tftypes.String, "mypassword123"),
				"passphrase_wo_version": tftypes.NewValue(tftypes.Number, 1),
			},
			wantErrors: []string{"Unexpected passphrase"},
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
			name: "null security type with passphrase skips validation",
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

			// Check expected errors.
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

			// Check expected warnings.
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

			// Check no unexpected warnings.
			if len(tt.wantWarnings) == 0 && len(resp.Diagnostics.Warnings()) > 0 {
				t.Errorf("unexpected warnings: %v", resp.Diagnostics.Warnings())
			}
		})
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
}

func TestFirewallZoneResourceMetadata(t *testing.T) {
	r := NewFirewallZoneResource()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "unifi"}, resp)

	if resp.TypeName != "unifi_firewall_zone" {
		t.Errorf("expected type name 'unifi_firewall_zone', got %q", resp.TypeName)
	}
}
