package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
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
}

func TestWifiBroadcastResourceMetadata(t *testing.T) {
	r := NewWifiBroadcastResource()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "unifi"}, resp)

	if resp.TypeName != "unifi_wifi_broadcast" {
		t.Errorf("expected type name 'unifi_wifi_broadcast', got %q", resp.TypeName)
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
