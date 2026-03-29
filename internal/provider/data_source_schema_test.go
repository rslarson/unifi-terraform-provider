package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

// --- Network data source ---

func TestNetworkDataSourceMetadata(t *testing.T) {
	d := NewNetworkDataSource()
	resp := &datasource.MetadataResponse{}
	d.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "unifi"}, resp)

	if resp.TypeName != "unifi_network" {
		t.Errorf("expected type name 'unifi_network', got %q", resp.TypeName)
	}
}

func TestNetworkDataSourceSchema(t *testing.T) {
	d := NewNetworkDataSource()
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

	for _, attr := range []string{"name", "management", "enabled", "vlan_id", "trusted_dhcp_server_ip_addresses"} {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Errorf("expected computed attribute %q", attr)
			continue
		}
		if !a.IsComputed() {
			t.Errorf("expected attribute %q to be computed", attr)
		}
	}
}

func TestNetworkDataSourceNoPassphrase(t *testing.T) {
	d := NewNetworkDataSource()
	resp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, resp)

	// Network data source should not have passphrase (it's not a field on networks)
	if _, ok := resp.Schema.Attributes["passphrase"]; ok {
		t.Error("network data source should not have 'passphrase' attribute")
	}
}

// --- WiFi Broadcast data source ---

func TestWifiBroadcastDataSourceMetadata(t *testing.T) {
	d := NewWifiBroadcastDataSource()
	resp := &datasource.MetadataResponse{}
	d.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "unifi"}, resp)

	if resp.TypeName != "unifi_wifi_broadcast" {
		t.Errorf("expected type name 'unifi_wifi_broadcast', got %q", resp.TypeName)
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

	computedAttrs := []string{
		"type", "name", "enabled", "security_type", "network_type", "network_id",
		"client_isolation_enabled", "hide_name", "multicast_to_unicast_conversion_enabled",
		"uapsd_enabled", "basic_data_rate_24ghz", "basic_data_rate_5ghz",
		"client_filter_action", "client_filter_mac_addresses", "blackout_schedule_days",
		"broadcasting_frequencies_ghz", "broadcasting_device_filter_type",
		"broadcasting_device_filter_ids", "multicast_filter_action", "mdns_proxy_mode",
		"band_steering_enabled", "mlo_enabled", "arp_proxy_enabled",
		"bss_transition_enabled", "advertise_device_name",
	}
	for _, attr := range computedAttrs {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Errorf("expected computed attribute %q", attr)
			continue
		}
		if !a.IsComputed() {
			t.Errorf("expected attribute %q to be computed", attr)
		}
	}
}

func TestWifiBroadcastDataSourceNoPassphrase(t *testing.T) {
	// Passphrase is never returned by the API on read, so must NOT be in the data source
	d := NewWifiBroadcastDataSource()
	resp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, resp)

	if _, ok := resp.Schema.Attributes["passphrase"]; ok {
		t.Error("wifi broadcast data source must not expose 'passphrase' (API never returns it)")
	}
	if _, ok := resp.Schema.Attributes["passphrase_wo"]; ok {
		t.Error("wifi broadcast data source must not expose 'passphrase_wo'")
	}
}

func TestWifiBroadcastDataSourceConfigure(t *testing.T) {
	d := NewWifiBroadcastDataSource().(*WifiBroadcastDataSource)

	resp := &datasource.ConfigureResponse{}
	d.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: nil}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected error for nil: %v", resp.Diagnostics)
	}

	resp2 := &datasource.ConfigureResponse{}
	d.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: "wrong"}, resp2)
	if !resp2.Diagnostics.HasError() {
		t.Error("expected error for wrong provider data type")
	}
}

// --- Firewall Zone data source ---

func TestFirewallZoneDataSourceMetadata(t *testing.T) {
	d := NewFirewallZoneDataSource()
	resp := &datasource.MetadataResponse{}
	d.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "unifi"}, resp)

	if resp.TypeName != "unifi_firewall_zone" {
		t.Errorf("expected type name 'unifi_firewall_zone', got %q", resp.TypeName)
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

	for _, attr := range []string{"name", "network_ids"} {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Errorf("expected computed attribute %q", attr)
			continue
		}
		if !a.IsComputed() {
			t.Errorf("expected attribute %q to be computed", attr)
		}
	}
}

// --- Device data source ---

func TestDeviceDataSourceMetadata(t *testing.T) {
	d := NewDeviceDataSource()
	resp := &datasource.MetadataResponse{}
	d.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "unifi"}, resp)

	if resp.TypeName != "unifi_device" {
		t.Errorf("expected type name 'unifi_device', got %q", resp.TypeName)
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
			t.Errorf("expected computed attribute %q", attr)
			continue
		}
		if !a.IsComputed() {
			t.Errorf("expected attribute %q to be computed", attr)
		}
	}
}

func TestDeviceDataSourceSchemaAttributeCount(t *testing.T) {
	d := NewDeviceDataSource()
	resp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, resp)

	// id, site_id + 12 computed fields = 14
	expectedCount := 14
	if len(resp.Schema.Attributes) != expectedCount {
		t.Errorf("expected %d attributes, got %d", expectedCount, len(resp.Schema.Attributes))
	}
}

func TestDeviceDataSourceConfigure(t *testing.T) {
	d := NewDeviceDataSource().(*DeviceDataSource)

	resp := &datasource.ConfigureResponse{}
	d.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: nil}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected error for nil: %v", resp.Diagnostics)
	}

	resp2 := &datasource.ConfigureResponse{}
	d.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: 99}, resp2)
	if !resp2.Diagnostics.HasError() {
		t.Error("expected error for wrong provider data type")
	}
}

// --- Sites data source ---

func TestSitesDataSourceMetadata(t *testing.T) {
	d := NewSitesDataSource()
	resp := &datasource.MetadataResponse{}
	d.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "unifi"}, resp)

	if resp.TypeName != "unifi_sites" {
		t.Errorf("expected type name 'unifi_sites', got %q", resp.TypeName)
	}
}

func TestSitesDataSourceSchema(t *testing.T) {
	d := NewSitesDataSource()
	resp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	sitesAttr, ok := resp.Schema.Attributes["sites"]
	if !ok {
		t.Fatal("expected 'sites' attribute")
	}
	if !sitesAttr.IsComputed() {
		t.Error("expected 'sites' to be computed")
	}
}

func TestSitesDataSourceConfigure(t *testing.T) {
	d := NewSitesDataSource().(*SitesDataSource)

	resp := &datasource.ConfigureResponse{}
	d.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: nil}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected error for nil: %v", resp.Diagnostics)
	}

	resp2 := &datasource.ConfigureResponse{}
	d.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: "bad"}, resp2)
	if !resp2.Diagnostics.HasError() {
		t.Error("expected error for wrong provider data type")
	}
}

// --- Clients data source ---

func TestClientsDataSourceMetadata(t *testing.T) {
	d := NewClientsDataSource()
	resp := &datasource.MetadataResponse{}
	d.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "unifi"}, resp)

	if resp.TypeName != "unifi_clients" {
		t.Errorf("expected type name 'unifi_clients', got %q", resp.TypeName)
	}
}

func TestClientsDataSourceSchema(t *testing.T) {
	d := NewClientsDataSource()
	resp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	siteAttr, ok := resp.Schema.Attributes["site_id"]
	if !ok {
		t.Fatal("expected 'site_id' attribute")
	}
	if !siteAttr.IsRequired() {
		t.Error("expected 'site_id' to be required")
	}

	clientsAttr, ok := resp.Schema.Attributes["clients"]
	if !ok {
		t.Fatal("expected 'clients' attribute")
	}
	if !clientsAttr.IsComputed() {
		t.Error("expected 'clients' to be computed")
	}
}

// --- WANs data source ---

func TestWansDataSourceMetadata(t *testing.T) {
	d := NewWansDataSource()
	resp := &datasource.MetadataResponse{}
	d.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "unifi"}, resp)

	if resp.TypeName != "unifi_wans" {
		t.Errorf("expected type name 'unifi_wans', got %q", resp.TypeName)
	}
}

func TestWansDataSourceSchema(t *testing.T) {
	d := NewWansDataSource()
	resp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	siteAttr, ok := resp.Schema.Attributes["site_id"]
	if !ok {
		t.Fatal("expected 'site_id' attribute")
	}
	if !siteAttr.IsRequired() {
		t.Error("expected 'site_id' to be required")
	}

	wansAttr, ok := resp.Schema.Attributes["wans"]
	if !ok {
		t.Fatal("expected 'wans' attribute")
	}
	if !wansAttr.IsComputed() {
		t.Error("expected 'wans' to be computed")
	}
}

// --- VPN Servers data source ---

func TestVpnServersDataSourceMetadata(t *testing.T) {
	d := NewVpnServersDataSource()
	resp := &datasource.MetadataResponse{}
	d.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "unifi"}, resp)

	if resp.TypeName != "unifi_vpn_servers" {
		t.Errorf("expected type name 'unifi_vpn_servers', got %q", resp.TypeName)
	}
}

func TestVpnServersDataSourceSchema(t *testing.T) {
	d := NewVpnServersDataSource()
	resp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	siteAttr, ok := resp.Schema.Attributes["site_id"]
	if !ok {
		t.Fatal("expected 'site_id' attribute")
	}
	if !siteAttr.IsRequired() {
		t.Error("expected 'site_id' to be required")
	}

	vpnAttr, ok := resp.Schema.Attributes["vpn_servers"]
	if !ok {
		t.Fatal("expected 'vpn_servers' attribute")
	}
	if !vpnAttr.IsComputed() {
		t.Error("expected 'vpn_servers' to be computed")
	}
}

// --- VPN Tunnels data source ---

func TestVpnTunnelsDataSourceMetadata(t *testing.T) {
	d := NewVpnTunnelsDataSource()
	resp := &datasource.MetadataResponse{}
	d.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "unifi"}, resp)

	if resp.TypeName != "unifi_vpn_tunnels" {
		t.Errorf("expected type name 'unifi_vpn_tunnels', got %q", resp.TypeName)
	}
}

func TestVpnTunnelsDataSourceSchema(t *testing.T) {
	d := NewVpnTunnelsDataSource()
	resp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	siteAttr, ok := resp.Schema.Attributes["site_id"]
	if !ok {
		t.Fatal("expected 'site_id' attribute")
	}
	if !siteAttr.IsRequired() {
		t.Error("expected 'site_id' to be required")
	}

	tunnelsAttr, ok := resp.Schema.Attributes["vpn_tunnels"]
	if !ok {
		t.Fatal("expected 'vpn_tunnels' attribute")
	}
	if !tunnelsAttr.IsComputed() {
		t.Error("expected 'vpn_tunnels' to be computed")
	}
}

// --- RADIUS Profiles data source ---

func TestRadiusProfilesDataSourceMetadata(t *testing.T) {
	d := NewRadiusProfilesDataSource()
	resp := &datasource.MetadataResponse{}
	d.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "unifi"}, resp)

	if resp.TypeName != "unifi_radius_profiles" {
		t.Errorf("expected type name 'unifi_radius_profiles', got %q", resp.TypeName)
	}
}

func TestRadiusProfilesDataSourceSchema(t *testing.T) {
	d := NewRadiusProfilesDataSource()
	resp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	siteAttr, ok := resp.Schema.Attributes["site_id"]
	if !ok {
		t.Fatal("expected 'site_id' attribute")
	}
	if !siteAttr.IsRequired() {
		t.Error("expected 'site_id' to be required")
	}

	profilesAttr, ok := resp.Schema.Attributes["radius_profiles"]
	if !ok {
		t.Fatal("expected 'radius_profiles' attribute")
	}
	if !profilesAttr.IsComputed() {
		t.Error("expected 'radius_profiles' to be computed")
	}
}

// --- Device Tags data source ---

func TestDeviceTagsDataSourceMetadata(t *testing.T) {
	d := NewDeviceTagsDataSource()
	resp := &datasource.MetadataResponse{}
	d.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "unifi"}, resp)

	if resp.TypeName != "unifi_device_tags" {
		t.Errorf("expected type name 'unifi_device_tags', got %q", resp.TypeName)
	}
}

func TestDeviceTagsDataSourceSchema(t *testing.T) {
	d := NewDeviceTagsDataSource()
	resp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	siteAttr, ok := resp.Schema.Attributes["site_id"]
	if !ok {
		t.Fatal("expected 'site_id' attribute")
	}
	if !siteAttr.IsRequired() {
		t.Error("expected 'site_id' to be required")
	}

	tagsAttr, ok := resp.Schema.Attributes["device_tags"]
	if !ok {
		t.Fatal("expected 'device_tags' attribute")
	}
	if !tagsAttr.IsComputed() {
		t.Error("expected 'device_tags' to be computed")
	}
}

// --- Pending Devices data source ---

func TestPendingDevicesDataSourceMetadata(t *testing.T) {
	d := NewPendingDevicesDataSource()
	resp := &datasource.MetadataResponse{}
	d.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "unifi"}, resp)

	if resp.TypeName != "unifi_pending_devices" {
		t.Errorf("expected type name 'unifi_pending_devices', got %q", resp.TypeName)
	}
}

func TestPendingDevicesDataSourceSchema(t *testing.T) {
	d := NewPendingDevicesDataSource()
	resp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	devicesAttr, ok := resp.Schema.Attributes["pending_devices"]
	if !ok {
		t.Fatal("expected 'pending_devices' attribute")
	}
	if !devicesAttr.IsComputed() {
		t.Error("expected 'pending_devices' to be computed")
	}
}

// --- DPI Categories data source ---

func TestDpiCategoriesDataSourceMetadata(t *testing.T) {
	d := NewDpiCategoriesDataSource()
	resp := &datasource.MetadataResponse{}
	d.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "unifi"}, resp)

	if resp.TypeName != "unifi_dpi_categories" {
		t.Errorf("expected type name 'unifi_dpi_categories', got %q", resp.TypeName)
	}
}

func TestDpiCategoriesDataSourceSchema(t *testing.T) {
	d := NewDpiCategoriesDataSource()
	resp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	categoriesAttr, ok := resp.Schema.Attributes["categories"]
	if !ok {
		t.Fatal("expected 'categories' attribute")
	}
	if !categoriesAttr.IsComputed() {
		t.Error("expected 'categories' to be computed")
	}
}

// --- DPI Applications data source ---

func TestDpiApplicationsDataSourceMetadata(t *testing.T) {
	d := NewDpiApplicationsDataSource()
	resp := &datasource.MetadataResponse{}
	d.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "unifi"}, resp)

	if resp.TypeName != "unifi_dpi_applications" {
		t.Errorf("expected type name 'unifi_dpi_applications', got %q", resp.TypeName)
	}
}

func TestDpiApplicationsDataSourceSchema(t *testing.T) {
	d := NewDpiApplicationsDataSource()
	resp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	appsAttr, ok := resp.Schema.Attributes["applications"]
	if !ok {
		t.Fatal("expected 'applications' attribute")
	}
	if !appsAttr.IsComputed() {
		t.Error("expected 'applications' to be computed")
	}
}
