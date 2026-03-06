package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestNetworkDataSourceSchema(t *testing.T) {
	d := NewNetworkDataSource()
	resp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Required inputs
	for _, attr := range []string{"id", "site_id"} {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Errorf("expected attribute %q", attr)
			continue
		}
		if !a.IsRequired() {
			t.Errorf("expected attribute %q to be required", attr)
		}
	}

	// Computed outputs
	for _, attr := range []string{"name", "management", "enabled", "vlan_id"} {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Errorf("expected attribute %q", attr)
			continue
		}
		if !a.IsComputed() {
			t.Errorf("expected attribute %q to be computed", attr)
		}
	}
}

func TestNetworkDataSourceMetadata(t *testing.T) {
	d := NewNetworkDataSource()
	resp := &datasource.MetadataResponse{}
	d.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "unifi"}, resp)

	if resp.TypeName != "unifi_network" {
		t.Errorf("expected type name 'unifi_network', got %q", resp.TypeName)
	}
}

func TestWifiBroadcastDataSourceSchema(t *testing.T) {
	d := NewWifiBroadcastDataSource()
	resp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	for _, attr := range []string{"id", "site_id"} {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Errorf("expected attribute %q", attr)
			continue
		}
		if !a.IsRequired() {
			t.Errorf("expected attribute %q to be required", attr)
		}
	}
}

func TestWifiBroadcastDataSourceMetadata(t *testing.T) {
	d := NewWifiBroadcastDataSource()
	resp := &datasource.MetadataResponse{}
	d.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "unifi"}, resp)

	if resp.TypeName != "unifi_wifi_broadcast" {
		t.Errorf("expected type name 'unifi_wifi_broadcast', got %q", resp.TypeName)
	}
}

func TestFirewallZoneDataSourceSchema(t *testing.T) {
	d := NewFirewallZoneDataSource()
	resp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	for _, attr := range []string{"id", "site_id"} {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Errorf("expected attribute %q", attr)
			continue
		}
		if !a.IsRequired() {
			t.Errorf("expected attribute %q to be required", attr)
		}
	}
}

func TestFirewallZoneDataSourceMetadata(t *testing.T) {
	d := NewFirewallZoneDataSource()
	resp := &datasource.MetadataResponse{}
	d.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "unifi"}, resp)

	if resp.TypeName != "unifi_firewall_zone" {
		t.Errorf("expected type name 'unifi_firewall_zone', got %q", resp.TypeName)
	}
}

func TestDeviceDataSourceSchema(t *testing.T) {
	d := NewDeviceDataSource()
	resp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	for _, attr := range []string{"id", "site_id"} {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Errorf("expected attribute %q", attr)
			continue
		}
		if !a.IsRequired() {
			t.Errorf("expected attribute %q to be required", attr)
		}
	}

	computedAttrs := []string{
		"mac_address", "ip_address", "name", "model", "supported",
		"state", "firmware_version", "firmware_updatable",
		"adopted_at", "provisioned_at", "configuration_id", "uplink_device_id",
	}
	for _, attr := range computedAttrs {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Errorf("expected attribute %q", attr)
			continue
		}
		if !a.IsComputed() {
			t.Errorf("expected attribute %q to be computed", attr)
		}
	}
}

func TestDeviceDataSourceMetadata(t *testing.T) {
	d := NewDeviceDataSource()
	resp := &datasource.MetadataResponse{}
	d.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "unifi"}, resp)

	if resp.TypeName != "unifi_device" {
		t.Errorf("expected type name 'unifi_device', got %q", resp.TypeName)
	}
}

func TestSitesDataSourceSchema(t *testing.T) {
	d := NewSitesDataSource()
	resp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	_, ok := resp.Schema.Attributes["sites"]
	if !ok {
		t.Fatal("expected 'sites' attribute")
	}
}

func TestSitesDataSourceMetadata(t *testing.T) {
	d := NewSitesDataSource()
	resp := &datasource.MetadataResponse{}
	d.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "unifi"}, resp)

	if resp.TypeName != "unifi_sites" {
		t.Errorf("expected type name 'unifi_sites', got %q", resp.TypeName)
	}
}
