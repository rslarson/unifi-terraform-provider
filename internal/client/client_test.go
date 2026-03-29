package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- Client construction ---

func TestNewClient(t *testing.T) {
	c := NewClient("test-key", "test-host")
	if c.apiKey != "test-key" {
		t.Errorf("expected apiKey 'test-key', got %q", c.apiKey)
	}
	if c.hostID != "test-host" {
		t.Errorf("expected hostID 'test-host', got %q", c.hostID)
	}
	if c.baseURL != defaultBaseURL {
		t.Errorf("expected baseURL %q, got %q", defaultBaseURL, c.baseURL)
	}
}

func TestNewClientForTesting(t *testing.T) {
	c := NewClientForTesting("key", "host", "http://localhost:9999")
	if c.baseURL != "http://localhost:9999" {
		t.Errorf("expected custom baseURL, got %q", c.baseURL)
	}
	if c.apiKey != "key" {
		t.Errorf("expected apiKey 'key', got %q", c.apiKey)
	}
}

// --- URL construction ---

func TestBuildURL(t *testing.T) {
	c := NewClient("key", "my-host-id")
	url := c.buildURL("sites/abc/networks")
	expected := "https://api.ui.com/v1/connector/consoles/my-host-id/v1/sites/abc/networks"
	if url != expected {
		t.Errorf("expected %q, got %q", expected, url)
	}
}

func TestBuildURLVariousPaths(t *testing.T) {
	c := NewClient("key", "host123")
	tests := []struct {
		path     string
		expected string
	}{
		{
			"sites",
			"https://api.ui.com/v1/connector/consoles/host123/v1/sites",
		},
		{
			"sites/site-1/wifi/broadcasts",
			"https://api.ui.com/v1/connector/consoles/host123/v1/sites/site-1/wifi/broadcasts",
		},
		{
			"sites/site-1/firewall/zones/zone-1",
			"https://api.ui.com/v1/connector/consoles/host123/v1/sites/site-1/firewall/zones/zone-1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := c.buildURL(tt.path)
			if got != tt.expected {
				t.Errorf("buildURL(%q) = %q, want %q", tt.path, got, tt.expected)
			}
		})
	}
}

// --- Request headers ---

func TestRequestSetsAuthHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "my-secret-key" {
			t.Errorf("expected X-API-Key 'my-secret-key', got %q", r.Header.Get("X-API-Key"))
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("expected Accept 'application/json', got %q", r.Header.Get("Accept"))
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(PaginatedResponse[Site]{Data: []Site{}}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("my-secret-key", "host", server.URL)
	_, err := c.ListSites(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPostRequestSetsContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("expected Content-Type 'application/json', got %q", r.Header.Get("Content-Type"))
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(Network{ID: "net-1", Name: "n"}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	_, err := c.CreateNetwork(context.Background(), "site-1", &Network{Name: "n", Management: ManagementUnmanaged, VlanID: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Network CRUD ---

func TestCreateNetwork(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sites/site-1/networks") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var req Network
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if req.Name != "Test Net" {
			t.Errorf("expected name 'Test Net', got %q", req.Name)
		}
		if req.Management != ManagementUnmanaged {
			t.Errorf("expected management %q, got %q", ManagementUnmanaged, req.Management)
		}
		if req.VlanID != 100 {
			t.Errorf("expected vlanId 100, got %d", req.VlanID)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(Network{
			ID:         "net-123",
			Name:       req.Name,
			Management: req.Management,
			Enabled:    req.Enabled,
			VlanID:     req.VlanID,
		}); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("test-key", "host-id", server.URL)
	result, err := c.CreateNetwork(context.Background(), "site-1", &Network{
		Name:       "Test Net",
		Management: ManagementUnmanaged,
		Enabled:    true,
		VlanID:     100,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "net-123" {
		t.Errorf("expected ID 'net-123', got %q", result.ID)
	}
	if result.Name != "Test Net" {
		t.Errorf("expected name 'Test Net', got %q", result.Name)
	}
}

func TestCreateNetworkWithDhcpGuarding(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req Network
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if req.DhcpGuarding == nil {
			t.Error("expected dhcpGuarding in request")
		} else if len(req.DhcpGuarding.TrustedDhcpServerIPAddresses) != 2 {
			t.Errorf("expected 2 trusted IPs, got %d", len(req.DhcpGuarding.TrustedDhcpServerIPAddresses))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(&req); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	_, err := c.CreateNetwork(context.Background(), "site-1", &Network{
		Name:       "Guarded",
		Management: ManagementGateway,
		VlanID:     200,
		DhcpGuarding: &DhcpGuarding{
			TrustedDhcpServerIPAddresses: []string{"192.168.1.1", "10.0.0.1"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetNetwork(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sites/site-1/networks/net-456") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(Network{
			ID:         "net-456",
			Name:       "My Network",
			Management: ManagementGateway,
			Enabled:    true,
			VlanID:     200,
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("test-key", "host-id", server.URL)
	result, err := c.GetNetwork(context.Background(), "site-1", "net-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "net-456" {
		t.Errorf("expected ID 'net-456', got %q", result.ID)
	}
	if result.Management != ManagementGateway {
		t.Errorf("expected management %q, got %q", ManagementGateway, result.Management)
	}
}

func TestUpdateNetwork(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sites/site-1/networks/net-123") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var req Network
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		req.ID = "net-123"
		if err := json.NewEncoder(w).Encode(req); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	result, err := c.UpdateNetwork(context.Background(), "site-1", "net-123", &Network{
		Name:       "Updated Net",
		Management: ManagementSwitch,
		Enabled:    false,
		VlanID:     300,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Updated Net" {
		t.Errorf("expected name 'Updated Net', got %q", result.Name)
	}
}

func TestDeleteNetwork(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sites/site-1/networks/net-123") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClientForTesting("test-key", "host-id", server.URL)
	err := c.DeleteNetwork(context.Background(), "site-1", "net-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListNetworks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("limit") != defaultPageSize {
			t.Errorf("expected limit=%s, got %q", defaultPageSize, r.URL.Query().Get("limit"))
		}
		if !strings.Contains(r.URL.Path, "/sites/site-1/networks") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(PaginatedResponse[Network]{
			Offset:     0,
			Limit:      200,
			Count:      2,
			TotalCount: 2,
			Data: []Network{
				{ID: "net-1", Name: "Network 1", Management: ManagementUnmanaged, VlanID: 100},
				{ID: "net-2", Name: "Network 2", Management: ManagementGateway, VlanID: 200},
			},
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("test-key", "host-id", server.URL)
	networks, err := c.ListNetworks(context.Background(), "site-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(networks) != 2 {
		t.Errorf("expected 2 networks, got %d", len(networks))
	}
	if networks[0].ID != "net-1" {
		t.Errorf("expected first network ID 'net-1', got %q", networks[0].ID)
	}
}

func TestListNetworksEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(PaginatedResponse[Network]{
			Offset: 0, Limit: 200, Count: 0, TotalCount: 0, Data: []Network{},
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	networks, err := c.ListNetworks(context.Background(), "site-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(networks) != 0 {
		t.Errorf("expected 0 networks, got %d", len(networks))
	}
}

// --- WiFi Broadcast CRUD ---

func TestCreateWifiBroadcast(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sites/site-1/wifi/broadcasts") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var req WifiBroadcast
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if req.SecurityConfiguration == nil {
			t.Error("expected securityConfiguration in request")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(WifiBroadcast{
			ID:      "wifi-123",
			Type:    req.Type,
			Name:    req.Name,
			Enabled: req.Enabled,
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("test-key", "host-id", server.URL)
	result, err := c.CreateWifiBroadcast(context.Background(), "site-1", &WifiBroadcast{
		Type:    BroadcastTypeStandard,
		Name:    "TestSSID",
		Enabled: true,
		SecurityConfiguration: &SecurityConfiguration{
			Type:       SecurityWPA2Personal,
			Passphrase: "password123",
		},
		Network: &BroadcastNetwork{Type: NetworkTypeNative},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "wifi-123" {
		t.Errorf("expected ID 'wifi-123', got %q", result.ID)
	}
}

func TestCreateWifiBroadcastPassphraseInBody(t *testing.T) {
	var captured WifiBroadcast
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(WifiBroadcast{ID: "wifi-1"}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	_, err := c.CreateWifiBroadcast(context.Background(), "site-1", &WifiBroadcast{
		Type: BroadcastTypeStandard,
		Name: "Secure",
		SecurityConfiguration: &SecurityConfiguration{
			Type:       SecurityWPA3Personal,
			Passphrase: "supersecret99",
		},
		Network: &BroadcastNetwork{Type: NetworkTypeNative},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured.SecurityConfiguration == nil || captured.SecurityConfiguration.Passphrase != "supersecret99" {
		t.Errorf("expected passphrase 'supersecret99' in request body")
	}
}

func TestGetWifiBroadcast(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sites/site-1/wifi/broadcasts/wifi-456") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(WifiBroadcast{
			ID:                    "wifi-456",
			Type:                  BroadcastTypeIoTOptimized,
			Name:                  "IoT Network",
			Enabled:               true,
			SecurityConfiguration: &SecurityConfiguration{Type: SecurityOpen},
			Network:               &BroadcastNetwork{Type: NetworkTypeNative},
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	result, err := c.GetWifiBroadcast(context.Background(), "site-1", "wifi-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "wifi-456" {
		t.Errorf("expected ID 'wifi-456', got %q", result.ID)
	}
	if result.Type != BroadcastTypeIoTOptimized {
		t.Errorf("expected type %q, got %q", BroadcastTypeIoTOptimized, result.Type)
	}
}

func TestUpdateWifiBroadcast(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sites/site-1/wifi/broadcasts/wifi-123") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var req WifiBroadcast
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		req.ID = "wifi-123"
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(req); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	result, err := c.UpdateWifiBroadcast(context.Background(), "site-1", "wifi-123", &WifiBroadcast{
		Type:                  BroadcastTypeStandard,
		Name:                  "Updated SSID",
		Enabled:               false,
		SecurityConfiguration: &SecurityConfiguration{Type: SecurityOpen},
		Network:               &BroadcastNetwork{Type: NetworkTypeNative},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Updated SSID" {
		t.Errorf("expected name 'Updated SSID', got %q", result.Name)
	}
}

func TestDeleteWifiBroadcast(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sites/site-1/wifi/broadcasts/wifi-123") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	err := c.DeleteWifiBroadcast(context.Background(), "site-1", "wifi-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Firewall Zone CRUD ---

func TestCreateFirewallZone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sites/site-1/firewall/zones") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var req FirewallZone
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(FirewallZone{
			ID:         "zone-123",
			Name:       req.Name,
			NetworkIDs: req.NetworkIDs,
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("test-key", "host-id", server.URL)
	result, err := c.CreateFirewallZone(context.Background(), "site-1", &FirewallZone{
		Name:       "Test Zone",
		NetworkIDs: []string{"net-1", "net-2"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "zone-123" {
		t.Errorf("expected ID 'zone-123', got %q", result.ID)
	}
	if len(result.NetworkIDs) != 2 {
		t.Errorf("expected 2 network IDs, got %d", len(result.NetworkIDs))
	}
}

func TestGetFirewallZone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sites/site-1/firewall/zones/zone-456") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(FirewallZone{
			ID:         "zone-456",
			Name:       "My Zone",
			NetworkIDs: []string{"net-a", "net-b", "net-c"},
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	result, err := c.GetFirewallZone(context.Background(), "site-1", "zone-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "My Zone" {
		t.Errorf("expected name 'My Zone', got %q", result.Name)
	}
	if len(result.NetworkIDs) != 3 {
		t.Errorf("expected 3 network IDs, got %d", len(result.NetworkIDs))
	}
}

func TestUpdateFirewallZone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sites/site-1/firewall/zones/zone-123") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var req FirewallZone
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		req.ID = "zone-123"
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(req); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	result, err := c.UpdateFirewallZone(context.Background(), "site-1", "zone-123", &FirewallZone{
		Name:       "Updated Zone",
		NetworkIDs: []string{"net-x"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Updated Zone" {
		t.Errorf("expected name 'Updated Zone', got %q", result.Name)
	}
}

func TestDeleteFirewallZone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sites/site-1/firewall/zones/zone-123") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	err := c.DeleteFirewallZone(context.Background(), "site-1", "zone-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Device operations ---

func TestGetDevice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sites/site-1/devices/dev-789") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(Device{
			ID:                "dev-789",
			MacAddress:        "aa:bb:cc:dd:ee:ff",
			IPAddress:         "192.168.1.100",
			Name:              "My AP",
			Model:             "U6-Pro",
			Supported:         true,
			State:             "ONLINE",
			FirmwareVersion:   "6.5.0",
			FirmwareUpdatable: false,
			AdoptedAt:         "2024-01-01T00:00:00Z",
			ProvisionedAt:     "2024-01-01T00:01:00Z",
			ConfigurationID:   "cfg-abc",
			Uplink:            &DeviceUplink{DeviceID: "dev-parent"},
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	result, err := c.GetDevice(context.Background(), "site-1", "dev-789")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "dev-789" {
		t.Errorf("expected ID 'dev-789', got %q", result.ID)
	}
	if result.MacAddress != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("expected mac 'aa:bb:cc:dd:ee:ff', got %q", result.MacAddress)
	}
	if result.Uplink == nil || result.Uplink.DeviceID != "dev-parent" {
		t.Error("expected uplink.deviceId 'dev-parent'")
	}
}

func TestListDevices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("limit") != defaultPageSize {
			t.Errorf("expected limit=%s, got %q", defaultPageSize, r.URL.Query().Get("limit"))
		}
		if !strings.Contains(r.URL.Path, "/sites/site-1/devices") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(PaginatedResponse[Device]{
			Offset: 0, Limit: 200, Count: 2, TotalCount: 2,
			Data: []Device{
				{ID: "dev-1", Name: "AP 1", State: "ONLINE"},
				{ID: "dev-2", Name: "Switch 1", State: "OFFLINE"},
			},
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	devices, err := c.ListDevices(context.Background(), "site-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(devices) != 2 {
		t.Errorf("expected 2 devices, got %d", len(devices))
	}
}

// --- Sites ---

func TestListSites(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/sites") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(PaginatedResponse[Site]{
			Offset:     0,
			Limit:      200,
			Count:      1,
			TotalCount: 1,
			Data: []Site{
				{ID: "site-1", Name: "Default", InternalReference: "default"},
			},
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("test-key", "host-id", server.URL)
	sites, err := c.ListSites(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sites) != 1 {
		t.Errorf("expected 1 site, got %d", len(sites))
	}
	if sites[0].Name != "Default" {
		t.Errorf("expected name 'Default', got %q", sites[0].Name)
	}
	if sites[0].InternalReference != "default" {
		t.Errorf("expected internalReference 'default', got %q", sites[0].InternalReference)
	}
}

// --- Error handling ---

func TestAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(APIError{
			StatusCode: 404,
			StatusName: "NOT_FOUND",
			Code:       "api.resource.not-found",
			Message:    "Network not found",
			RequestID:  "req-789",
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("test-key", "host-id", server.URL)
	_, err := c.GetNetwork(context.Background(), "site-1", "missing")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("expected status code 404, got %d", apiErr.StatusCode)
	}
	if apiErr.Code != "api.resource.not-found" {
		t.Errorf("expected code 'api.resource.not-found', got %q", apiErr.Code)
	}
	if apiErr.RequestID != "req-789" {
		t.Errorf("expected requestId 'req-789', got %q", apiErr.RequestID)
	}
}

func TestAPIErrorNon404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(APIError{
			StatusCode: 400,
			StatusName: "BAD_REQUEST",
			Code:       "api.validation.error",
			Message:    "vlanId must be between 2 and 4009",
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	_, err := c.CreateNetwork(context.Background(), "site-1", &Network{Name: "bad", VlanID: 1})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 400 {
		t.Errorf("expected 400, got %d", apiErr.StatusCode)
	}
	if IsNotFound(err) {
		t.Error("expected IsNotFound to return false for 400 error")
	}
}

func TestAPIErrorNonJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := io.WriteString(w, "Internal Server Error"); err != nil {
			t.Fatalf("write error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	_, err := c.GetNetwork(context.Background(), "site-1", "net-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Non-JSON error body should still return an error (not a typed APIError)
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		t.Error("expected non-APIError for non-JSON response")
	}
}

func TestAPIErrorString(t *testing.T) {
	err := &APIError{
		StatusCode: 404,
		StatusName: "NOT_FOUND",
		Code:       "api.resource.not-found",
		Message:    "resource does not exist",
		RequestID:  "req-abc",
	}
	s := err.Error()
	if !strings.Contains(s, "404") {
		t.Errorf("expected '404' in error string, got %q", s)
	}
	if !strings.Contains(s, "NOT_FOUND") {
		t.Errorf("expected 'NOT_FOUND' in error string, got %q", s)
	}
	if !strings.Contains(s, "resource does not exist") {
		t.Errorf("expected message in error string, got %q", s)
	}
	if !strings.Contains(s, "req-abc") {
		t.Errorf("expected requestId in error string, got %q", s)
	}
}

func TestIsNotFound(t *testing.T) {
	notFoundErr := &APIError{StatusCode: http.StatusNotFound, StatusName: "NOT_FOUND"}
	if !IsNotFound(notFoundErr) {
		t.Error("expected IsNotFound to return true for 404")
	}

	badRequestErr := &APIError{StatusCode: http.StatusBadRequest, StatusName: "BAD_REQUEST"}
	if IsNotFound(badRequestErr) {
		t.Error("expected IsNotFound to return false for 400")
	}

	serverErr := &APIError{StatusCode: http.StatusInternalServerError, StatusName: "INTERNAL"}
	if IsNotFound(serverErr) {
		t.Error("expected IsNotFound to return false for 500")
	}

	if IsNotFound(nil) {
		t.Error("expected IsNotFound to return false for nil")
	}

	genericErr := fmt.Errorf("some error")
	if IsNotFound(genericErr) {
		t.Error("expected IsNotFound to return false for non-APIError")
	}
}

func TestGetNetworkNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(APIError{StatusCode: 404, StatusName: "NOT_FOUND"}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	_, err := c.GetNetwork(context.Background(), "site-1", "missing")
	if !IsNotFound(err) {
		t.Errorf("expected IsNotFound to be true, got err: %v", err)
	}
}

// --- JSON marshaling ---

func TestNetworkJSONMarshal(t *testing.T) {
	n := Network{
		Name:       "Test",
		Management: ManagementGateway,
		Enabled:    true,
		VlanID:     100,
		DhcpGuarding: &DhcpGuarding{
			TrustedDhcpServerIPAddresses: []string{"10.0.0.1"},
		},
	}
	data, err := json.Marshal(n)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded Network
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.Name != "Test" {
		t.Errorf("expected name 'Test', got %q", decoded.Name)
	}
	if decoded.VlanID != 100 {
		t.Errorf("expected vlanId 100, got %d", decoded.VlanID)
	}
	if decoded.DhcpGuarding == nil || len(decoded.DhcpGuarding.TrustedDhcpServerIPAddresses) != 1 {
		t.Error("expected dhcpGuarding with 1 trusted IP")
	}
}

func TestNetworkIDOmittedWhenEmpty(t *testing.T) {
	n := Network{Name: "Test", Management: ManagementUnmanaged, VlanID: 10}
	data, err := json.Marshal(n)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	s := string(data)
	if strings.Contains(s, `"id"`) {
		t.Errorf("expected 'id' to be omitted when empty, got: %s", s)
	}
}

func TestWifiBroadcastJSONMarshal(t *testing.T) {
	wb := WifiBroadcast{
		ID:      "wifi-1",
		Type:    BroadcastTypeStandard,
		Name:    "MySSID",
		Enabled: true,
		SecurityConfiguration: &SecurityConfiguration{
			Type:       SecurityWPA2Personal,
			Passphrase: "mypass123",
		},
		Network: &BroadcastNetwork{
			Type:      NetworkTypeSpecific,
			NetworkID: "net-abc",
		},
		ClientIsolationEnabled: true,
		HideName:               false,
	}
	data, err := json.Marshal(wb)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded WifiBroadcast
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.SecurityConfiguration.Passphrase != "mypass123" {
		t.Errorf("expected passphrase 'mypass123', got %q", decoded.SecurityConfiguration.Passphrase)
	}
	if decoded.Network.NetworkID != "net-abc" {
		t.Errorf("expected networkId 'net-abc', got %q", decoded.Network.NetworkID)
	}
}

func TestFirewallZoneJSONMarshal(t *testing.T) {
	z := FirewallZone{
		Name:       "LAN Zone",
		NetworkIDs: []string{"net-1", "net-2"},
	}
	data, err := json.Marshal(z)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var decoded FirewallZone
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.Name != "LAN Zone" {
		t.Errorf("expected name 'LAN Zone', got %q", decoded.Name)
	}
	if len(decoded.NetworkIDs) != 2 {
		t.Errorf("expected 2 network IDs, got %d", len(decoded.NetworkIDs))
	}
}

func TestPaginatedResponseParsing(t *testing.T) {
	raw := `{
		"offset": 0,
		"limit": 200,
		"count": 3,
		"totalCount": 10,
		"data": [
			{"id": "s1", "name": "Site 1", "internalReference": "site1"},
			{"id": "s2", "name": "Site 2", "internalReference": "site2"},
			{"id": "s3", "name": "Site 3", "internalReference": "site3"}
		]
	}`
	var resp PaginatedResponse[Site]
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if resp.Count != 3 {
		t.Errorf("expected count 3, got %d", resp.Count)
	}
	if resp.TotalCount != 10 {
		t.Errorf("expected totalCount 10, got %d", resp.TotalCount)
	}
	if len(resp.Data) != 3 {
		t.Errorf("expected 3 sites, got %d", len(resp.Data))
	}
	if resp.Data[1].Name != "Site 2" {
		t.Errorf("expected second site name 'Site 2', got %q", resp.Data[1].Name)
	}
}

// --- Constants and helper functions ---

func TestIsPersonalSecurityType(t *testing.T) {
	tests := []struct {
		securityType string
		want         bool
	}{
		{SecurityWPA2Personal, true},
		{SecurityWPA3Personal, true},
		{SecurityWPA2WPA3Personal, true},
		{SecurityOpen, false},
		{SecurityWPA2Enterprise, false},
		{SecurityWPA3Enterprise, false},
		{SecurityWPA2WPA3Enterprise, false},
		{"UNKNOWN", false},
		{"", false},
	}
	for _, tt := range tests {
		name := tt.securityType
		if name == "" {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			if got := IsPersonalSecurityType(tt.securityType); got != tt.want {
				t.Errorf("IsPersonalSecurityType(%q) = %v, want %v", tt.securityType, got, tt.want)
			}
		})
	}
}

func TestConstants(t *testing.T) {
	if ManagementUnmanaged != "UNMANAGED" {
		t.Errorf("unexpected value: %s", ManagementUnmanaged)
	}
	if ManagementGateway != "GATEWAY" {
		t.Errorf("unexpected value: %s", ManagementGateway)
	}
	if ManagementSwitch != "SWITCH" {
		t.Errorf("unexpected value: %s", ManagementSwitch)
	}
	if BroadcastTypeStandard != "STANDARD" {
		t.Errorf("unexpected value: %s", BroadcastTypeStandard)
	}
	if BroadcastTypeIoTOptimized != "IOT_OPTIMIZED" {
		t.Errorf("unexpected value: %s", BroadcastTypeIoTOptimized)
	}
	if SecurityOpen != "OPEN" {
		t.Errorf("unexpected value: %s", SecurityOpen)
	}
	if SecurityWPA2Personal != "WPA2_PERSONAL" {
		t.Errorf("unexpected value: %s", SecurityWPA2Personal)
	}
	if SecurityWPA3Personal != "WPA3_PERSONAL" {
		t.Errorf("unexpected value: %s", SecurityWPA3Personal)
	}
	if SecurityWPA2WPA3Personal != "WPA2_WPA3_PERSONAL" {
		t.Errorf("unexpected value: %s", SecurityWPA2WPA3Personal)
	}
	if SecurityWPA2Enterprise != "WPA2_ENTERPRISE" {
		t.Errorf("unexpected value: %s", SecurityWPA2Enterprise)
	}
	if SecurityWPA3Enterprise != "WPA3_ENTERPRISE" {
		t.Errorf("unexpected value: %s", SecurityWPA3Enterprise)
	}
	if SecurityWPA2WPA3Enterprise != "WPA2_WPA3_ENTERPRISE" {
		t.Errorf("unexpected value: %s", SecurityWPA2WPA3Enterprise)
	}
	if NetworkTypeNative != "NATIVE" {
		t.Errorf("unexpected value: %s", NetworkTypeNative)
	}
	if NetworkTypeSpecific != "SPECIFIC" {
		t.Errorf("unexpected value: %s", NetworkTypeSpecific)
	}
}

// --- ACL Rule CRUD ---

func TestCreateAclRule(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sites/site-1/acl-rules") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var req AclRule
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if req.Name != "Block IoT" {
			t.Errorf("expected name 'Block IoT', got %q", req.Name)
		}
		if req.Type != AclTypeIPv4 {
			t.Errorf("expected type %q, got %q", AclTypeIPv4, req.Type)
		}
		if req.Action != ActionBlock {
			t.Errorf("expected action %q, got %q", ActionBlock, req.Action)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(AclRule{
			ID:           "acl-123",
			Type:         req.Type,
			Enabled:      req.Enabled,
			Name:         req.Name,
			Action:       req.Action,
			SourceFilter: req.SourceFilter,
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("test-key", "host-id", server.URL)
	result, err := c.CreateAclRule(context.Background(), "site-1", &AclRule{
		Type:    AclTypeIPv4,
		Enabled: true,
		Name:    "Block IoT",
		Action:  ActionBlock,
		SourceFilter: &AclFilter{
			Type:                 "IP_ADDRESS",
			IPAddressesOrSubnets: []string{"192.168.10.0/24"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "acl-123" {
		t.Errorf("expected ID 'acl-123', got %q", result.ID)
	}
	if result.Name != "Block IoT" {
		t.Errorf("expected name 'Block IoT', got %q", result.Name)
	}
	if result.Action != ActionBlock {
		t.Errorf("expected action %q, got %q", ActionBlock, result.Action)
	}
}

func TestGetAclRule(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sites/site-1/acl-rules/acl-456") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(AclRule{
			ID:          "acl-456",
			Type:        AclTypeMAC,
			Enabled:     true,
			Name:        "MAC Filter",
			Description: "Block by MAC",
			Action:      ActionBlock,
			SourceFilter: &AclFilter{
				Type:         "MAC_ADDRESS",
				MacAddresses: []string{"aa:bb:cc:dd:ee:ff"},
			},
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	result, err := c.GetAclRule(context.Background(), "site-1", "acl-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "acl-456" {
		t.Errorf("expected ID 'acl-456', got %q", result.ID)
	}
	if result.Type != AclTypeMAC {
		t.Errorf("expected type %q, got %q", AclTypeMAC, result.Type)
	}
	if result.Name != "MAC Filter" {
		t.Errorf("expected name 'MAC Filter', got %q", result.Name)
	}
	if result.SourceFilter == nil || len(result.SourceFilter.MacAddresses) != 1 {
		t.Error("expected sourceFilter with 1 MAC address")
	}
}

func TestUpdateAclRule(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sites/site-1/acl-rules/acl-123") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var req AclRule
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		req.ID = "acl-123"
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(req); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	result, err := c.UpdateAclRule(context.Background(), "site-1", "acl-123", &AclRule{
		Type:    AclTypeIPv4,
		Enabled: false,
		Name:    "Updated ACL",
		Action:  ActionAllow,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Updated ACL" {
		t.Errorf("expected name 'Updated ACL', got %q", result.Name)
	}
	if result.Action != ActionAllow {
		t.Errorf("expected action %q, got %q", ActionAllow, result.Action)
	}
}

func TestDeleteAclRule(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sites/site-1/acl-rules/acl-123") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	err := c.DeleteAclRule(context.Background(), "site-1", "acl-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListAclRules(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("limit") != defaultPageSize {
			t.Errorf("expected limit=%s, got %q", defaultPageSize, r.URL.Query().Get("limit"))
		}
		if !strings.Contains(r.URL.Path, "/sites/site-1/acl-rules") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(PaginatedResponse[AclRule]{
			Offset: 0, Limit: 200, Count: 2, TotalCount: 2,
			Data: []AclRule{
				{ID: "acl-1", Name: "Rule 1", Type: AclTypeIPv4, Action: ActionAllow},
				{ID: "acl-2", Name: "Rule 2", Type: AclTypeMAC, Action: ActionBlock},
			},
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	rules, err := c.ListAclRules(context.Background(), "site-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 2 {
		t.Errorf("expected 2 ACL rules, got %d", len(rules))
	}
	if rules[0].ID != "acl-1" {
		t.Errorf("expected first rule ID 'acl-1', got %q", rules[0].ID)
	}
	if rules[1].Action != ActionBlock {
		t.Errorf("expected second rule action %q, got %q", ActionBlock, rules[1].Action)
	}
}

// --- Firewall Policy CRUD ---

func TestCreateFirewallPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sites/site-1/firewall/policies") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var req FirewallPolicy
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if req.Name != "Block Outbound" {
			t.Errorf("expected name 'Block Outbound', got %q", req.Name)
		}
		if req.Action == nil || req.Action.Type != ActionBlock {
			t.Errorf("expected action type %q", ActionBlock)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(FirewallPolicy{
			ID:              "pol-123",
			Enabled:         req.Enabled,
			Name:            req.Name,
			Action:          req.Action,
			Source:          req.Source,
			Destination:     req.Destination,
			IPProtocolScope: req.IPProtocolScope,
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("test-key", "host-id", server.URL)
	result, err := c.CreateFirewallPolicy(context.Background(), "site-1", &FirewallPolicy{
		Enabled:         true,
		Name:            "Block Outbound",
		Action:          &FirewallAction{Type: ActionBlock},
		Source:          &FirewallEndpoint{ZoneID: "zone-lan"},
		Destination:     &FirewallEndpoint{ZoneID: "zone-wan"},
		IPProtocolScope: &IPProtocolScope{IPVersion: IPVersionIPv4},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "pol-123" {
		t.Errorf("expected ID 'pol-123', got %q", result.ID)
	}
	if result.Name != "Block Outbound" {
		t.Errorf("expected name 'Block Outbound', got %q", result.Name)
	}
	if result.Source == nil || result.Source.ZoneID != "zone-lan" {
		t.Error("expected source zoneId 'zone-lan'")
	}
}

func TestGetFirewallPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sites/site-1/firewall/policies/pol-456") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		allowReturn := true
		if err := json.NewEncoder(w).Encode(FirewallPolicy{
			ID:          "pol-456",
			Enabled:     true,
			Name:        "Allow SSH",
			Description: "Allow SSH from LAN to servers",
			Action:      &FirewallAction{Type: ActionAllow, AllowReturnTraffic: &allowReturn},
			Source:      &FirewallEndpoint{ZoneID: "zone-lan"},
			Destination: &FirewallEndpoint{ZoneID: "zone-dmz"},
			IPProtocolScope: &IPProtocolScope{
				IPVersion: IPVersionIPv4,
				Protocol:  &ProtocolFilter{Type: "TCP"},
			},
			LoggingEnabled: true,
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	result, err := c.GetFirewallPolicy(context.Background(), "site-1", "pol-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "pol-456" {
		t.Errorf("expected ID 'pol-456', got %q", result.ID)
	}
	if result.Name != "Allow SSH" {
		t.Errorf("expected name 'Allow SSH', got %q", result.Name)
	}
	if result.Action == nil || result.Action.Type != ActionAllow {
		t.Errorf("expected action type %q", ActionAllow)
	}
	if result.Action.AllowReturnTraffic == nil || !*result.Action.AllowReturnTraffic {
		t.Error("expected allowReturnTraffic to be true")
	}
	if !result.LoggingEnabled {
		t.Error("expected loggingEnabled to be true")
	}
}

func TestUpdateFirewallPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sites/site-1/firewall/policies/pol-123") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var req FirewallPolicy
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		req.ID = "pol-123"
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(req); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	result, err := c.UpdateFirewallPolicy(context.Background(), "site-1", "pol-123", &FirewallPolicy{
		Enabled:         false,
		Name:            "Updated Policy",
		Action:          &FirewallAction{Type: ActionReject},
		Source:          &FirewallEndpoint{ZoneID: "zone-lan"},
		Destination:     &FirewallEndpoint{ZoneID: "zone-wan"},
		IPProtocolScope: &IPProtocolScope{IPVersion: IPVersionIPv6},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Updated Policy" {
		t.Errorf("expected name 'Updated Policy', got %q", result.Name)
	}
	if result.Action == nil || result.Action.Type != ActionReject {
		t.Errorf("expected action type %q", ActionReject)
	}
	if result.IPProtocolScope == nil || result.IPProtocolScope.IPVersion != IPVersionIPv6 {
		t.Errorf("expected ipVersion %q", IPVersionIPv6)
	}
}

func TestDeleteFirewallPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sites/site-1/firewall/policies/pol-123") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	err := c.DeleteFirewallPolicy(context.Background(), "site-1", "pol-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Traffic Matching List CRUD ---

func TestCreateTrafficMatchingList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sites/site-1/traffic-matching-lists") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var req TrafficMatchingList
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if req.Name != "Blocked Ports" {
			t.Errorf("expected name 'Blocked Ports', got %q", req.Name)
		}
		if req.Type != TrafficMatchingPorts {
			t.Errorf("expected type %q, got %q", TrafficMatchingPorts, req.Type)
		}
		if len(req.Items) != 1 {
			t.Errorf("expected 1 item, got %d", len(req.Items))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(TrafficMatchingList{
			ID:    "tml-123",
			Type:  req.Type,
			Name:  req.Name,
			Items: req.Items,
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	port := 443
	c := NewClientForTesting("test-key", "host-id", server.URL)
	result, err := c.CreateTrafficMatchingList(context.Background(), "site-1", &TrafficMatchingList{
		Type: TrafficMatchingPorts,
		Name: "Blocked Ports",
		Items: []TrafficMatchingItem{
			{Type: "SINGLE_PORT", PortNumber: &port},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "tml-123" {
		t.Errorf("expected ID 'tml-123', got %q", result.ID)
	}
	if result.Name != "Blocked Ports" {
		t.Errorf("expected name 'Blocked Ports', got %q", result.Name)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(result.Items))
	}
}

func TestGetTrafficMatchingList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sites/site-1/traffic-matching-lists/tml-456") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(TrafficMatchingList{
			ID:   "tml-456",
			Type: TrafficMatchingIPv4Addresses,
			Name: "Server IPs",
			Items: []TrafficMatchingItem{
				{Type: "SUBNET", Subnet: "10.0.0.0/8"},
				{Type: "IP_RANGE", StartIP: "192.168.1.1", EndIP: "192.168.1.100"},
			},
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	result, err := c.GetTrafficMatchingList(context.Background(), "site-1", "tml-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "tml-456" {
		t.Errorf("expected ID 'tml-456', got %q", result.ID)
	}
	if result.Type != TrafficMatchingIPv4Addresses {
		t.Errorf("expected type %q, got %q", TrafficMatchingIPv4Addresses, result.Type)
	}
	if len(result.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(result.Items))
	}
	if result.Items[0].Subnet != "10.0.0.0/8" {
		t.Errorf("expected first item subnet '10.0.0.0/8', got %q", result.Items[0].Subnet)
	}
}

func TestUpdateTrafficMatchingList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sites/site-1/traffic-matching-lists/tml-123") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var req TrafficMatchingList
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		req.ID = "tml-123"
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(req); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	result, err := c.UpdateTrafficMatchingList(context.Background(), "site-1", "tml-123", &TrafficMatchingList{
		Type: TrafficMatchingIPv6Addresses,
		Name: "Updated IPv6 List",
		Items: []TrafficMatchingItem{
			{Type: "SUBNET", Subnet: "fd00::/64"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Updated IPv6 List" {
		t.Errorf("expected name 'Updated IPv6 List', got %q", result.Name)
	}
	if result.Type != TrafficMatchingIPv6Addresses {
		t.Errorf("expected type %q, got %q", TrafficMatchingIPv6Addresses, result.Type)
	}
}

func TestDeleteTrafficMatchingList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sites/site-1/traffic-matching-lists/tml-123") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	err := c.DeleteTrafficMatchingList(context.Background(), "site-1", "tml-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- DNS Policy CRUD ---

func TestCreateDnsPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sites/site-1/dns/policies") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var req DnsPolicy
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if req.Type != DnsPolicyARecord {
			t.Errorf("expected type %q, got %q", DnsPolicyARecord, req.Type)
		}
		if req.Domain != "app.example.com" {
			t.Errorf("expected domain 'app.example.com', got %q", req.Domain)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(DnsPolicy{
			ID:          "dns-123",
			Type:        req.Type,
			Enabled:     req.Enabled,
			Name:        req.Name,
			Domain:      req.Domain,
			IPv4Address: req.IPv4Address,
			TTLSeconds:  req.TTLSeconds,
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	ttl := 300
	c := NewClientForTesting("test-key", "host-id", server.URL)
	result, err := c.CreateDnsPolicy(context.Background(), "site-1", &DnsPolicy{
		Type:        DnsPolicyARecord,
		Enabled:     true,
		Name:        "App DNS",
		Domain:      "app.example.com",
		IPv4Address: "10.0.0.50",
		TTLSeconds:  &ttl,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "dns-123" {
		t.Errorf("expected ID 'dns-123', got %q", result.ID)
	}
	if result.Domain != "app.example.com" {
		t.Errorf("expected domain 'app.example.com', got %q", result.Domain)
	}
	if result.TTLSeconds == nil || *result.TTLSeconds != 300 {
		t.Errorf("expected TTL 300, got %v", result.TTLSeconds)
	}
}

func TestGetDnsPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sites/site-1/dns/policies/dns-456") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		ttl := 600
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(DnsPolicy{
			ID:         "dns-456",
			Type:       DnsPolicyForwardDomain,
			Enabled:    true,
			Name:       "Forward Corp",
			Domain:     "corp.example.com",
			ForwardTo:  []string{"8.8.8.8", "8.8.4.4"},
			TTLSeconds: &ttl,
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	result, err := c.GetDnsPolicy(context.Background(), "site-1", "dns-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "dns-456" {
		t.Errorf("expected ID 'dns-456', got %q", result.ID)
	}
	if result.Type != DnsPolicyForwardDomain {
		t.Errorf("expected type %q, got %q", DnsPolicyForwardDomain, result.Type)
	}
	if len(result.ForwardTo) != 2 {
		t.Errorf("expected 2 forwardTo entries, got %d", len(result.ForwardTo))
	}
	if result.ForwardTo[0] != "8.8.8.8" {
		t.Errorf("expected first forwardTo '8.8.8.8', got %q", result.ForwardTo[0])
	}
}

func TestUpdateDnsPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sites/site-1/dns/policies/dns-123") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var req DnsPolicy
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		req.ID = "dns-123"
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(req); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	result, err := c.UpdateDnsPolicy(context.Background(), "site-1", "dns-123", &DnsPolicy{
		Type:    DnsPolicyCNAMERecord,
		Enabled: true,
		Name:    "Updated CNAME",
		Domain:  "alias.example.com",
		Target:  "real.example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Updated CNAME" {
		t.Errorf("expected name 'Updated CNAME', got %q", result.Name)
	}
	if result.Target != "real.example.com" {
		t.Errorf("expected target 'real.example.com', got %q", result.Target)
	}
}

func TestDeleteDnsPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sites/site-1/dns/policies/dns-123") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	err := c.DeleteDnsPolicy(context.Background(), "site-1", "dns-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Hotspot Voucher operations ---

func TestCreateHotspotVoucher(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sites/site-1/hotspot/vouchers") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var req HotspotVoucherCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if req.Name != "Guest Pass" {
			t.Errorf("expected name 'Guest Pass', got %q", req.Name)
		}
		if req.TimeLimitMinutes != 1440 {
			t.Errorf("expected timeLimitMinutes 1440, got %d", req.TimeLimitMinutes)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode([]HotspotVoucher{
			{
				ID:               "voucher-1",
				Name:             req.Name,
				Code:             "ABCD-1234",
				TimeLimitMinutes: req.TimeLimitMinutes,
			},
			{
				ID:               "voucher-2",
				Name:             req.Name,
				Code:             "EFGH-5678",
				TimeLimitMinutes: req.TimeLimitMinutes,
			},
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	guestLimit := 1
	c := NewClientForTesting("test-key", "host-id", server.URL)
	result, err := c.CreateHotspotVoucher(context.Background(), "site-1", &HotspotVoucherCreateRequest{
		Name:                 "Guest Pass",
		Count:                2,
		TimeLimitMinutes:     1440,
		AuthorizedGuestLimit: &guestLimit,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 vouchers, got %d", len(result))
	}
	if result[0].ID != "voucher-1" {
		t.Errorf("expected first voucher ID 'voucher-1', got %q", result[0].ID)
	}
	if result[0].Code != "ABCD-1234" {
		t.Errorf("expected first voucher code 'ABCD-1234', got %q", result[0].Code)
	}
	if result[1].ID != "voucher-2" {
		t.Errorf("expected second voucher ID 'voucher-2', got %q", result[1].ID)
	}
}

func TestGetHotspotVoucher(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sites/site-1/hotspot/vouchers/voucher-456") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		expired := false
		dataLimit := 1024
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(HotspotVoucher{
			ID:                   "voucher-456",
			Name:                 "Day Pass",
			Code:                 "WXYZ-9999",
			TimeLimitMinutes:     1440,
			DataUsageLimitMBytes: &dataLimit,
			Expired:              &expired,
			CreatedAt:            "2024-06-01T00:00:00Z",
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	result, err := c.GetHotspotVoucher(context.Background(), "site-1", "voucher-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "voucher-456" {
		t.Errorf("expected ID 'voucher-456', got %q", result.ID)
	}
	if result.Code != "WXYZ-9999" {
		t.Errorf("expected code 'WXYZ-9999', got %q", result.Code)
	}
	if result.TimeLimitMinutes != 1440 {
		t.Errorf("expected timeLimitMinutes 1440, got %d", result.TimeLimitMinutes)
	}
	if result.DataUsageLimitMBytes == nil || *result.DataUsageLimitMBytes != 1024 {
		t.Errorf("expected dataUsageLimitMBytes 1024, got %v", result.DataUsageLimitMBytes)
	}
	if result.Expired == nil || *result.Expired != false {
		t.Error("expected expired to be false")
	}
}

func TestDeleteHotspotVoucher(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sites/site-1/hotspot/vouchers/voucher-123") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	err := c.DeleteHotspotVoucher(context.Background(), "site-1", "voucher-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Read-only data sources ---

func TestListClients(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("limit") != defaultPageSize {
			t.Errorf("expected limit=%s, got %q", defaultPageSize, r.URL.Query().Get("limit"))
		}
		if !strings.Contains(r.URL.Path, "/sites/site-1/clients") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(PaginatedResponse[NetworkClient]{
			Offset: 0, Limit: 200, Count: 2, TotalCount: 2,
			Data: []NetworkClient{
				{ID: "client-1", Type: "WIRED", Name: "Desktop", IPAddress: "192.168.1.10"},
				{ID: "client-2", Type: "WIRELESS", Name: "Laptop", IPAddress: "192.168.1.20"},
			},
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	clients, err := c.ListClients(context.Background(), "site-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clients) != 2 {
		t.Errorf("expected 2 clients, got %d", len(clients))
	}
	if clients[0].ID != "client-1" {
		t.Errorf("expected first client ID 'client-1', got %q", clients[0].ID)
	}
	if clients[0].Type != "WIRED" {
		t.Errorf("expected first client type 'WIRED', got %q", clients[0].Type)
	}
	if clients[1].IPAddress != "192.168.1.20" {
		t.Errorf("expected second client IP '192.168.1.20', got %q", clients[1].IPAddress)
	}
}

func TestListWans(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("limit") != defaultPageSize {
			t.Errorf("expected limit=%s, got %q", defaultPageSize, r.URL.Query().Get("limit"))
		}
		if !strings.Contains(r.URL.Path, "/sites/site-1/wans") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(PaginatedResponse[Wan]{
			Offset: 0, Limit: 200, Count: 2, TotalCount: 2,
			Data: []Wan{
				{ID: "wan-1", Name: "WAN 1"},
				{ID: "wan-2", Name: "WAN 2"},
			},
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	wans, err := c.ListWans(context.Background(), "site-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(wans) != 2 {
		t.Errorf("expected 2 WANs, got %d", len(wans))
	}
	if wans[0].ID != "wan-1" {
		t.Errorf("expected first WAN ID 'wan-1', got %q", wans[0].ID)
	}
	if wans[1].Name != "WAN 2" {
		t.Errorf("expected second WAN name 'WAN 2', got %q", wans[1].Name)
	}
}

func TestListVpnServers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("limit") != defaultPageSize {
			t.Errorf("expected limit=%s, got %q", defaultPageSize, r.URL.Query().Get("limit"))
		}
		if !strings.Contains(r.URL.Path, "/sites/site-1/vpn/servers") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(PaginatedResponse[VpnServer]{
			Offset: 0, Limit: 200, Count: 1, TotalCount: 1,
			Data: []VpnServer{
				{ID: "vpn-srv-1", Type: "WIREGUARD", Name: "WireGuard Server", Enabled: true},
			},
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	servers, err := c.ListVpnServers(context.Background(), "site-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(servers) != 1 {
		t.Errorf("expected 1 VPN server, got %d", len(servers))
	}
	if servers[0].ID != "vpn-srv-1" {
		t.Errorf("expected VPN server ID 'vpn-srv-1', got %q", servers[0].ID)
	}
	if servers[0].Type != "WIREGUARD" {
		t.Errorf("expected VPN server type 'WIREGUARD', got %q", servers[0].Type)
	}
	if !servers[0].Enabled {
		t.Error("expected VPN server to be enabled")
	}
}

func TestListVpnTunnels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("limit") != defaultPageSize {
			t.Errorf("expected limit=%s, got %q", defaultPageSize, r.URL.Query().Get("limit"))
		}
		if !strings.Contains(r.URL.Path, "/sites/site-1/vpn/site-to-site-tunnels") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(PaginatedResponse[VpnTunnel]{
			Offset: 0, Limit: 200, Count: 2, TotalCount: 2,
			Data: []VpnTunnel{
				{ID: "tunnel-1", Type: "IPSEC", Name: "Office Tunnel"},
				{ID: "tunnel-2", Type: "WIREGUARD", Name: "Remote Tunnel"},
			},
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	tunnels, err := c.ListVpnTunnels(context.Background(), "site-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tunnels) != 2 {
		t.Errorf("expected 2 VPN tunnels, got %d", len(tunnels))
	}
	if tunnels[0].ID != "tunnel-1" {
		t.Errorf("expected first tunnel ID 'tunnel-1', got %q", tunnels[0].ID)
	}
	if tunnels[0].Type != "IPSEC" {
		t.Errorf("expected first tunnel type 'IPSEC', got %q", tunnels[0].Type)
	}
	if tunnels[1].Name != "Remote Tunnel" {
		t.Errorf("expected second tunnel name 'Remote Tunnel', got %q", tunnels[1].Name)
	}
}

func TestListRadiusProfiles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("limit") != defaultPageSize {
			t.Errorf("expected limit=%s, got %q", defaultPageSize, r.URL.Query().Get("limit"))
		}
		if !strings.Contains(r.URL.Path, "/sites/site-1/radius/profiles") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(PaginatedResponse[RadiusProfile]{
			Offset: 0, Limit: 200, Count: 2, TotalCount: 2,
			Data: []RadiusProfile{
				{ID: "radius-1", Name: "Corp RADIUS"},
				{ID: "radius-2", Name: "Guest RADIUS"},
			},
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	profiles, err := c.ListRadiusProfiles(context.Background(), "site-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(profiles) != 2 {
		t.Errorf("expected 2 RADIUS profiles, got %d", len(profiles))
	}
	if profiles[0].ID != "radius-1" {
		t.Errorf("expected first profile ID 'radius-1', got %q", profiles[0].ID)
	}
	if profiles[1].Name != "Guest RADIUS" {
		t.Errorf("expected second profile name 'Guest RADIUS', got %q", profiles[1].Name)
	}
}

func TestListDeviceTags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("limit") != defaultPageSize {
			t.Errorf("expected limit=%s, got %q", defaultPageSize, r.URL.Query().Get("limit"))
		}
		if !strings.Contains(r.URL.Path, "/sites/site-1/device-tags") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(PaginatedResponse[DeviceTag]{
			Offset: 0, Limit: 200, Count: 2, TotalCount: 2,
			Data: []DeviceTag{
				{ID: "tag-1", Name: "Floor 1 APs", DeviceIDs: []string{"dev-1", "dev-2"}},
				{ID: "tag-2", Name: "Floor 2 APs", DeviceIDs: []string{"dev-3"}},
			},
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	tags, err := c.ListDeviceTags(context.Background(), "site-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 2 {
		t.Errorf("expected 2 device tags, got %d", len(tags))
	}
	if tags[0].ID != "tag-1" {
		t.Errorf("expected first tag ID 'tag-1', got %q", tags[0].ID)
	}
	if len(tags[0].DeviceIDs) != 2 {
		t.Errorf("expected first tag to have 2 device IDs, got %d", len(tags[0].DeviceIDs))
	}
	if tags[1].Name != "Floor 2 APs" {
		t.Errorf("expected second tag name 'Floor 2 APs', got %q", tags[1].Name)
	}
}

func TestListPendingDevices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("limit") != defaultPageSize {
			t.Errorf("expected limit=%s, got %q", defaultPageSize, r.URL.Query().Get("limit"))
		}
		if !strings.Contains(r.URL.Path, "/pending-devices") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(PaginatedResponse[PendingDevice]{
			Offset: 0, Limit: 200, Count: 1, TotalCount: 1,
			Data: []PendingDevice{
				{
					MacAddress:        "11:22:33:44:55:66",
					IPAddress:         "192.168.1.50",
					Model:             "U6-LR",
					State:             "PENDING",
					Supported:         true,
					FirmwareVersion:   "7.0.0",
					FirmwareUpdatable: true,
					Features:          []string{"access_point"},
				},
			},
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	devices, err := c.ListPendingDevices(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(devices) != 1 {
		t.Errorf("expected 1 pending device, got %d", len(devices))
	}
	if devices[0].MacAddress != "11:22:33:44:55:66" {
		t.Errorf("expected MAC '11:22:33:44:55:66', got %q", devices[0].MacAddress)
	}
	if devices[0].Model != "U6-LR" {
		t.Errorf("expected model 'U6-LR', got %q", devices[0].Model)
	}
	if !devices[0].Supported {
		t.Error("expected device to be supported")
	}
	if len(devices[0].Features) != 1 || devices[0].Features[0] != "access_point" {
		t.Errorf("expected features ['access_point'], got %v", devices[0].Features)
	}
}

func TestListDpiCategories(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("limit") != defaultPageSize {
			t.Errorf("expected limit=%s, got %q", defaultPageSize, r.URL.Query().Get("limit"))
		}
		if !strings.Contains(r.URL.Path, "/dpi/categories") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(PaginatedResponse[DpiCategory]{
			Offset: 0, Limit: 200, Count: 3, TotalCount: 3,
			Data: []DpiCategory{
				{ID: 1, Name: "Streaming"},
				{ID: 2, Name: "Social Media"},
				{ID: 3, Name: "Gaming"},
			},
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	categories, err := c.ListDpiCategories(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(categories) != 3 {
		t.Errorf("expected 3 DPI categories, got %d", len(categories))
	}
	if categories[0].ID != 1 {
		t.Errorf("expected first category ID 1, got %d", categories[0].ID)
	}
	if categories[0].Name != "Streaming" {
		t.Errorf("expected first category name 'Streaming', got %q", categories[0].Name)
	}
	if categories[2].Name != "Gaming" {
		t.Errorf("expected third category name 'Gaming', got %q", categories[2].Name)
	}
}

func TestListDpiApplications(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("limit") != defaultPageSize {
			t.Errorf("expected limit=%s, got %q", defaultPageSize, r.URL.Query().Get("limit"))
		}
		if !strings.Contains(r.URL.Path, "/dpi/applications") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(PaginatedResponse[DpiApplication]{
			Offset: 0, Limit: 200, Count: 3, TotalCount: 3,
			Data: []DpiApplication{
				{ID: 101, Name: "Netflix"},
				{ID: 102, Name: "YouTube"},
				{ID: 103, Name: "Spotify"},
			},
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer server.Close()

	c := NewClientForTesting("key", "host", server.URL)
	apps, err := c.ListDpiApplications(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(apps) != 3 {
		t.Errorf("expected 3 DPI applications, got %d", len(apps))
	}
	if apps[0].ID != 101 {
		t.Errorf("expected first app ID 101, got %d", apps[0].ID)
	}
	if apps[0].Name != "Netflix" {
		t.Errorf("expected first app name 'Netflix', got %q", apps[0].Name)
	}
	if apps[2].Name != "Spotify" {
		t.Errorf("expected third app name 'Spotify', got %q", apps[2].Name)
	}
}
