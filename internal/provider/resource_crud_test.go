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

// networkTFValue builds a tftypes.Value for a network resource/data source model.
// Pass nil or a map of overrides to customize specific attributes.
func networkTFValue(t *testing.T, attrs map[string]tftypes.Value) tftypes.Value {
	t.Helper()

	defaults := map[string]tftypes.Value{
		"id":                               tftypes.NewValue(tftypes.String, nil),
		"site_id":                          tftypes.NewValue(tftypes.String, "site-1"),
		"name":                             tftypes.NewValue(tftypes.String, "Test Net"),
		"management":                       tftypes.NewValue(tftypes.String, client.ManagementUnmanaged),
		"enabled":                          tftypes.NewValue(tftypes.Bool, true),
		"vlan_id":                          tftypes.NewValue(tftypes.Number, 100),
		"trusted_dhcp_server_ip_addresses": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
	}
	for k, v := range attrs {
		defaults[k] = v
	}

	return tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":                               tftypes.String,
			"site_id":                          tftypes.String,
			"name":                             tftypes.String,
			"management":                       tftypes.String,
			"enabled":                          tftypes.Bool,
			"vlan_id":                          tftypes.Number,
			"trusted_dhcp_server_ip_addresses": tftypes.List{ElementType: tftypes.String},
		},
	}, defaults)
}

// firewallZoneTFValue builds a tftypes.Value for a firewall zone resource/data source model.
// Pass nil or a map of overrides to customize specific attributes.
func firewallZoneTFValue(t *testing.T, attrs map[string]tftypes.Value) tftypes.Value {
	t.Helper()

	defaults := map[string]tftypes.Value{
		"id":          tftypes.NewValue(tftypes.String, nil),
		"site_id":     tftypes.NewValue(tftypes.String, "site-1"),
		"name":        tftypes.NewValue(tftypes.String, "Test Zone"),
		"network_ids": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "net-1")}),
	}
	for k, v := range attrs {
		defaults[k] = v
	}

	return tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":          tftypes.String,
			"site_id":     tftypes.String,
			"name":        tftypes.String,
			"network_ids": tftypes.List{ElementType: tftypes.String},
		},
	}, defaults)
}

// stringListVal builds a tftypes list of strings.
func stringListVal(vals ...string) tftypes.Value {
	elems := make([]tftypes.Value, len(vals))
	for i, v := range vals {
		elems[i] = tftypes.NewValue(tftypes.String, v)
	}
	return tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, elems)
}

// resourceSchemaFor extracts the schema from any resource.Resource implementation.
func resourceSchemaFor(t *testing.T, r resource.Resource) resource.SchemaResponse {
	t.Helper()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema errors: %v", resp.Diagnostics)
	}
	return *resp
}

// dsSchemaFor extracts the schema from any datasource.DataSource implementation.
func dsSchemaFor(t *testing.T, ds datasource.DataSource) datasource.SchemaResponse {
	t.Helper()
	resp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), datasource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema errors: %v", resp.Diagnostics)
	}
	return *resp
}

// newTestProvider creates a mock HTTP server and returns a configured client.
// The server is automatically closed when the test completes.
func newTestProvider(t *testing.T, handler http.HandlerFunc) *client.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c := client.NewClient("key", "host")
	c.SetBaseURL(server.URL)
	return c
}

// apiErrorHandler returns an HTTP handler that responds with the given API error.
func apiErrorHandler(statusCode int, statusName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(client.APIError{StatusCode: statusCode, StatusName: statusName})
	}
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
	c := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(client.Network{
			ID:         "net-new",
			Name:       "Test Net",
			Management: client.ManagementUnmanaged,
			Enabled:    true,
			VlanID:     100,
		})
	})
	r := &NetworkResource{client: c}

	schemaResp := resourceSchemaFor(t, NewNetworkResource())
	planVal := networkTFValue(t, nil)

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Create(context.Background(), resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planVal},
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestNetworkResourceRead(t *testing.T) {
	c := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.Network{
			ID:         "net-123",
			Name:       "Test Net",
			Management: client.ManagementGateway,
			Enabled:    true,
			VlanID:     100,
		})
	})
	r := &NetworkResource{client: c}

	schemaResp := resourceSchemaFor(t, NewNetworkResource())
	stateVal := networkTFValue(t, map[string]tftypes.Value{
		"id":         tftypes.NewValue(tftypes.String, "net-123"),
		"management": tftypes.NewValue(tftypes.String, client.ManagementGateway),
	})

	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal}}
	r.Read(context.Background(), resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal},
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestNetworkResourceReadNotFound(t *testing.T) {
	c := newTestProvider(t, apiErrorHandler(http.StatusNotFound, "NOT_FOUND"))
	r := &NetworkResource{client: c}

	schemaResp := resourceSchemaFor(t, NewNetworkResource())
	stateVal := networkTFValue(t, map[string]tftypes.Value{
		"id":   tftypes.NewValue(tftypes.String, "net-gone"),
		"name": tftypes.NewValue(tftypes.String, "Gone"),
	})

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
	c := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.Network{
			ID:         "net-123",
			Name:       "Updated Net",
			Management: client.ManagementGateway,
			Enabled:    false,
			VlanID:     200,
		})
	})
	r := &NetworkResource{client: c}

	schemaResp := resourceSchemaFor(t, NewNetworkResource())
	stateVal := networkTFValue(t, map[string]tftypes.Value{
		"id":   tftypes.NewValue(tftypes.String, "net-123"),
		"name": tftypes.NewValue(tftypes.String, "Old Net"),
	})
	planVal := networkTFValue(t, map[string]tftypes.Value{
		"id":         tftypes.NewValue(tftypes.String, "net-123"),
		"name":       tftypes.NewValue(tftypes.String, "Updated Net"),
		"management": tftypes.NewValue(tftypes.String, client.ManagementGateway),
		"enabled":    tftypes.NewValue(tftypes.Bool, false),
		"vlan_id":    tftypes.NewValue(tftypes.Number, 200),
	})

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
	c := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r := &NetworkResource{client: c}

	schemaResp := resourceSchemaFor(t, NewNetworkResource())
	stateVal := networkTFValue(t, map[string]tftypes.Value{
		"id":   tftypes.NewValue(tftypes.String, "net-123"),
		"name": tftypes.NewValue(tftypes.String, "Del Net"),
	})

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
	schemaResp := resourceSchemaFor(t, NewNetworkResource())

	stateVal := networkTFValue(t, map[string]tftypes.Value{
		"site_id":    tftypes.NewValue(tftypes.String, ""),
		"name":       tftypes.NewValue(tftypes.String, ""),
		"management": tftypes.NewValue(tftypes.String, ""),
		"enabled":    tftypes.NewValue(tftypes.Bool, false),
		"vlan_id":    tftypes.NewValue(tftypes.Number, 0),
	})
	resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal}}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: "site-1/net-123"}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestNetworkResourceImportStateInvalid(t *testing.T) {
	r := &NetworkResource{}
	schemaResp := resourceSchemaFor(t, NewNetworkResource())

	stateVal := networkTFValue(t, map[string]tftypes.Value{
		"site_id":    tftypes.NewValue(tftypes.String, ""),
		"name":       tftypes.NewValue(tftypes.String, ""),
		"management": tftypes.NewValue(tftypes.String, ""),
		"enabled":    tftypes.NewValue(tftypes.Bool, false),
		"vlan_id":    tftypes.NewValue(tftypes.Number, 0),
	})
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
	c := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(client.FirewallZone{
			ID:         "zone-new",
			Name:       "My Zone",
			NetworkIDs: []string{"net-1"},
		})
	})
	r := &FirewallZoneResource{client: c}

	schemaResp := resourceSchemaFor(t, NewFirewallZoneResource())
	planVal := firewallZoneTFValue(t, map[string]tftypes.Value{
		"name": tftypes.NewValue(tftypes.String, "My Zone"),
	})

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Create(context.Background(), resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planVal},
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestFirewallZoneResourceRead(t *testing.T) {
	c := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.FirewallZone{
			ID:         "zone-123",
			Name:       "My Zone",
			NetworkIDs: []string{"net-1", "net-2"},
		})
	})
	r := &FirewallZoneResource{client: c}

	schemaResp := resourceSchemaFor(t, NewFirewallZoneResource())
	stateVal := firewallZoneTFValue(t, map[string]tftypes.Value{
		"id":          tftypes.NewValue(tftypes.String, "zone-123"),
		"name":        tftypes.NewValue(tftypes.String, "My Zone"),
		"network_ids": stringListVal("net-1", "net-2"),
	})

	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal}}
	r.Read(context.Background(), resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal},
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestFirewallZoneResourceReadNotFound(t *testing.T) {
	c := newTestProvider(t, apiErrorHandler(http.StatusNotFound, "NOT_FOUND"))
	r := &FirewallZoneResource{client: c}

	schemaResp := resourceSchemaFor(t, NewFirewallZoneResource())
	stateVal := firewallZoneTFValue(t, map[string]tftypes.Value{
		"id":          tftypes.NewValue(tftypes.String, "zone-gone"),
		"name":        tftypes.NewValue(tftypes.String, "Gone"),
		"network_ids": stringListVal(),
	})

	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal}}
	r.Read(context.Background(), resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal},
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestFirewallZoneResourceUpdate(t *testing.T) {
	c := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.FirewallZone{
			ID:         "zone-123",
			Name:       "Updated Zone",
			NetworkIDs: []string{"net-3"},
		})
	})
	r := &FirewallZoneResource{client: c}

	schemaResp := resourceSchemaFor(t, NewFirewallZoneResource())
	stateVal := firewallZoneTFValue(t, map[string]tftypes.Value{
		"id":   tftypes.NewValue(tftypes.String, "zone-123"),
		"name": tftypes.NewValue(tftypes.String, "Old Zone"),
	})
	planVal := firewallZoneTFValue(t, map[string]tftypes.Value{
		"id":          tftypes.NewValue(tftypes.String, "zone-123"),
		"name":        tftypes.NewValue(tftypes.String, "Updated Zone"),
		"network_ids": stringListVal("net-3"),
	})

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
	c := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r := &FirewallZoneResource{client: c}

	schemaResp := resourceSchemaFor(t, NewFirewallZoneResource())
	stateVal := firewallZoneTFValue(t, map[string]tftypes.Value{
		"id":   tftypes.NewValue(tftypes.String, "zone-123"),
		"name": tftypes.NewValue(tftypes.String, "Del Zone"),
	})

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
	schemaResp := resourceSchemaFor(t, NewFirewallZoneResource())

	stateVal := firewallZoneTFValue(t, map[string]tftypes.Value{
		"site_id":     tftypes.NewValue(tftypes.String, ""),
		"name":        tftypes.NewValue(tftypes.String, ""),
		"network_ids": stringListVal(),
	})
	resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal}}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: "site-1/zone-123"}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

// --- WiFi Broadcast Resource CRUD tests ---

func wifiBroadcastTFValue(t *testing.T, attrs map[string]tftypes.Value) tftypes.Value {
	t.Helper()

	defaults := map[string]tftypes.Value{
		"id":                                      tftypes.NewValue(tftypes.String, nil),
		"site_id":                                 tftypes.NewValue(tftypes.String, "site-1"),
		"type":                                    tftypes.NewValue(tftypes.String, client.BroadcastTypeStandard),
		"name":                                    tftypes.NewValue(tftypes.String, "Test WiFi"),
		"enabled":                                 tftypes.NewValue(tftypes.Bool, true),
		"security_type":                           tftypes.NewValue(tftypes.String, client.SecurityOpen),
		"passphrase":                              tftypes.NewValue(tftypes.String, nil),
		"passphrase_wo":                           tftypes.NewValue(tftypes.String, nil),
		"passphrase_wo_version":                   tftypes.NewValue(tftypes.Number, nil),
		"network_type":                            tftypes.NewValue(tftypes.String, client.NetworkTypeNative),
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
	c := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
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
	})
	r := &WifiBroadcastResource{client: c}

	schemaResp := resourceSchemaFor(t, NewWifiBroadcastResource())
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
	c := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.WifiBroadcast{
			ID:      "wifi-123",
			Type:    client.BroadcastTypeStandard,
			Name:    "Test WiFi",
			Enabled: true,
			SecurityConfiguration: &client.SecurityConfiguration{Type: client.SecurityOpen},
			Network:               &client.BroadcastNetwork{Type: client.NetworkTypeNative},
		})
	})
	r := &WifiBroadcastResource{client: c}

	schemaResp := resourceSchemaFor(t, NewWifiBroadcastResource())
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
	c := newTestProvider(t, apiErrorHandler(http.StatusNotFound, "NOT_FOUND"))
	r := &WifiBroadcastResource{client: c}

	schemaResp := resourceSchemaFor(t, NewWifiBroadcastResource())
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
	c := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.WifiBroadcast{
			ID:      "wifi-123",
			Type:    client.BroadcastTypeStandard,
			Name:    "Updated WiFi",
			Enabled: true,
			SecurityConfiguration: &client.SecurityConfiguration{Type: client.SecurityOpen},
			Network:               &client.BroadcastNetwork{Type: client.NetworkTypeNative},
		})
	})
	r := &WifiBroadcastResource{client: c}

	schemaResp := resourceSchemaFor(t, NewWifiBroadcastResource())
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
	c := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r := &WifiBroadcastResource{client: c}

	schemaResp := resourceSchemaFor(t, NewWifiBroadcastResource())
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
	schemaResp := resourceSchemaFor(t, NewWifiBroadcastResource())

	stateVal := wifiBroadcastTFValue(t, nil)
	resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal}}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: "site-1/wifi-123"}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

// --- Error path tests ---

func TestNetworkResourceCreateError(t *testing.T) {
	c := newTestProvider(t, apiErrorHandler(http.StatusBadRequest, "BAD_REQUEST"))
	r := &NetworkResource{client: c}

	schemaResp := resourceSchemaFor(t, NewNetworkResource())
	planVal := networkTFValue(t, map[string]tftypes.Value{
		"name": tftypes.NewValue(tftypes.String, "Bad Net"),
	})

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Create(context.Background(), resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planVal},
	}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error from API")
	}
}

func TestNetworkResourceReadError(t *testing.T) {
	c := newTestProvider(t, apiErrorHandler(http.StatusInternalServerError, "INTERNAL"))
	r := &NetworkResource{client: c}

	schemaResp := resourceSchemaFor(t, NewNetworkResource())
	stateVal := networkTFValue(t, map[string]tftypes.Value{
		"id":   tftypes.NewValue(tftypes.String, "net-123"),
		"name": tftypes.NewValue(tftypes.String, "Net"),
	})

	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal}}
	r.Read(context.Background(), resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal},
	}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error from API")
	}
}

func TestNetworkResourceUpdateError(t *testing.T) {
	c := newTestProvider(t, apiErrorHandler(http.StatusBadRequest, "BAD_REQUEST"))
	r := &NetworkResource{client: c}

	schemaResp := resourceSchemaFor(t, NewNetworkResource())
	stateVal := networkTFValue(t, map[string]tftypes.Value{
		"id":   tftypes.NewValue(tftypes.String, "net-123"),
		"name": tftypes.NewValue(tftypes.String, "Old"),
	})
	planVal := networkTFValue(t, map[string]tftypes.Value{
		"id":         tftypes.NewValue(tftypes.String, "net-123"),
		"name":       tftypes.NewValue(tftypes.String, "New"),
		"management": tftypes.NewValue(tftypes.String, client.ManagementGateway),
		"vlan_id":    tftypes.NewValue(tftypes.Number, 200),
	})

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
	c := newTestProvider(t, apiErrorHandler(http.StatusForbidden, "FORBIDDEN"))
	r := &NetworkResource{client: c}

	schemaResp := resourceSchemaFor(t, NewNetworkResource())
	stateVal := networkTFValue(t, map[string]tftypes.Value{
		"id":   tftypes.NewValue(tftypes.String, "net-123"),
		"name": tftypes.NewValue(tftypes.String, "Net"),
	})

	resp := &resource.DeleteResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Delete(context.Background(), resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal},
	}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error from API")
	}
}

func TestFirewallZoneResourceCreateError(t *testing.T) {
	c := newTestProvider(t, apiErrorHandler(http.StatusBadRequest, "BAD_REQUEST"))
	r := &FirewallZoneResource{client: c}

	schemaResp := resourceSchemaFor(t, NewFirewallZoneResource())
	planVal := firewallZoneTFValue(t, map[string]tftypes.Value{
		"name": tftypes.NewValue(tftypes.String, "Zone"),
	})

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Create(context.Background(), resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planVal},
	}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error from API")
	}
}

func TestFirewallZoneResourceDeleteError(t *testing.T) {
	c := newTestProvider(t, apiErrorHandler(http.StatusForbidden, "FORBIDDEN"))
	r := &FirewallZoneResource{client: c}

	schemaResp := resourceSchemaFor(t, NewFirewallZoneResource())
	stateVal := firewallZoneTFValue(t, map[string]tftypes.Value{
		"id":   tftypes.NewValue(tftypes.String, "zone-123"),
		"name": tftypes.NewValue(tftypes.String, "Zone"),
	})

	resp := &resource.DeleteResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Delete(context.Background(), resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateVal},
	}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error from API")
	}
}

func TestWifiBroadcastResourceCreateError(t *testing.T) {
	c := newTestProvider(t, apiErrorHandler(http.StatusBadRequest, "BAD_REQUEST"))
	r := &WifiBroadcastResource{client: c}

	schemaResp := resourceSchemaFor(t, NewWifiBroadcastResource())
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
	c := newTestProvider(t, apiErrorHandler(http.StatusForbidden, "FORBIDDEN"))
	r := &WifiBroadcastResource{client: c}

	schemaResp := resourceSchemaFor(t, NewWifiBroadcastResource())
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
	c := newTestProvider(t, apiErrorHandler(http.StatusNotFound, "NOT_FOUND"))
	d := &NetworkDataSource{client: c}

	ds := NewNetworkDataSource()
	schemaResp := dsSchemaFor(t, ds)

	configVal := networkTFValue(t, map[string]tftypes.Value{
		"id": tftypes.NewValue(tftypes.String, "net-missing"),
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
	c := newTestProvider(t, apiErrorHandler(http.StatusInternalServerError, "INTERNAL"))
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
	c := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.Network{
			ID:         "net-123",
			Name:       "Test Net",
			Management: client.ManagementGateway,
			Enabled:    true,
			VlanID:     100,
		})
	})
	d := &NetworkDataSource{client: c}

	ds := NewNetworkDataSource()
	schemaResp := dsSchemaFor(t, ds)

	configVal := networkTFValue(t, map[string]tftypes.Value{
		"id": tftypes.NewValue(tftypes.String, "net-123"),
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
	c := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.FirewallZone{
			ID:         "zone-123",
			Name:       "My Zone",
			NetworkIDs: []string{"net-1"},
		})
	})
	d := &FirewallZoneDataSource{client: c}

	ds := NewFirewallZoneDataSource()
	schemaResp := dsSchemaFor(t, ds)

	configVal := firewallZoneTFValue(t, map[string]tftypes.Value{
		"id":          tftypes.NewValue(tftypes.String, "zone-123"),
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
	c := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.Device{
			ID:                "dev-123",
			MacAddress:        "aa:bb:cc:dd:ee:ff",
			IPAddress:         "192.168.1.1",
			Name:              "Switch",
			Model:             "USW-24",
			Supported:         true,
			State:             "ONLINE",
			FirmwareVersion:   "7.0.0",
			FirmwareUpdatable: true,
			AdoptedAt:         "2024-01-01",
			ProvisionedAt:     "2024-01-02",
			ConfigurationID:   "cfg-1",
			Uplink:            &client.DeviceUplink{DeviceID: "dev-parent"},
		})
	})
	d := &DeviceDataSource{client: c}

	ds := NewDeviceDataSource()
	schemaResp := dsSchemaFor(t, ds)

	configVal := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":                 tftypes.String,
			"site_id":            tftypes.String,
			"mac_address":        tftypes.String,
			"ip_address":         tftypes.String,
			"name":               tftypes.String,
			"model":              tftypes.String,
			"supported":          tftypes.Bool,
			"state":              tftypes.String,
			"firmware_version":   tftypes.String,
			"firmware_updatable": tftypes.Bool,
			"adopted_at":         tftypes.String,
			"provisioned_at":     tftypes.String,
			"configuration_id":   tftypes.String,
			"uplink_device_id":   tftypes.String,
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
	c := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.PaginatedResponse[client.Site]{
			Data: []client.Site{
				{ID: "site-1", Name: "Default", InternalReference: "default"},
				{ID: "site-2", Name: "Remote", InternalReference: "remote"},
			},
			TotalCount: 2,
			Count:      2,
		})
	})
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
	c := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.WifiBroadcast{
			ID:      "wifi-123",
			Type:    client.BroadcastTypeStandard,
			Name:    "TestSSID",
			Enabled: true,
			SecurityConfiguration: &client.SecurityConfiguration{Type: client.SecurityWPA2Personal},
			Network:               &client.BroadcastNetwork{Type: client.NetworkTypeNative},
		})
	})
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
