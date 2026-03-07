package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

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

func TestBuildURL(t *testing.T) {
	c := NewClient("key", "my-host-id")
	url := c.buildURL("sites/abc/networks")
	expected := "https://api.ui.com/v1/connector/consoles/my-host-id/v1/sites/abc/networks"
	if url != expected {
		t.Errorf("expected %q, got %q", expected, url)
	}
}

func TestCreateNetwork(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("X-API-Key") != "test-key" {
			t.Errorf("expected X-API-Key 'test-key', got %q", r.Header.Get("X-API-Key"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type 'application/json', got %q", r.Header.Get("Content-Type"))
		}

		var req Network
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if req.Name != "Test Net" {
			t.Errorf("expected name 'Test Net', got %q", req.Name)
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

	c := NewClient("test-key", "host-id")
	c.baseURL = server.URL

	result, err := c.CreateNetwork(context.Background(), "site-1", &Network{
		Name:       "Test Net",
		Management: "UNMANAGED",
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

func TestGetNetwork(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(Network{
			ID:         "net-456",
			Name:       "My Network",
			Management: "GATEWAY",
			Enabled:    true,
			VlanID:     200,
		}); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	c := NewClient("test-key", "host-id")
	c.baseURL = server.URL

	result, err := c.GetNetwork(context.Background(), "site-1", "net-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "net-456" {
		t.Errorf("expected ID 'net-456', got %q", result.ID)
	}
}

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
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	c := NewClient("test-key", "host-id")
	c.baseURL = server.URL

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
}

func TestCreateWifiBroadcast(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req WifiBroadcast
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(WifiBroadcast{
			ID:      "wifi-123",
			Type:    req.Type,
			Name:    req.Name,
			Enabled: req.Enabled,
		}); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	c := NewClient("test-key", "host-id")
	c.baseURL = server.URL

	result, err := c.CreateWifiBroadcast(context.Background(), "site-1", &WifiBroadcast{
		Type:    "STANDARD",
		Name:    "TestSSID",
		Enabled: true,
		SecurityConfiguration: &SecurityConfiguration{
			Type:       "WPA2_PERSONAL",
			Passphrase: "password123",
		},
		Network: &BroadcastNetwork{
			Type: "NATIVE",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "wifi-123" {
		t.Errorf("expected ID 'wifi-123', got %q", result.ID)
	}
}

func TestCreateFirewallZone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req FirewallZone
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(FirewallZone{
			ID:         "zone-123",
			Name:       req.Name,
			NetworkIDs: req.NetworkIDs,
		}); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	c := NewClient("test-key", "host-id")
	c.baseURL = server.URL

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
}

func TestDeleteNetwork(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClient("test-key", "host-id")
	c.baseURL = server.URL

	err := c.DeleteNetwork(context.Background(), "site-1", "net-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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

	if IsNotFound(nil) {
		t.Error("expected IsNotFound to return false for nil")
	}

	genericErr := fmt.Errorf("some error")
	if IsNotFound(genericErr) {
		t.Error("expected IsNotFound to return false for non-APIError")
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
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	c := NewClient("test-key", "host-id")
	c.baseURL = server.URL

	networks, err := c.ListNetworks(context.Background(), "site-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(networks) != 2 {
		t.Errorf("expected 2 networks, got %d", len(networks))
	}
}

func TestListSites(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	c := NewClient("test-key", "host-id")
	c.baseURL = server.URL

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
}

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
	// Verify constants match expected API values
	if ManagementUnmanaged != "UNMANAGED" {
		t.Errorf("unexpected value: %s", ManagementUnmanaged)
	}
	if BroadcastTypeStandard != "STANDARD" {
		t.Errorf("unexpected value: %s", BroadcastTypeStandard)
	}
	if SecurityWPA2Personal != "WPA2_PERSONAL" {
		t.Errorf("unexpected value: %s", SecurityWPA2Personal)
	}
	if NetworkTypeNative != "NATIVE" {
		t.Errorf("unexpected value: %s", NetworkTypeNative)
	}
}

func TestAPIErrorString(t *testing.T) {
	err := &APIError{
		StatusCode: 404,
		StatusName: "NOT_FOUND",
		Message:    "Resource not found",
		Code:       "api.resource.not-found",
		RequestID:  "req-abc",
	}
	got := err.Error()
	expected := "UniFi API error 404 (NOT_FOUND): Resource not found [code=api.resource.not-found, requestId=req-abc]"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestUpdateNetwork(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}

		var req Network
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
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

	c := NewClient("test-key", "host-id")
	c.baseURL = server.URL

	result, err := c.UpdateNetwork(context.Background(), "site-1", "net-123", &Network{
		Name:       "Updated Net",
		Management: ManagementGateway,
		Enabled:    false,
		VlanID:     200,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Updated Net" {
		t.Errorf("expected name 'Updated Net', got %q", result.Name)
	}
	if result.VlanID != 200 {
		t.Errorf("expected vlanID 200, got %d", result.VlanID)
	}
}

func TestGetWifiBroadcast(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(WifiBroadcast{
			ID:      "wifi-456",
			Type:    BroadcastTypeStandard,
			Name:    "My WiFi",
			Enabled: true,
			SecurityConfiguration: &SecurityConfiguration{
				Type: SecurityWPA2Personal,
			},
			Network: &BroadcastNetwork{
				Type: NetworkTypeNative,
			},
		}); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	c := NewClient("test-key", "host-id")
	c.baseURL = server.URL

	result, err := c.GetWifiBroadcast(context.Background(), "site-1", "wifi-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "wifi-456" {
		t.Errorf("expected ID 'wifi-456', got %q", result.ID)
	}
	if result.SecurityConfiguration.Type != SecurityWPA2Personal {
		t.Errorf("expected security type %q, got %q", SecurityWPA2Personal, result.SecurityConfiguration.Type)
	}
}

func TestUpdateWifiBroadcast(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}

		var req WifiBroadcast
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(WifiBroadcast{
			ID:   "wifi-456",
			Name: req.Name,
			Type: req.Type,
		}); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	c := NewClient("test-key", "host-id")
	c.baseURL = server.URL

	result, err := c.UpdateWifiBroadcast(context.Background(), "site-1", "wifi-456", &WifiBroadcast{
		Name: "Updated WiFi",
		Type: BroadcastTypeStandard,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Updated WiFi" {
		t.Errorf("expected name 'Updated WiFi', got %q", result.Name)
	}
}

func TestDeleteWifiBroadcast(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClient("test-key", "host-id")
	c.baseURL = server.URL

	err := c.DeleteWifiBroadcast(context.Background(), "site-1", "wifi-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetFirewallZone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(FirewallZone{
			ID:         "zone-456",
			Name:       "My Zone",
			NetworkIDs: []string{"net-1", "net-2"},
		}); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	c := NewClient("test-key", "host-id")
	c.baseURL = server.URL

	result, err := c.GetFirewallZone(context.Background(), "site-1", "zone-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "zone-456" {
		t.Errorf("expected ID 'zone-456', got %q", result.ID)
	}
	if len(result.NetworkIDs) != 2 {
		t.Errorf("expected 2 network IDs, got %d", len(result.NetworkIDs))
	}
}

func TestUpdateFirewallZone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}

		var req FirewallZone
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(FirewallZone{
			ID:         "zone-456",
			Name:       req.Name,
			NetworkIDs: req.NetworkIDs,
		}); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	c := NewClient("test-key", "host-id")
	c.baseURL = server.URL

	result, err := c.UpdateFirewallZone(context.Background(), "site-1", "zone-456", &FirewallZone{
		Name:       "Updated Zone",
		NetworkIDs: []string{"net-3"},
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
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClient("test-key", "host-id")
	c.baseURL = server.URL

	err := c.DeleteFirewallZone(context.Background(), "site-1", "zone-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetDevice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(Device{
			ID:              "dev-123",
			MacAddress:      "aa:bb:cc:dd:ee:ff",
			IPAddress:       "192.168.1.1",
			Name:            "Switch",
			Model:           "USW-24",
			Supported:       true,
			State:           "ONLINE",
			FirmwareVersion: "7.0.0",
			Uplink:          &DeviceUplink{DeviceID: "dev-parent"},
		}); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	c := NewClient("test-key", "host-id")
	c.baseURL = server.URL

	result, err := c.GetDevice(context.Background(), "site-1", "dev-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "dev-123" {
		t.Errorf("expected ID 'dev-123', got %q", result.ID)
	}
	if result.MacAddress != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("expected mac 'aa:bb:cc:dd:ee:ff', got %q", result.MacAddress)
	}
	if result.Uplink == nil || result.Uplink.DeviceID != "dev-parent" {
		t.Error("expected uplink device ID 'dev-parent'")
	}
}

func TestListDevices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(PaginatedResponse[Device]{
			Offset:     0,
			Limit:      200,
			Count:      2,
			TotalCount: 2,
			Data: []Device{
				{ID: "dev-1", Name: "Switch 1"},
				{ID: "dev-2", Name: "AP 1"},
			},
		}); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	c := NewClient("test-key", "host-id")
	c.baseURL = server.URL

	devices, err := c.ListDevices(context.Background(), "site-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(devices) != 2 {
		t.Errorf("expected 2 devices, got %d", len(devices))
	}
}

func TestNonJSONErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	c := NewClient("test-key", "host-id")
	c.baseURL = server.URL

	_, err := c.GetNetwork(context.Background(), "site-1", "net-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Should not be an APIError since the response wasn't JSON
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		t.Error("expected non-APIError for non-JSON error response")
	}
}

func TestAPIErrorOnCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIError{
			StatusCode: 400,
			StatusName: "BAD_REQUEST",
			Code:       "api.validation.error",
			Message:    "Invalid VLAN ID",
			RequestID:  "req-999",
		})
	}))
	defer server.Close()

	c := NewClient("test-key", "host-id")
	c.baseURL = server.URL

	_, err := c.CreateNetwork(context.Background(), "site-1", &Network{
		Name:   "Bad Net",
		VlanID: 9999,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 400 {
		t.Errorf("expected status 400, got %d", apiErr.StatusCode)
	}
}
