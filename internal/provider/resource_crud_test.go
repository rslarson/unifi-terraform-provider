package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/rslarson/terraform-provider-unifi/internal/client"
)

// makePlan creates a tfsdk.Plan from a resource's schema and a map of tftypes values.
func makePlan(t *testing.T, r resource.Resource, vals map[string]tftypes.Value) tfsdk.Plan {
	t.Helper()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema error: %v", schemaResp.Diagnostics)
	}
	tfType := schemaResp.Schema.Type().TerraformType(context.Background())
	rawVal, err := schemaResp.Schema.Type().ValueFromTerraform(context.Background(),
		tftypes.NewValue(tfType, vals))
	if err != nil {
		t.Fatalf("ValueFromTerraform: %v", err)
	}
	raw, err := rawVal.ToTerraformValue(context.Background())
	if err != nil {
		t.Fatalf("ToTerraformValue: %v", err)
	}
	return tfsdk.Plan{Schema: schemaResp.Schema, Raw: raw}
}

// makeState creates a tfsdk.State from a resource's schema and a map of tftypes values.
func makeState(t *testing.T, r resource.Resource, vals map[string]tftypes.Value) tfsdk.State {
	t.Helper()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema error: %v", schemaResp.Diagnostics)
	}
	tfType := schemaResp.Schema.Type().TerraformType(context.Background())
	rawVal, err := schemaResp.Schema.Type().ValueFromTerraform(context.Background(),
		tftypes.NewValue(tfType, vals))
	if err != nil {
		t.Fatalf("ValueFromTerraform: %v", err)
	}
	raw, err := rawVal.ToTerraformValue(context.Background())
	if err != nil {
		t.Fatalf("ToTerraformValue: %v", err)
	}
	return tfsdk.State{Schema: schemaResp.Schema, Raw: raw}
}

// wifiBroadcastTestValues merges the provided overrides into a complete set of
// tftypes values for a WifiBroadcast resource, providing null defaults for all
// optional/new fields so tests don't need to list every attribute.
func wifiBroadcastTestValues(overrides map[string]tftypes.Value) map[string]tftypes.Value {
	blackoutDayObjType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"type": tftypes.String, "day": tftypes.String,
		"start_time": tftypes.String, "end_time": tftypes.String,
	}}
	defaults := map[string]tftypes.Value{
		"id":                       tftypes.NewValue(tftypes.String, nil),
		"site_id":                  tftypes.NewValue(tftypes.String, "site-1"),
		"type":                     tftypes.NewValue(tftypes.String, "STANDARD"),
		"name":                     tftypes.NewValue(tftypes.String, "Test"),
		"enabled":                  tftypes.NewValue(tftypes.Bool, true),
		"security_type":            tftypes.NewValue(tftypes.String, nil),
		"passphrase":               tftypes.NewValue(tftypes.String, nil),
		"passphrase_wo":            tftypes.NewValue(tftypes.String, nil),
		"passphrase_wo_version":    tftypes.NewValue(tftypes.Number, nil),
		"network_type":             tftypes.NewValue(tftypes.String, "NATIVE"),
		"network_id":               tftypes.NewValue(tftypes.String, nil),
		"client_isolation_enabled": tftypes.NewValue(tftypes.Bool, false),
		"hide_name":                tftypes.NewValue(tftypes.Bool, false),
		"multicast_to_unicast_conversion_enabled": tftypes.NewValue(tftypes.Bool, false),
		"uapsd_enabled":                   tftypes.NewValue(tftypes.Bool, false),
		"basic_data_rate_24ghz":           tftypes.NewValue(tftypes.Number, nil),
		"basic_data_rate_5ghz":            tftypes.NewValue(tftypes.Number, nil),
		"client_filter_action":            tftypes.NewValue(tftypes.String, nil),
		"client_filter_mac_addresses":     tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
		"blackout_schedule_days":          tftypes.NewValue(tftypes.List{ElementType: blackoutDayObjType}, nil),
		"broadcasting_frequencies_ghz":    tftypes.NewValue(tftypes.List{ElementType: tftypes.Number}, nil),
		"broadcasting_device_filter_type": tftypes.NewValue(tftypes.String, nil),
		"broadcasting_device_filter_ids":  tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
		"multicast_filter_action":         tftypes.NewValue(tftypes.String, nil),
		"mdns_proxy_mode":                 tftypes.NewValue(tftypes.String, nil),
		"band_steering_enabled":           tftypes.NewValue(tftypes.Bool, nil),
		"mlo_enabled":                     tftypes.NewValue(tftypes.Bool, nil),
		"arp_proxy_enabled":               tftypes.NewValue(tftypes.Bool, nil),
		"bss_transition_enabled":          tftypes.NewValue(tftypes.Bool, nil),
		"advertise_device_name":           tftypes.NewValue(tftypes.Bool, nil),
	}
	for k, v := range overrides {
		defaults[k] = v
	}
	return defaults
}

// networkTestValues merges the provided overrides into a complete set of
// tftypes values for a Network resource, providing null defaults for all
// optional/new fields so tests don't need to list every attribute.
func networkTestValues(overrides map[string]tftypes.Value) map[string]tftypes.Value {
	ipv4ObjType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"auto_scale_enabled":      tftypes.Bool,
		"host_ip_address":         tftypes.String,
		"prefix_length":           tftypes.Number,
		"dhcp_mode":               tftypes.String,
		"dhcp_start":              tftypes.String,
		"dhcp_stop":               tftypes.String,
		"dhcp_lease_time_seconds": tftypes.Number,
		"dhcp_dns_servers":        tftypes.List{ElementType: tftypes.String},
		"dhcp_gateway_override":   tftypes.String,
		"dhcp_domain_name":        tftypes.String,
		"dhcp_relay_addresses":    tftypes.List{ElementType: tftypes.String},
	}}
	defaults := map[string]tftypes.Value{
		"id":                               tftypes.NewValue(tftypes.String, nil),
		"site_id":                          tftypes.NewValue(tftypes.String, "site-1"),
		"name":                             tftypes.NewValue(tftypes.String, "Test Network"),
		"management":                       tftypes.NewValue(tftypes.String, client.ManagementUnmanaged),
		"enabled":                          tftypes.NewValue(tftypes.Bool, true),
		"vlan_id":                          tftypes.NewValue(tftypes.Number, 42),
		"trusted_dhcp_server_ip_addresses": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
		"isolation_enabled":                tftypes.NewValue(tftypes.Bool, nil),
		"cellular_backup_enabled":          tftypes.NewValue(tftypes.Bool, nil),
		"internet_access_enabled":          tftypes.NewValue(tftypes.Bool, nil),
		"mdns_forwarding_enabled":          tftypes.NewValue(tftypes.Bool, nil),
		"zone_id":                          tftypes.NewValue(tftypes.String, nil),
		"device_id":                        tftypes.NewValue(tftypes.String, nil),
		"ipv4_configuration":               tftypes.NewValue(ipv4ObjType, nil),
	}
	for k, v := range overrides {
		defaults[k] = v
	}
	return defaults
}

// --- Network resource CRUD ---

func TestNetworkResourceCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(client.Network{
			ID:         "net-created",
			Name:       "My Network",
			Management: client.ManagementUnmanaged,
			Enabled:    true,
			VlanID:     42,
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	r := NewNetworkResource()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)

	nr := r.(*NetworkResource)
	nr.client = client.NewClientForTesting("key", "host", server.URL)

	plan := makePlan(t, r, networkTestValues(map[string]tftypes.Value{
		"name":       tftypes.NewValue(tftypes.String, "My Network"),
		"management": tftypes.NewValue(tftypes.String, client.ManagementUnmanaged),
		"vlan_id":    tftypes.NewValue(tftypes.Number, 42),
	}))

	createResp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)

	if createResp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", createResp.Diagnostics)
	}

	var state NetworkResourceModel
	createResp.State.Get(context.Background(), &state)
	if state.ID.ValueString() != "net-created" {
		t.Errorf("expected ID 'net-created', got %q", state.ID.ValueString())
	}
	if state.Name.ValueString() != "My Network" {
		t.Errorf("expected name 'My Network', got %q", state.Name.ValueString())
	}
	if state.VlanID.ValueInt64() != 42 {
		t.Errorf("expected vlanId 42, got %d", state.VlanID.ValueInt64())
	}
}

func TestNetworkResourceRead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(client.Network{
			ID:         "net-existing",
			Name:       "Existing Network",
			Management: client.ManagementGateway,
			Enabled:    false,
			VlanID:     100,
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	r := NewNetworkResource()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)

	nr := r.(*NetworkResource)
	nr.client = client.NewClientForTesting("key", "host", server.URL)

	state := makeState(t, r, networkTestValues(map[string]tftypes.Value{
		"id":         tftypes.NewValue(tftypes.String, "net-existing"),
		"name":       tftypes.NewValue(tftypes.String, "old-name"),
		"management": tftypes.NewValue(tftypes.String, client.ManagementUnmanaged),
		"vlan_id":    tftypes.NewValue(tftypes.Number, 50),
	}))

	readResp := &resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, readResp)

	if readResp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", readResp.Diagnostics)
	}

	var result NetworkResourceModel
	readResp.State.Get(context.Background(), &result)
	if result.Name.ValueString() != "Existing Network" {
		t.Errorf("expected name 'Existing Network', got %q", result.Name.ValueString())
	}
	if result.Management.ValueString() != client.ManagementGateway {
		t.Errorf("expected management %q, got %q", client.ManagementGateway, result.Management.ValueString())
	}
}

func TestNetworkResourceReadNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(client.APIError{
			StatusCode: 404, StatusName: "NOT_FOUND",
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	r := NewNetworkResource()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)

	nr := r.(*NetworkResource)
	nr.client = client.NewClientForTesting("key", "host", server.URL)

	state := makeState(t, r, networkTestValues(map[string]tftypes.Value{
		"id":      tftypes.NewValue(tftypes.String, "net-gone"),
		"name":    tftypes.NewValue(tftypes.String, "gone"),
		"vlan_id": tftypes.NewValue(tftypes.Number, 10),
	}))

	readResp := &resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, readResp)

	if readResp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors on 404: %v", readResp.Diagnostics)
	}
	// State should be empty (resource removed)
	if !readResp.State.Raw.IsNull() {
		t.Error("expected state to be null after 404")
	}
}

func TestNetworkResourceUpdate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		var req client.Network
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		req.ID = "net-existing"
		if err := json.NewEncoder(w).Encode(req); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	r := NewNetworkResource()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)

	nr := r.(*NetworkResource)
	nr.client = client.NewClientForTesting("key", "host", server.URL)

	plan := makePlan(t, r, networkTestValues(map[string]tftypes.Value{
		"name":       tftypes.NewValue(tftypes.String, "Updated Network"),
		"management": tftypes.NewValue(tftypes.String, client.ManagementSwitch),
		"enabled":    tftypes.NewValue(tftypes.Bool, false),
		"vlan_id":    tftypes.NewValue(tftypes.Number, 200),
	}))
	state := makeState(t, r, networkTestValues(map[string]tftypes.Value{
		"id":         tftypes.NewValue(tftypes.String, "net-existing"),
		"name":       tftypes.NewValue(tftypes.String, "Old Network"),
		"management": tftypes.NewValue(tftypes.String, client.ManagementUnmanaged),
		"vlan_id":    tftypes.NewValue(tftypes.Number, 100),
	}))

	updateResp := &resource.UpdateResponse{State: state}
	r.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, updateResp)

	if updateResp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", updateResp.Diagnostics)
	}

	var result NetworkResourceModel
	updateResp.State.Get(context.Background(), &result)
	if result.Name.ValueString() != "Updated Network" {
		t.Errorf("expected name 'Updated Network', got %q", result.Name.ValueString())
	}
}

func TestNetworkResourceDelete(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	r := NewNetworkResource()

	nr := r.(*NetworkResource)
	nr.client = client.NewClientForTesting("key", "host", server.URL)

	state := makeState(t, r, networkTestValues(map[string]tftypes.Value{
		"id":      tftypes.NewValue(tftypes.String, "net-del"),
		"name":    tftypes.NewValue(tftypes.String, "To Delete"),
		"vlan_id": tftypes.NewValue(tftypes.Number, 10),
	}))

	deleteResp := &resource.DeleteResponse{}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)

	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", deleteResp.Diagnostics)
	}
	if !called {
		t.Error("expected DELETE request to be sent")
	}
}

// --- FirewallZone resource CRUD ---

func TestFirewallZoneResourceCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var req client.FirewallZone
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(client.FirewallZone{
			ID:         "zone-new",
			Name:       req.Name,
			NetworkIDs: req.NetworkIDs,
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	r := NewFirewallZoneResource()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)

	fzr := r.(*FirewallZoneResource)
	fzr.client = client.NewClientForTesting("key", "host", server.URL)

	plan := makePlan(t, r, map[string]tftypes.Value{
		"id":      tftypes.NewValue(tftypes.String, nil),
		"site_id": tftypes.NewValue(tftypes.String, "site-1"),
		"name":    tftypes.NewValue(tftypes.String, "New Zone"),
		"network_ids": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
			tftypes.NewValue(tftypes.String, "net-1"),
			tftypes.NewValue(tftypes.String, "net-2"),
		}),
	})

	createResp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)

	if createResp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", createResp.Diagnostics)
	}

	var state FirewallZoneResourceModel
	createResp.State.Get(context.Background(), &state)
	if state.ID.ValueString() != "zone-new" {
		t.Errorf("expected ID 'zone-new', got %q", state.ID.ValueString())
	}
	if state.Name.ValueString() != "New Zone" {
		t.Errorf("expected name 'New Zone', got %q", state.Name.ValueString())
	}
}

func TestFirewallZoneResourceReadNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(client.APIError{StatusCode: 404, StatusName: "NOT_FOUND"}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	r := NewFirewallZoneResource()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)

	fzr := r.(*FirewallZoneResource)
	fzr.client = client.NewClientForTesting("key", "host", server.URL)

	state := makeState(t, r, map[string]tftypes.Value{
		"id":          tftypes.NewValue(tftypes.String, "zone-gone"),
		"site_id":     tftypes.NewValue(tftypes.String, "site-1"),
		"name":        tftypes.NewValue(tftypes.String, "Gone Zone"),
		"network_ids": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{}),
	})

	readResp := &resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, readResp)

	if readResp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors on 404: %v", readResp.Diagnostics)
	}
	if !readResp.State.Raw.IsNull() {
		t.Error("expected state to be null after 404")
	}
}

func TestFirewallZoneResourceDelete(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	r := NewFirewallZoneResource()

	fzr := r.(*FirewallZoneResource)
	fzr.client = client.NewClientForTesting("key", "host", server.URL)

	state := makeState(t, r, map[string]tftypes.Value{
		"id":          tftypes.NewValue(tftypes.String, "zone-del"),
		"site_id":     tftypes.NewValue(tftypes.String, "site-1"),
		"name":        tftypes.NewValue(tftypes.String, "To Delete"),
		"network_ids": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{}),
	})

	deleteResp := &resource.DeleteResponse{}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)

	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", deleteResp.Diagnostics)
	}
	if !called {
		t.Error("expected DELETE request to be sent")
	}
}

// --- WifiBroadcast resource CRUD ---

func TestWifiBroadcastResourceCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var req client.WifiBroadcast
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(client.WifiBroadcast{
			ID:      "wifi-new",
			Type:    req.Type,
			Name:    req.Name,
			Enabled: req.Enabled,
			SecurityConfiguration: &client.SecurityConfiguration{
				Type: req.SecurityConfiguration.Type,
			},
			Network: req.Network,
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	r := NewWifiBroadcastResource()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)

	wbr := r.(*WifiBroadcastResource)
	wbr.client = client.NewClientForTesting("key", "host", server.URL)

	plan := makePlan(t, r, wifiBroadcastTestValues(map[string]tftypes.Value{
		"id":            tftypes.NewValue(tftypes.String, nil),
		"site_id":       tftypes.NewValue(tftypes.String, "site-1"),
		"type":          tftypes.NewValue(tftypes.String, client.BroadcastTypeStandard),
		"name":          tftypes.NewValue(tftypes.String, "HomeWifi"),
		"enabled":       tftypes.NewValue(tftypes.Bool, true),
		"security_type": tftypes.NewValue(tftypes.String, client.SecurityOpen),
		"network_type":  tftypes.NewValue(tftypes.String, client.NetworkTypeNative),
	}))

	createResp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}
	r.Create(context.Background(), resource.CreateRequest{
		Plan:   plan,
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: plan.Raw},
	}, createResp)

	if createResp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", createResp.Diagnostics)
	}

	var state WifiBroadcastResourceModel
	createResp.State.Get(context.Background(), &state)
	if state.ID.ValueString() != "wifi-new" {
		t.Errorf("expected ID 'wifi-new', got %q", state.ID.ValueString())
	}
	if state.Name.ValueString() != "HomeWifi" {
		t.Errorf("expected name 'HomeWifi', got %q", state.Name.ValueString())
	}
}

func TestWifiBroadcastResourceReadNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(client.APIError{StatusCode: 404, StatusName: "NOT_FOUND"}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	r := NewWifiBroadcastResource()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)

	wbr := r.(*WifiBroadcastResource)
	wbr.client = client.NewClientForTesting("key", "host", server.URL)

	state := makeState(t, r, wifiBroadcastTestValues(map[string]tftypes.Value{
		"id":            tftypes.NewValue(tftypes.String, "wifi-gone"),
		"site_id":       tftypes.NewValue(tftypes.String, "site-1"),
		"type":          tftypes.NewValue(tftypes.String, client.BroadcastTypeStandard),
		"name":          tftypes.NewValue(tftypes.String, "Gone WiFi"),
		"enabled":       tftypes.NewValue(tftypes.Bool, true),
		"security_type": tftypes.NewValue(tftypes.String, client.SecurityOpen),
		"network_type":  tftypes.NewValue(tftypes.String, client.NetworkTypeNative),
	}))

	readResp := &resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, readResp)

	if readResp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors on 404: %v", readResp.Diagnostics)
	}
	if !readResp.State.Raw.IsNull() {
		t.Error("expected state to be null after 404")
	}
}

func TestWifiBroadcastResourceDelete(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	r := NewWifiBroadcastResource()

	wbr := r.(*WifiBroadcastResource)
	wbr.client = client.NewClientForTesting("key", "host", server.URL)

	state := makeState(t, r, wifiBroadcastTestValues(map[string]tftypes.Value{
		"id":            tftypes.NewValue(tftypes.String, "wifi-del"),
		"site_id":       tftypes.NewValue(tftypes.String, "site-1"),
		"type":          tftypes.NewValue(tftypes.String, client.BroadcastTypeStandard),
		"name":          tftypes.NewValue(tftypes.String, "To Delete"),
		"enabled":       tftypes.NewValue(tftypes.Bool, true),
		"security_type": tftypes.NewValue(tftypes.String, client.SecurityOpen),
		"network_type":  tftypes.NewValue(tftypes.String, client.NetworkTypeNative),
	}))

	deleteResp := &resource.DeleteResponse{}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)

	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", deleteResp.Diagnostics)
	}
	if !called {
		t.Error("expected DELETE request to be sent")
	}
}

// --- ACL Rule resource CRUD ---

// aclRuleTestValues merges the provided overrides into a complete set of
// tftypes values for an AclRule resource.
func aclRuleTestValues(overrides map[string]tftypes.Value) map[string]tftypes.Value {
	defaults := map[string]tftypes.Value{
		"id":                        tftypes.NewValue(tftypes.String, nil),
		"site_id":                   tftypes.NewValue(tftypes.String, "site-1"),
		"type":                      tftypes.NewValue(tftypes.String, "IPV4"),
		"enabled":                   tftypes.NewValue(tftypes.Bool, true),
		"name":                      tftypes.NewValue(tftypes.String, "Test ACL"),
		"description":               tftypes.NewValue(tftypes.String, nil),
		"action":                    tftypes.NewValue(tftypes.String, "ALLOW"),
		"source_filter_type":        tftypes.NewValue(tftypes.String, nil),
		"source_filter_values":      tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
		"source_filter_ports":       tftypes.NewValue(tftypes.List{ElementType: tftypes.Number}, nil),
		"destination_filter_type":   tftypes.NewValue(tftypes.String, nil),
		"destination_filter_values": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
		"destination_filter_ports":  tftypes.NewValue(tftypes.List{ElementType: tftypes.Number}, nil),
		"protocol_filter":           tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
		"enforcing_device_ids":      tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
		"network_id_filter":         tftypes.NewValue(tftypes.String, nil),
	}
	for k, v := range overrides {
		defaults[k] = v
	}
	return defaults
}

func TestAclRuleResourceCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(client.AclRule{
			ID:      "acl-created",
			Type:    "IPV4",
			Enabled: true,
			Name:    "Block SSH",
			Action:  "BLOCK",
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	r := NewAclRuleResource()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)

	ar := r.(*AclRuleResource)
	ar.client = client.NewClientForTesting("key", "host", server.URL)

	plan := makePlan(t, r, aclRuleTestValues(map[string]tftypes.Value{
		"name":   tftypes.NewValue(tftypes.String, "Block SSH"),
		"action": tftypes.NewValue(tftypes.String, "BLOCK"),
	}))

	createResp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)

	if createResp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", createResp.Diagnostics)
	}

	var state AclRuleResourceModel
	createResp.State.Get(context.Background(), &state)
	if state.ID.ValueString() != "acl-created" {
		t.Errorf("expected ID 'acl-created', got %q", state.ID.ValueString())
	}
	if state.Name.ValueString() != "Block SSH" {
		t.Errorf("expected name 'Block SSH', got %q", state.Name.ValueString())
	}
}

func TestAclRuleResourceReadNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(client.APIError{StatusCode: 404, StatusName: "NOT_FOUND"}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	r := NewAclRuleResource()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)

	ar := r.(*AclRuleResource)
	ar.client = client.NewClientForTesting("key", "host", server.URL)

	state := makeState(t, r, aclRuleTestValues(map[string]tftypes.Value{
		"id":   tftypes.NewValue(tftypes.String, "acl-gone"),
		"name": tftypes.NewValue(tftypes.String, "Gone ACL"),
	}))

	readResp := &resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, readResp)

	if readResp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors on 404: %v", readResp.Diagnostics)
	}
	if !readResp.State.Raw.IsNull() {
		t.Error("expected state to be null after 404")
	}
}

func TestAclRuleResourceDelete(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	r := NewAclRuleResource()

	ar := r.(*AclRuleResource)
	ar.client = client.NewClientForTesting("key", "host", server.URL)

	state := makeState(t, r, aclRuleTestValues(map[string]tftypes.Value{
		"id":   tftypes.NewValue(tftypes.String, "acl-del"),
		"name": tftypes.NewValue(tftypes.String, "To Delete"),
	}))

	deleteResp := &resource.DeleteResponse{}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)

	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", deleteResp.Diagnostics)
	}
	if !called {
		t.Error("expected DELETE request to be sent")
	}
}

// --- Firewall Policy resource CRUD ---

// firewallPolicyTestValues merges the provided overrides into a complete set of
// tftypes values for a FirewallPolicy resource.
func firewallPolicyTestValues(overrides map[string]tftypes.Value) map[string]tftypes.Value {
	defaults := map[string]tftypes.Value{
		"id":                                tftypes.NewValue(tftypes.String, nil),
		"site_id":                           tftypes.NewValue(tftypes.String, "site-1"),
		"enabled":                           tftypes.NewValue(tftypes.Bool, true),
		"name":                              tftypes.NewValue(tftypes.String, "Test Policy"),
		"description":                       tftypes.NewValue(tftypes.String, nil),
		"action_type":                       tftypes.NewValue(tftypes.String, "ALLOW"),
		"allow_return_traffic":              tftypes.NewValue(tftypes.Bool, nil),
		"source_zone_id":                    tftypes.NewValue(tftypes.String, "zone-src"),
		"source_traffic_filter_type":        tftypes.NewValue(tftypes.String, nil),
		"source_traffic_filter_values":      tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
		"destination_zone_id":               tftypes.NewValue(tftypes.String, "zone-dst"),
		"destination_traffic_filter_type":   tftypes.NewValue(tftypes.String, nil),
		"destination_traffic_filter_values": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
		"ip_version":                        tftypes.NewValue(tftypes.String, "IPV4"),
		"connection_state_filter":           tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
		"ipsec_filter":                      tftypes.NewValue(tftypes.String, nil),
		"logging_enabled":                   tftypes.NewValue(tftypes.Bool, false),
	}
	for k, v := range overrides {
		defaults[k] = v
	}
	return defaults
}

func TestFirewallPolicyResourceCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(client.FirewallPolicy{
			ID:              "policy-created",
			Enabled:         true,
			Name:            "Block WAN",
			Action:          &client.FirewallAction{Type: "BLOCK"},
			Source:          &client.FirewallEndpoint{ZoneID: "zone-src"},
			Destination:     &client.FirewallEndpoint{ZoneID: "zone-dst"},
			IPProtocolScope: &client.IPProtocolScope{IPVersion: "IPV4"},
			LoggingEnabled:  true,
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	r := NewFirewallPolicyResource()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)

	fpr := r.(*FirewallPolicyResource)
	fpr.client = client.NewClientForTesting("key", "host", server.URL)

	plan := makePlan(t, r, firewallPolicyTestValues(map[string]tftypes.Value{
		"name":            tftypes.NewValue(tftypes.String, "Block WAN"),
		"action_type":     tftypes.NewValue(tftypes.String, "BLOCK"),
		"logging_enabled": tftypes.NewValue(tftypes.Bool, true),
	}))

	createResp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)

	if createResp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", createResp.Diagnostics)
	}

	var state FirewallPolicyResourceModel
	createResp.State.Get(context.Background(), &state)
	if state.ID.ValueString() != "policy-created" {
		t.Errorf("expected ID 'policy-created', got %q", state.ID.ValueString())
	}
	if state.Name.ValueString() != "Block WAN" {
		t.Errorf("expected name 'Block WAN', got %q", state.Name.ValueString())
	}
}

func TestFirewallPolicyResourceReadNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(client.APIError{StatusCode: 404, StatusName: "NOT_FOUND"}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	r := NewFirewallPolicyResource()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)

	fpr := r.(*FirewallPolicyResource)
	fpr.client = client.NewClientForTesting("key", "host", server.URL)

	state := makeState(t, r, firewallPolicyTestValues(map[string]tftypes.Value{
		"id":   tftypes.NewValue(tftypes.String, "policy-gone"),
		"name": tftypes.NewValue(tftypes.String, "Gone Policy"),
	}))

	readResp := &resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, readResp)

	if readResp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors on 404: %v", readResp.Diagnostics)
	}
	if !readResp.State.Raw.IsNull() {
		t.Error("expected state to be null after 404")
	}
}

func TestFirewallPolicyResourceDelete(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	r := NewFirewallPolicyResource()

	fpr := r.(*FirewallPolicyResource)
	fpr.client = client.NewClientForTesting("key", "host", server.URL)

	state := makeState(t, r, firewallPolicyTestValues(map[string]tftypes.Value{
		"id":   tftypes.NewValue(tftypes.String, "policy-del"),
		"name": tftypes.NewValue(tftypes.String, "To Delete"),
	}))

	deleteResp := &resource.DeleteResponse{}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)

	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", deleteResp.Diagnostics)
	}
	if !called {
		t.Error("expected DELETE request to be sent")
	}
}

// --- Traffic Matching List resource CRUD ---

// trafficMatchingListTestValues merges the provided overrides into a complete set of
// tftypes values for a TrafficMatchingList resource.
func trafficMatchingListTestValues(overrides map[string]tftypes.Value) map[string]tftypes.Value {
	itemObjType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"type":        tftypes.String,
		"value":       tftypes.String,
		"subnet":      tftypes.String,
		"start":       tftypes.String,
		"end":         tftypes.String,
		"port_number": tftypes.Number,
		"start_port":  tftypes.Number,
		"end_port":    tftypes.Number,
	}}
	defaults := map[string]tftypes.Value{
		"id":      tftypes.NewValue(tftypes.String, nil),
		"site_id": tftypes.NewValue(tftypes.String, "site-1"),
		"type":    tftypes.NewValue(tftypes.String, "IPV4_ADDRESSES"),
		"name":    tftypes.NewValue(tftypes.String, "Test List"),
		"items": tftypes.NewValue(tftypes.List{ElementType: itemObjType}, []tftypes.Value{
			tftypes.NewValue(itemObjType, map[string]tftypes.Value{
				"type":        tftypes.NewValue(tftypes.String, "IP_ADDRESS"),
				"value":       tftypes.NewValue(tftypes.String, "10.0.0.1"),
				"subnet":      tftypes.NewValue(tftypes.String, nil),
				"start":       tftypes.NewValue(tftypes.String, nil),
				"end":         tftypes.NewValue(tftypes.String, nil),
				"port_number": tftypes.NewValue(tftypes.Number, nil),
				"start_port":  tftypes.NewValue(tftypes.Number, nil),
				"end_port":    tftypes.NewValue(tftypes.Number, nil),
			}),
		}),
	}
	for k, v := range overrides {
		defaults[k] = v
	}
	return defaults
}

func TestTrafficMatchingListResourceCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(client.TrafficMatchingList{
			ID:   "tml-created",
			Type: "IPV4_ADDRESSES",
			Name: "Blocklist",
			Items: []client.TrafficMatchingItem{
				{Type: "IP_ADDRESS", Value: "10.0.0.1"},
			},
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	r := NewTrafficMatchingListResource()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)

	tmlr := r.(*TrafficMatchingListResource)
	tmlr.client = client.NewClientForTesting("key", "host", server.URL)

	plan := makePlan(t, r, trafficMatchingListTestValues(map[string]tftypes.Value{
		"name": tftypes.NewValue(tftypes.String, "Blocklist"),
	}))

	createResp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)

	if createResp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", createResp.Diagnostics)
	}

	var state TrafficMatchingListResourceModel
	createResp.State.Get(context.Background(), &state)
	if state.ID.ValueString() != "tml-created" {
		t.Errorf("expected ID 'tml-created', got %q", state.ID.ValueString())
	}
	if state.Name.ValueString() != "Blocklist" {
		t.Errorf("expected name 'Blocklist', got %q", state.Name.ValueString())
	}
}

func TestTrafficMatchingListResourceReadNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(client.APIError{StatusCode: 404, StatusName: "NOT_FOUND"}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	r := NewTrafficMatchingListResource()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)

	tmlr := r.(*TrafficMatchingListResource)
	tmlr.client = client.NewClientForTesting("key", "host", server.URL)

	state := makeState(t, r, trafficMatchingListTestValues(map[string]tftypes.Value{
		"id":   tftypes.NewValue(tftypes.String, "tml-gone"),
		"name": tftypes.NewValue(tftypes.String, "Gone List"),
	}))

	readResp := &resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, readResp)

	if readResp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors on 404: %v", readResp.Diagnostics)
	}
	if !readResp.State.Raw.IsNull() {
		t.Error("expected state to be null after 404")
	}
}

func TestTrafficMatchingListResourceDelete(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	r := NewTrafficMatchingListResource()

	tmlr := r.(*TrafficMatchingListResource)
	tmlr.client = client.NewClientForTesting("key", "host", server.URL)

	state := makeState(t, r, trafficMatchingListTestValues(map[string]tftypes.Value{
		"id":   tftypes.NewValue(tftypes.String, "tml-del"),
		"name": tftypes.NewValue(tftypes.String, "To Delete"),
	}))

	deleteResp := &resource.DeleteResponse{}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)

	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", deleteResp.Diagnostics)
	}
	if !called {
		t.Error("expected DELETE request to be sent")
	}
}

// --- DNS Policy resource CRUD ---

// dnsPolicyTestValues merges the provided overrides into a complete set of
// tftypes values for a DnsPolicy resource.
func dnsPolicyTestValues(overrides map[string]tftypes.Value) map[string]tftypes.Value {
	defaults := map[string]tftypes.Value{
		"id":           tftypes.NewValue(tftypes.String, nil),
		"site_id":      tftypes.NewValue(tftypes.String, "site-1"),
		"type":         tftypes.NewValue(tftypes.String, "A_RECORD"),
		"enabled":      tftypes.NewValue(tftypes.Bool, true),
		"name":         tftypes.NewValue(tftypes.String, "Test DNS"),
		"domain":       tftypes.NewValue(tftypes.String, nil),
		"ipv4_address": tftypes.NewValue(tftypes.String, nil),
		"ipv6_address": tftypes.NewValue(tftypes.String, nil),
		"target":       tftypes.NewValue(tftypes.String, nil),
		"ttl_seconds":  tftypes.NewValue(tftypes.Number, nil),
		"priority":     tftypes.NewValue(tftypes.Number, nil),
		"weight":       tftypes.NewValue(tftypes.Number, nil),
		"port":         tftypes.NewValue(tftypes.Number, nil),
		"txt_value":    tftypes.NewValue(tftypes.String, nil),
		"forward_to":   tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
	}
	for k, v := range overrides {
		defaults[k] = v
	}
	return defaults
}

func TestDnsPolicyResourceCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(client.DnsPolicy{
			ID:          "dns-created",
			Type:        "A_RECORD",
			Enabled:     true,
			Name:        "My A Record",
			Domain:      "example.com",
			IPv4Address: "1.2.3.4",
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	r := NewDnsPolicyResource()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)

	dpr := r.(*DnsPolicyResource)
	dpr.client = client.NewClientForTesting("key", "host", server.URL)

	plan := makePlan(t, r, dnsPolicyTestValues(map[string]tftypes.Value{
		"name":         tftypes.NewValue(tftypes.String, "My A Record"),
		"domain":       tftypes.NewValue(tftypes.String, "example.com"),
		"ipv4_address": tftypes.NewValue(tftypes.String, "1.2.3.4"),
	}))

	createResp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)

	if createResp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", createResp.Diagnostics)
	}

	var state DnsPolicyResourceModel
	createResp.State.Get(context.Background(), &state)
	if state.ID.ValueString() != "dns-created" {
		t.Errorf("expected ID 'dns-created', got %q", state.ID.ValueString())
	}
	if state.Name.ValueString() != "My A Record" {
		t.Errorf("expected name 'My A Record', got %q", state.Name.ValueString())
	}
}

func TestDnsPolicyResourceReadNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(client.APIError{StatusCode: 404, StatusName: "NOT_FOUND"}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	r := NewDnsPolicyResource()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)

	dpr := r.(*DnsPolicyResource)
	dpr.client = client.NewClientForTesting("key", "host", server.URL)

	state := makeState(t, r, dnsPolicyTestValues(map[string]tftypes.Value{
		"id":   tftypes.NewValue(tftypes.String, "dns-gone"),
		"name": tftypes.NewValue(tftypes.String, "Gone DNS"),
	}))

	readResp := &resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, readResp)

	if readResp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors on 404: %v", readResp.Diagnostics)
	}
	if !readResp.State.Raw.IsNull() {
		t.Error("expected state to be null after 404")
	}
}

func TestDnsPolicyResourceDelete(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	r := NewDnsPolicyResource()

	dpr := r.(*DnsPolicyResource)
	dpr.client = client.NewClientForTesting("key", "host", server.URL)

	state := makeState(t, r, dnsPolicyTestValues(map[string]tftypes.Value{
		"id":   tftypes.NewValue(tftypes.String, "dns-del"),
		"name": tftypes.NewValue(tftypes.String, "To Delete"),
	}))

	deleteResp := &resource.DeleteResponse{}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)

	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", deleteResp.Diagnostics)
	}
	if !called {
		t.Error("expected DELETE request to be sent")
	}
}

// --- Hotspot Voucher resource CRUD ---

// hotspotVoucherTestValues merges the provided overrides into a complete set of
// tftypes values for a HotspotVoucher resource.
func hotspotVoucherTestValues(overrides map[string]tftypes.Value) map[string]tftypes.Value {
	defaults := map[string]tftypes.Value{
		"id":                      tftypes.NewValue(tftypes.String, nil),
		"site_id":                 tftypes.NewValue(tftypes.String, "site-1"),
		"name":                    tftypes.NewValue(tftypes.String, "Test Voucher"),
		"code":                    tftypes.NewValue(tftypes.String, nil),
		"time_limit_minutes":      tftypes.NewValue(tftypes.Number, 60),
		"authorized_guest_limit":  tftypes.NewValue(tftypes.Number, nil),
		"data_usage_limit_mbytes": tftypes.NewValue(tftypes.Number, nil),
		"rx_rate_limit_kbps":      tftypes.NewValue(tftypes.Number, nil),
		"tx_rate_limit_kbps":      tftypes.NewValue(tftypes.Number, nil),
	}
	for k, v := range overrides {
		defaults[k] = v
	}
	return defaults
}

func TestHotspotVoucherResourceCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		// The API returns a JSON array of vouchers
		if err := json.NewEncoder(w).Encode([]client.HotspotVoucher{
			{
				ID:               "voucher-created",
				Name:             "Guest Pass",
				Code:             "ABCD-1234",
				TimeLimitMinutes: 120,
			},
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	r := NewHotspotVoucherResource()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)

	hvr := r.(*HotspotVoucherResource)
	hvr.client = client.NewClientForTesting("key", "host", server.URL)

	plan := makePlan(t, r, hotspotVoucherTestValues(map[string]tftypes.Value{
		"name":               tftypes.NewValue(tftypes.String, "Guest Pass"),
		"time_limit_minutes": tftypes.NewValue(tftypes.Number, 120),
	}))

	createResp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)

	if createResp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", createResp.Diagnostics)
	}

	var state HotspotVoucherResourceModel
	createResp.State.Get(context.Background(), &state)
	if state.ID.ValueString() != "voucher-created" {
		t.Errorf("expected ID 'voucher-created', got %q", state.ID.ValueString())
	}
	if state.Code.ValueString() != "ABCD-1234" {
		t.Errorf("expected code 'ABCD-1234', got %q", state.Code.ValueString())
	}
	if state.Name.ValueString() != "Guest Pass" {
		t.Errorf("expected name 'Guest Pass', got %q", state.Name.ValueString())
	}
}

func TestHotspotVoucherResourceDelete(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	r := NewHotspotVoucherResource()

	hvr := r.(*HotspotVoucherResource)
	hvr.client = client.NewClientForTesting("key", "host", server.URL)

	state := makeState(t, r, hotspotVoucherTestValues(map[string]tftypes.Value{
		"id":   tftypes.NewValue(tftypes.String, "voucher-del"),
		"name": tftypes.NewValue(tftypes.String, "To Delete"),
		"code": tftypes.NewValue(tftypes.String, "XXXX-9999"),
	}))

	deleteResp := &resource.DeleteResponse{}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)

	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", deleteResp.Diagnostics)
	}
	if !called {
		t.Error("expected DELETE request to be sent")
	}
}
