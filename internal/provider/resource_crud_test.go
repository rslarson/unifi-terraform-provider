package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/rslarson/terraform-provider-unifi/internal/client"
)

// --- Test helpers ---

// networkTFValue builds a tftypes.Value for a network resource model.
// id can be a string or nil (for computed/unknown).
func networkTFValue(t *testing.T, id interface{}, siteID, name, management string, enabled bool, vlanID int, dhcpIPs interface{}) tftypes.Value {
	t.Helper()

	var dhcpVal tftypes.Value
	if dhcpIPs == nil {
		dhcpVal = tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil)
	} else {
		ips := dhcpIPs.([]string)
		vals := make([]tftypes.Value, len(ips))
		for i, ip := range ips {
			vals[i] = tftypes.NewValue(tftypes.String, ip)
		}
		dhcpVal = tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, vals)
	}

	return tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":                                 tftypes.String,
			"site_id":                            tftypes.String,
			"name":                               tftypes.String,
			"management":                         tftypes.String,
			"enabled":                            tftypes.Bool,
			"vlan_id":                            tftypes.Number,
			"trusted_dhcp_server_ip_addresses":   tftypes.List{ElementType: tftypes.String},
		},
	}, map[string]tftypes.Value{
		"id":                               tftypes.NewValue(tftypes.String, id),
		"site_id":                          tftypes.NewValue(tftypes.String, siteID),
		"name":                             tftypes.NewValue(tftypes.String, name),
		"management":                       tftypes.NewValue(tftypes.String, management),
		"enabled":                          tftypes.NewValue(tftypes.Bool, enabled),
		"vlan_id":                          tftypes.NewValue(tftypes.Number, vlanID),
		"trusted_dhcp_server_ip_addresses": dhcpVal,
	})
}

// firewallZoneTFValue builds a tftypes.Value for a firewall zone resource model.
// id can be a string or nil (for computed/unknown).
func firewallZoneTFValue(t *testing.T, id interface{}, siteID, name string, networkIDs []string) tftypes.Value {
	t.Helper()

	vals := make([]tftypes.Value, len(networkIDs))
	for i, nid := range networkIDs {
		vals[i] = tftypes.NewValue(tftypes.String, nid)
	}

	return tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":          tftypes.String,
			"site_id":     tftypes.String,
			"name":        tftypes.String,
			"network_ids": tftypes.List{ElementType: tftypes.String},
		},
	}, map[string]tftypes.Value{
		"id":          tftypes.NewValue(tftypes.String, id),
		"site_id":     tftypes.NewValue(tftypes.String, siteID),
		"name":        tftypes.NewValue(tftypes.String, name),
		"network_ids": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, vals),
	})
}

func networkSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	r := NewNetworkResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema errors: %v", resp.Diagnostics)
	}
	return *resp
}

func firewallZoneSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	r := NewFirewallZoneResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema errors: %v", resp.Diagnostics)
	}
	return *resp
}

// --- Network Resource CRUD tests ---

func TestNetworkResourceConfigure(t *testing.T) {
	r := &NetworkResource{}

	// Test with nil provider data (no error, no client set)
	resp := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: nil}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Test with valid client
	c := client.NewClient("key", "host")
	resp = &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: c}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
	if r.client != c {
		t.Error("expected client to be set")
	}
}

func TestNetworkResourceCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(client.Network{
			ID:         "net-new",
			Name:       "Test Net",
			Management: client.ManagementUnmanaged,
			Enabled:    true,
			VlanID:     100,
		})
	}))
	defer server.Close()

	c := client.NewClient("key", "host")
	c.SetBaseURL(server.URL)
	r := &NetworkResource{client: c}

	schemaResp := networkSchema(t)
	planVal := networkTFValue(t, nil, "site-1", "Test Net", client.ManagementUnmanaged, true, 100, nil)

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Create(context.Background(), resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planVal},
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestNetworkResourceRead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.Network{
			ID:         "net-123",
			Name:       "Test Net",
			Management: client.ManagementGateway,
			Enabled:    true,
			VlanID:     100,
		})
	}))
	defer server.Close()

	c := client.NewClient("key", "host")
	c.SetBaseURL(server.URL)
	r := &NetworkResource{client: c}

	schemaResp := networkSchema(t)
	stateVal := networkTFValue(t, "net-123", "site-1", "Test Net", client.ManagementGateway, true, 100, nil)

	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal}}
	r.Read(context.Background(), resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal},
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestNetworkResourceReadNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(client.APIError{
			StatusCode: 404,
			StatusName: "NOT_FOUND",
			Message:    "Not found",
		})
	}))
	defer server.Close()

	c := client.NewClient("key", "host")
	c.SetBaseURL(server.URL)
	r := &NetworkResource{client: c}

	schemaResp := networkSchema(t)
	stateVal := networkTFValue(t, "net-gone", "site-1", "Gone", client.ManagementUnmanaged, true, 100, nil)

	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal}}
	r.Read(context.Background(), resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal},
	}, resp)

	// Should not have errors - resource should be removed from state
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestNetworkResourceUpdate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.Network{
			ID:         "net-123",
			Name:       "Updated Net",
			Management: client.ManagementGateway,
			Enabled:    false,
			VlanID:     200,
		})
	}))
	defer server.Close()

	c := client.NewClient("key", "host")
	c.SetBaseURL(server.URL)
	r := &NetworkResource{client: c}

	schemaResp := networkSchema(t)
	stateVal := networkTFValue(t, "net-123", "site-1", "Old Net", client.ManagementUnmanaged, true, 100, nil)
	planVal := networkTFValue(t, "net-123", "site-1", "Updated Net", client.ManagementGateway, false, 200, nil)

	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Update(context.Background(), resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: planVal},
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal},
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestNetworkResourceDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := client.NewClient("key", "host")
	c.SetBaseURL(server.URL)
	r := &NetworkResource{client: c}

	schemaResp := networkSchema(t)
	stateVal := networkTFValue(t, "net-123", "site-1", "Del Net", client.ManagementUnmanaged, true, 100, nil)

	resp := &resource.DeleteResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Delete(context.Background(), resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal},
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestNetworkResourceImportState(t *testing.T) {
	r := &NetworkResource{}
	schemaResp := networkSchema(t)

	stateVal := networkTFValue(t, nil, "", "", "", false, 0, nil)
	resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal}}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: "site-1/net-123"}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestNetworkResourceImportStateInvalid(t *testing.T) {
	r := &NetworkResource{}
	schemaResp := networkSchema(t)

	stateVal := networkTFValue(t, nil, "", "", "", false, 0, nil)
	resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal}}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: "invalid"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for invalid import ID")
	}
}

// --- Firewall Zone Resource CRUD tests ---

func TestFirewallZoneResourceConfigure(t *testing.T) {
	r := &FirewallZoneResource{}
	c := client.NewClient("key", "host")
	resp := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: c}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
	if r.client != c {
		t.Error("expected client to be set")
	}
}

func TestFirewallZoneResourceCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(client.FirewallZone{
			ID:         "zone-new",
			Name:       "My Zone",
			NetworkIDs: []string{"net-1"},
		})
	}))
	defer server.Close()

	c := client.NewClient("key", "host")
	c.SetBaseURL(server.URL)
	r := &FirewallZoneResource{client: c}

	schemaResp := firewallZoneSchema(t)
	planVal := firewallZoneTFValue(t, nil, "site-1", "My Zone", []string{"net-1"})

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Create(context.Background(), resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planVal},
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestFirewallZoneResourceRead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.FirewallZone{
			ID:         "zone-123",
			Name:       "My Zone",
			NetworkIDs: []string{"net-1", "net-2"},
		})
	}))
	defer server.Close()

	c := client.NewClient("key", "host")
	c.SetBaseURL(server.URL)
	r := &FirewallZoneResource{client: c}

	schemaResp := firewallZoneSchema(t)
	stateVal := firewallZoneTFValue(t, "zone-123", "site-1", "My Zone", []string{"net-1", "net-2"})

	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal}}
	r.Read(context.Background(), resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal},
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestFirewallZoneResourceReadNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(client.APIError{StatusCode: 404, StatusName: "NOT_FOUND"})
	}))
	defer server.Close()

	c := client.NewClient("key", "host")
	c.SetBaseURL(server.URL)
	r := &FirewallZoneResource{client: c}

	schemaResp := firewallZoneSchema(t)
	stateVal := firewallZoneTFValue(t, "zone-gone", "site-1", "Gone", []string{})

	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal}}
	r.Read(context.Background(), resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal},
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestFirewallZoneResourceUpdate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.FirewallZone{
			ID:         "zone-123",
			Name:       "Updated Zone",
			NetworkIDs: []string{"net-3"},
		})
	}))
	defer server.Close()

	c := client.NewClient("key", "host")
	c.SetBaseURL(server.URL)
	r := &FirewallZoneResource{client: c}

	schemaResp := firewallZoneSchema(t)
	stateVal := firewallZoneTFValue(t, "zone-123", "site-1", "Old Zone", []string{"net-1"})
	planVal := firewallZoneTFValue(t, "zone-123", "site-1", "Updated Zone", []string{"net-3"})

	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Update(context.Background(), resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: planVal},
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal},
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestFirewallZoneResourceDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := client.NewClient("key", "host")
	c.SetBaseURL(server.URL)
	r := &FirewallZoneResource{client: c}

	schemaResp := firewallZoneSchema(t)
	stateVal := firewallZoneTFValue(t, "zone-123", "site-1", "Del Zone", []string{"net-1"})

	resp := &resource.DeleteResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Delete(context.Background(), resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal},
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestFirewallZoneResourceImportState(t *testing.T) {
	r := &FirewallZoneResource{}
	schemaResp := firewallZoneSchema(t)

	stateVal := firewallZoneTFValue(t, nil, "", "", []string{})
	resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal}}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: "site-1/zone-123"}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

// --- WiFi Broadcast Resource CRUD tests ---

func wifiBroadcastSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	r := NewWifiBroadcastResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema errors: %v", resp.Diagnostics)
	}
	return *resp
}

func wifiBroadcastTFValue(t *testing.T, attrs map[string]tftypes.Value) tftypes.Value {
	t.Helper()

	defaults := map[string]tftypes.Value{
		"id":                                      tftypes.NewValue(tftypes.String, nil),
		"site_id":                                 tftypes.NewValue(tftypes.String, "site-1"),
		"type":                                    tftypes.NewValue(tftypes.String, "STANDARD"),
		"name":                                    tftypes.NewValue(tftypes.String, "Test WiFi"),
		"enabled":                                 tftypes.NewValue(tftypes.Bool, true),
		"security_type":                           tftypes.NewValue(tftypes.String, "OPEN"),
		"passphrase":                              tftypes.NewValue(tftypes.String, nil),
		"passphrase_wo":                           tftypes.NewValue(tftypes.String, nil),
		"passphrase_wo_version":                   tftypes.NewValue(tftypes.Number, nil),
		"network_type":                            tftypes.NewValue(tftypes.String, "NATIVE"),
		"network_id":                              tftypes.NewValue(tftypes.String, nil),
		"client_isolation_enabled":                tftypes.NewValue(tftypes.Bool, false),
		"hide_name":                               tftypes.NewValue(tftypes.Bool, false),
		"multicast_to_unicast_conversion_enabled": tftypes.NewValue(tftypes.Bool, false),
		"uapsd_enabled":                           tftypes.NewValue(tftypes.Bool, false),
	}
	for k, v := range attrs {
		defaults[k] = v
	}

	return tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":                                      tftypes.String,
			"site_id":                                 tftypes.String,
			"type":                                    tftypes.String,
			"name":                                    tftypes.String,
			"enabled":                                 tftypes.Bool,
			"security_type":                           tftypes.String,
			"passphrase":                              tftypes.String,
			"passphrase_wo":                           tftypes.String,
			"passphrase_wo_version":                   tftypes.Number,
			"network_type":                            tftypes.String,
			"network_id":                              tftypes.String,
			"client_isolation_enabled":                tftypes.Bool,
			"hide_name":                               tftypes.Bool,
			"multicast_to_unicast_conversion_enabled": tftypes.Bool,
			"uapsd_enabled":                           tftypes.Bool,
		},
	}, defaults)
}

func TestWifiBroadcastResourceConfigure(t *testing.T) {
	r := &WifiBroadcastResource{}
	c := client.NewClient("key", "host")
	resp := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: c}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
	if r.client != c {
		t.Error("expected client to be set")
	}
}

func TestWifiBroadcastResourceCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(client.WifiBroadcast{
			ID:      "wifi-new",
			Type:    client.BroadcastTypeStandard,
			Name:    "Test WiFi",
			Enabled: true,
			SecurityConfiguration: &client.SecurityConfiguration{Type: client.SecurityOpen},
			Network:               &client.BroadcastNetwork{Type: client.NetworkTypeNative},
		})
	}))
	defer server.Close()

	c := client.NewClient("key", "host")
	c.SetBaseURL(server.URL)
	r := &WifiBroadcastResource{client: c}

	schemaResp := wifiBroadcastSchema(t)
	planVal := wifiBroadcastTFValue(t, nil)

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Create(context.Background(), resource.CreateRequest{
		Plan:   tfsdk.Plan{Schema: schemaResp.Schema, Raw: planVal},
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: planVal},
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestWifiBroadcastResourceRead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.WifiBroadcast{
			ID:      "wifi-123",
			Type:    client.BroadcastTypeStandard,
			Name:    "Test WiFi",
			Enabled: true,
			SecurityConfiguration: &client.SecurityConfiguration{Type: client.SecurityOpen},
			Network:               &client.BroadcastNetwork{Type: client.NetworkTypeNative},
		})
	}))
	defer server.Close()

	c := client.NewClient("key", "host")
	c.SetBaseURL(server.URL)
	r := &WifiBroadcastResource{client: c}

	schemaResp := wifiBroadcastSchema(t)
	stateVal := wifiBroadcastTFValue(t, map[string]tftypes.Value{
		"id": tftypes.NewValue(tftypes.String, "wifi-123"),
	})

	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal}}
	r.Read(context.Background(), resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal},
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestWifiBroadcastResourceReadNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(client.APIError{StatusCode: 404, StatusName: "NOT_FOUND"})
	}))
	defer server.Close()

	c := client.NewClient("key", "host")
	c.SetBaseURL(server.URL)
	r := &WifiBroadcastResource{client: c}

	schemaResp := wifiBroadcastSchema(t)
	stateVal := wifiBroadcastTFValue(t, map[string]tftypes.Value{
		"id": tftypes.NewValue(tftypes.String, "wifi-gone"),
	})

	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal}}
	r.Read(context.Background(), resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal},
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestWifiBroadcastResourceUpdate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.WifiBroadcast{
			ID:      "wifi-123",
			Type:    client.BroadcastTypeStandard,
			Name:    "Updated WiFi",
			Enabled: true,
			SecurityConfiguration: &client.SecurityConfiguration{Type: client.SecurityOpen},
			Network:               &client.BroadcastNetwork{Type: client.NetworkTypeNative},
		})
	}))
	defer server.Close()

	c := client.NewClient("key", "host")
	c.SetBaseURL(server.URL)
	r := &WifiBroadcastResource{client: c}

	schemaResp := wifiBroadcastSchema(t)
	stateVal := wifiBroadcastTFValue(t, map[string]tftypes.Value{
		"id": tftypes.NewValue(tftypes.String, "wifi-123"),
	})
	planVal := wifiBroadcastTFValue(t, map[string]tftypes.Value{
		"id":   tftypes.NewValue(tftypes.String, "wifi-123"),
		"name": tftypes.NewValue(tftypes.String, "Updated WiFi"),
	})

	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Update(context.Background(), resource.UpdateRequest{
		Plan:   tfsdk.Plan{Schema: schemaResp.Schema, Raw: planVal},
		State:  tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal},
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: planVal},
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestWifiBroadcastResourceDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := client.NewClient("key", "host")
	c.SetBaseURL(server.URL)
	r := &WifiBroadcastResource{client: c}

	schemaResp := wifiBroadcastSchema(t)
	stateVal := wifiBroadcastTFValue(t, map[string]tftypes.Value{
		"id": tftypes.NewValue(tftypes.String, "wifi-123"),
	})

	resp := &resource.DeleteResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Delete(context.Background(), resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal},
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestWifiBroadcastResourceImportState(t *testing.T) {
	r := &WifiBroadcastResource{}
	schemaResp := wifiBroadcastSchema(t)

	stateVal := wifiBroadcastTFValue(t, nil)
	resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal}}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: "site-1/wifi-123"}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

// --- Error path tests ---

func TestNetworkResourceCreateError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(client.APIError{StatusCode: 400, StatusName: "BAD_REQUEST", Message: "Invalid"})
	}))
	defer server.Close()

	c := client.NewClient("key", "host")
	c.SetBaseURL(server.URL)
	r := &NetworkResource{client: c}

	schemaResp := networkSchema(t)
	planVal := networkTFValue(t, nil, "site-1", "Bad Net", client.ManagementUnmanaged, true, 100, nil)

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Create(context.Background(), resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planVal},
	}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error from API")
	}
}

func TestNetworkResourceReadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(client.APIError{StatusCode: 500, StatusName: "INTERNAL"})
	}))
	defer server.Close()

	c := client.NewClient("key", "host")
	c.SetBaseURL(server.URL)
	r := &NetworkResource{client: c}

	schemaResp := networkSchema(t)
	stateVal := networkTFValue(t, "net-123", "site-1", "Net", client.ManagementUnmanaged, true, 100, nil)

	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal}}
	r.Read(context.Background(), resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal},
	}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error from API")
	}
}

func TestNetworkResourceUpdateError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(client.APIError{StatusCode: 400, StatusName: "BAD_REQUEST"})
	}))
	defer server.Close()

	c := client.NewClient("key", "host")
	c.SetBaseURL(server.URL)
	r := &NetworkResource{client: c}

	schemaResp := networkSchema(t)
	stateVal := networkTFValue(t, "net-123", "site-1", "Old", client.ManagementUnmanaged, true, 100, nil)
	planVal := networkTFValue(t, "net-123", "site-1", "New", client.ManagementGateway, true, 200, nil)

	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Update(context.Background(), resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: planVal},
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal},
	}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error from API")
	}
}

func TestNetworkResourceDeleteError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(client.APIError{StatusCode: 403, StatusName: "FORBIDDEN"})
	}))
	defer server.Close()

	c := client.NewClient("key", "host")
	c.SetBaseURL(server.URL)
	r := &NetworkResource{client: c}

	schemaResp := networkSchema(t)
	stateVal := networkTFValue(t, "net-123", "site-1", "Net", client.ManagementUnmanaged, true, 100, nil)

	resp := &resource.DeleteResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Delete(context.Background(), resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal},
	}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error from API")
	}
}

func TestFirewallZoneResourceCreateError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(client.APIError{StatusCode: 400, StatusName: "BAD_REQUEST"})
	}))
	defer server.Close()

	c := client.NewClient("key", "host")
	c.SetBaseURL(server.URL)
	r := &FirewallZoneResource{client: c}

	schemaResp := firewallZoneSchema(t)
	planVal := firewallZoneTFValue(t, nil, "site-1", "Zone", []string{"net-1"})

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Create(context.Background(), resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planVal},
	}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error from API")
	}
}

func TestFirewallZoneResourceDeleteError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(client.APIError{StatusCode: 403, StatusName: "FORBIDDEN"})
	}))
	defer server.Close()

	c := client.NewClient("key", "host")
	c.SetBaseURL(server.URL)
	r := &FirewallZoneResource{client: c}

	schemaResp := firewallZoneSchema(t)
	stateVal := firewallZoneTFValue(t, "zone-123", "site-1", "Zone", []string{"net-1"})

	resp := &resource.DeleteResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Delete(context.Background(), resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal},
	}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error from API")
	}
}

func TestWifiBroadcastResourceCreateError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(client.APIError{StatusCode: 400, StatusName: "BAD_REQUEST"})
	}))
	defer server.Close()

	c := client.NewClient("key", "host")
	c.SetBaseURL(server.URL)
	r := &WifiBroadcastResource{client: c}

	schemaResp := wifiBroadcastSchema(t)
	planVal := wifiBroadcastTFValue(t, nil)

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Create(context.Background(), resource.CreateRequest{
		Plan:   tfsdk.Plan{Schema: schemaResp.Schema, Raw: planVal},
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: planVal},
	}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error from API")
	}
}

func TestWifiBroadcastResourceDeleteError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(client.APIError{StatusCode: 403, StatusName: "FORBIDDEN"})
	}))
	defer server.Close()

	c := client.NewClient("key", "host")
	c.SetBaseURL(server.URL)
	r := &WifiBroadcastResource{client: c}

	schemaResp := wifiBroadcastSchema(t)
	stateVal := wifiBroadcastTFValue(t, map[string]tftypes.Value{
		"id": tftypes.NewValue(tftypes.String, "wifi-123"),
	})

	resp := &resource.DeleteResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Delete(context.Background(), resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal},
	}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error from API")
	}
}

func TestNetworkDataSourceReadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(client.APIError{StatusCode: 404, StatusName: "NOT_FOUND"})
	}))
	defer server.Close()

	c := client.NewClient("key", "host")
	c.SetBaseURL(server.URL)
	d := &NetworkDataSource{client: c}

	ds := NewNetworkDataSource()
	schemaResp := dsSchemaFor(t, ds)

	configVal := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":                               tftypes.String,
			"site_id":                          tftypes.String,
			"name":                             tftypes.String,
			"management":                       tftypes.String,
			"enabled":                          tftypes.Bool,
			"vlan_id":                          tftypes.Number,
			"trusted_dhcp_server_ip_addresses": tftypes.List{ElementType: tftypes.String},
		},
	}, map[string]tftypes.Value{
		"id":                               tftypes.NewValue(tftypes.String, "net-missing"),
		"site_id":                          tftypes.NewValue(tftypes.String, "site-1"),
		"name":                             tftypes.NewValue(tftypes.String, nil),
		"management":                       tftypes.NewValue(tftypes.String, nil),
		"enabled":                          tftypes.NewValue(tftypes.Bool, nil),
		"vlan_id":                          tftypes.NewValue(tftypes.Number, nil),
		"trusted_dhcp_server_ip_addresses": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
	})

	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	d.Read(context.Background(), datasource.ReadRequest{
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configVal},
	}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error from API")
	}
}

func TestSitesDataSourceReadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(client.APIError{StatusCode: 500, StatusName: "INTERNAL"})
	}))
	defer server.Close()

	c := client.NewClient("key", "host")
	c.SetBaseURL(server.URL)
	d := &SitesDataSource{client: c}

	ds := NewSitesDataSource()
	schemaResp := dsSchemaFor(t, ds)

	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	d.Read(context.Background(), datasource.ReadRequest{
		Config: tfsdk.Config{Schema: schemaResp.Schema},
	}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error from API")
	}
}

// --- Data Source CRUD tests ---

func dsSchemaFor(t *testing.T, ds datasource.DataSource) datasource.SchemaResponse {
	t.Helper()
	resp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), datasource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema errors: %v", resp.Diagnostics)
	}
	return *resp
}

func TestNetworkDataSourceConfigure(t *testing.T) {
	d := &NetworkDataSource{}
	c := client.NewClient("key", "host")
	resp := &datasource.ConfigureResponse{}
	d.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: c}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
	if d.client != c {
		t.Error("expected client to be set")
	}
}

func TestNetworkDataSourceRead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.Network{
			ID:         "net-123",
			Name:       "Test Net",
			Management: client.ManagementGateway,
			Enabled:    true,
			VlanID:     100,
		})
	}))
	defer server.Close()

	c := client.NewClient("key", "host")
	c.SetBaseURL(server.URL)
	d := &NetworkDataSource{client: c}

	ds := NewNetworkDataSource()
	schemaResp := dsSchemaFor(t, ds)

	configVal := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":                               tftypes.String,
			"site_id":                          tftypes.String,
			"name":                             tftypes.String,
			"management":                       tftypes.String,
			"enabled":                          tftypes.Bool,
			"vlan_id":                          tftypes.Number,
			"trusted_dhcp_server_ip_addresses": tftypes.List{ElementType: tftypes.String},
		},
	}, map[string]tftypes.Value{
		"id":                               tftypes.NewValue(tftypes.String, "net-123"),
		"site_id":                          tftypes.NewValue(tftypes.String, "site-1"),
		"name":                             tftypes.NewValue(tftypes.String, nil),
		"management":                       tftypes.NewValue(tftypes.String, nil),
		"enabled":                          tftypes.NewValue(tftypes.Bool, nil),
		"vlan_id":                          tftypes.NewValue(tftypes.Number, nil),
		"trusted_dhcp_server_ip_addresses": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
	})

	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	d.Read(context.Background(), datasource.ReadRequest{
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configVal},
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestFirewallZoneDataSourceConfigure(t *testing.T) {
	d := &FirewallZoneDataSource{}
	c := client.NewClient("key", "host")
	resp := &datasource.ConfigureResponse{}
	d.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: c}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestFirewallZoneDataSourceRead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.FirewallZone{
			ID:         "zone-123",
			Name:       "My Zone",
			NetworkIDs: []string{"net-1"},
		})
	}))
	defer server.Close()

	c := client.NewClient("key", "host")
	c.SetBaseURL(server.URL)
	d := &FirewallZoneDataSource{client: c}

	ds := NewFirewallZoneDataSource()
	schemaResp := dsSchemaFor(t, ds)

	configVal := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":          tftypes.String,
			"site_id":     tftypes.String,
			"name":        tftypes.String,
			"network_ids": tftypes.List{ElementType: tftypes.String},
		},
	}, map[string]tftypes.Value{
		"id":          tftypes.NewValue(tftypes.String, "zone-123"),
		"site_id":     tftypes.NewValue(tftypes.String, "site-1"),
		"name":        tftypes.NewValue(tftypes.String, nil),
		"network_ids": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
	})

	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	d.Read(context.Background(), datasource.ReadRequest{
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configVal},
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestDeviceDataSourceConfigure(t *testing.T) {
	d := &DeviceDataSource{}
	c := client.NewClient("key", "host")
	resp := &datasource.ConfigureResponse{}
	d.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: c}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestDeviceDataSourceRead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.Device{
			ID:              "dev-123",
			MacAddress:      "aa:bb:cc:dd:ee:ff",
			IPAddress:       "192.168.1.1",
			Name:            "Switch",
			Model:           "USW-24",
			Supported:       true,
			State:           "ONLINE",
			FirmwareVersion: "7.0.0",
			FirmwareUpdatable: true,
			AdoptedAt:       "2024-01-01",
			ProvisionedAt:   "2024-01-02",
			ConfigurationID: "cfg-1",
			Uplink:          &client.DeviceUplink{DeviceID: "dev-parent"},
		})
	}))
	defer server.Close()

	c := client.NewClient("key", "host")
	c.SetBaseURL(server.URL)
	d := &DeviceDataSource{client: c}

	ds := NewDeviceDataSource()
	schemaResp := dsSchemaFor(t, ds)

	configVal := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":                tftypes.String,
			"site_id":           tftypes.String,
			"mac_address":       tftypes.String,
			"ip_address":        tftypes.String,
			"name":              tftypes.String,
			"model":             tftypes.String,
			"supported":         tftypes.Bool,
			"state":             tftypes.String,
			"firmware_version":  tftypes.String,
			"firmware_updatable": tftypes.Bool,
			"adopted_at":        tftypes.String,
			"provisioned_at":    tftypes.String,
			"configuration_id":  tftypes.String,
			"uplink_device_id":  tftypes.String,
		},
	}, map[string]tftypes.Value{
		"id":                 tftypes.NewValue(tftypes.String, "dev-123"),
		"site_id":            tftypes.NewValue(tftypes.String, "site-1"),
		"mac_address":        tftypes.NewValue(tftypes.String, nil),
		"ip_address":         tftypes.NewValue(tftypes.String, nil),
		"name":               tftypes.NewValue(tftypes.String, nil),
		"model":              tftypes.NewValue(tftypes.String, nil),
		"supported":          tftypes.NewValue(tftypes.Bool, nil),
		"state":              tftypes.NewValue(tftypes.String, nil),
		"firmware_version":   tftypes.NewValue(tftypes.String, nil),
		"firmware_updatable": tftypes.NewValue(tftypes.Bool, nil),
		"adopted_at":         tftypes.NewValue(tftypes.String, nil),
		"provisioned_at":     tftypes.NewValue(tftypes.String, nil),
		"configuration_id":   tftypes.NewValue(tftypes.String, nil),
		"uplink_device_id":   tftypes.NewValue(tftypes.String, nil),
	})

	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	d.Read(context.Background(), datasource.ReadRequest{
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configVal},
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestSitesDataSourceConfigure(t *testing.T) {
	d := &SitesDataSource{}
	c := client.NewClient("key", "host")
	resp := &datasource.ConfigureResponse{}
	d.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: c}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestSitesDataSourceRead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.PaginatedResponse[client.Site]{
			Data: []client.Site{
				{ID: "site-1", Name: "Default", InternalReference: "default"},
				{ID: "site-2", Name: "Remote", InternalReference: "remote"},
			},
			TotalCount: 2,
			Count:      2,
		})
	}))
	defer server.Close()

	c := client.NewClient("key", "host")
	c.SetBaseURL(server.URL)
	d := &SitesDataSource{client: c}

	ds := NewSitesDataSource()
	schemaResp := dsSchemaFor(t, ds)

	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	d.Read(context.Background(), datasource.ReadRequest{
		Config: tfsdk.Config{Schema: schemaResp.Schema},
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestWifiBroadcastDataSourceConfigure(t *testing.T) {
	d := &WifiBroadcastDataSource{}
	c := client.NewClient("key", "host")
	resp := &datasource.ConfigureResponse{}
	d.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: c}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestWifiBroadcastDataSourceRead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.WifiBroadcast{
			ID:      "wifi-123",
			Type:    client.BroadcastTypeStandard,
			Name:    "TestSSID",
			Enabled: true,
			SecurityConfiguration: &client.SecurityConfiguration{Type: client.SecurityWPA2Personal},
			Network:               &client.BroadcastNetwork{Type: client.NetworkTypeNative},
		})
	}))
	defer server.Close()

	c := client.NewClient("key", "host")
	c.SetBaseURL(server.URL)
	d := &WifiBroadcastDataSource{client: c}

	ds := NewWifiBroadcastDataSource()
	schemaResp := dsSchemaFor(t, ds)

	configVal := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":                                      tftypes.String,
			"site_id":                                 tftypes.String,
			"type":                                    tftypes.String,
			"name":                                    tftypes.String,
			"enabled":                                 tftypes.Bool,
			"security_type":                           tftypes.String,
			"network_type":                            tftypes.String,
			"network_id":                              tftypes.String,
			"client_isolation_enabled":                tftypes.Bool,
			"hide_name":                               tftypes.Bool,
			"multicast_to_unicast_conversion_enabled": tftypes.Bool,
			"uapsd_enabled":                           tftypes.Bool,
		},
	}, map[string]tftypes.Value{
		"id":                                      tftypes.NewValue(tftypes.String, "wifi-123"),
		"site_id":                                 tftypes.NewValue(tftypes.String, "site-1"),
		"type":                                    tftypes.NewValue(tftypes.String, nil),
		"name":                                    tftypes.NewValue(tftypes.String, nil),
		"enabled":                                 tftypes.NewValue(tftypes.Bool, nil),
		"security_type":                           tftypes.NewValue(tftypes.String, nil),
		"network_type":                            tftypes.NewValue(tftypes.String, nil),
		"network_id":                              tftypes.NewValue(tftypes.String, nil),
		"client_isolation_enabled":                tftypes.NewValue(tftypes.Bool, nil),
		"hide_name":                               tftypes.NewValue(tftypes.Bool, nil),
		"multicast_to_unicast_conversion_enabled": tftypes.NewValue(tftypes.Bool, nil),
		"uapsd_enabled":                           tftypes.NewValue(tftypes.Bool, nil),
	})

	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	d.Read(context.Background(), datasource.ReadRequest{
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configVal},
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}
