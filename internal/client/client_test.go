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
		json.NewEncoder(w).Encode(Network{
			ID:         "net-123",
			Name:       req.Name,
			Management: req.Management,
			Enabled:    req.Enabled,
			VlanID:     req.VlanID,
		})
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
		json.NewEncoder(w).Encode(Network{
			ID:         "net-456",
			Name:       "My Network",
			Management: "GATEWAY",
			Enabled:    true,
			VlanID:     200,
		})
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
		json.NewEncoder(w).Encode(APIError{
			StatusCode: 404,
			StatusName: "NOT_FOUND",
			Code:       "api.resource.not-found",
			Message:    "Network not found",
			RequestID:  "req-789",
		})
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
		json.NewDecoder(r.Body).Decode(&req)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(WifiBroadcast{
			ID:      "wifi-123",
			Type:    req.Type,
			Name:    req.Name,
			Enabled: req.Enabled,
		})
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
		json.NewDecoder(r.Body).Decode(&req)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(FirewallZone{
			ID:         "zone-123",
			Name:       req.Name,
			NetworkIDs: req.NetworkIDs,
		})
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
		json.NewEncoder(w).Encode(PaginatedResponse[Network]{
			Offset:     0,
			Limit:      200,
			Count:      2,
			TotalCount: 2,
			Data: []Network{
				{ID: "net-1", Name: "Network 1", Management: ManagementUnmanaged, VlanID: 100},
				{ID: "net-2", Name: "Network 2", Management: ManagementGateway, VlanID: 200},
			},
		})
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
		json.NewEncoder(w).Encode(PaginatedResponse[Site]{
			Offset:     0,
			Limit:      200,
			Count:      1,
			TotalCount: 1,
			Data: []Site{
				{ID: "site-1", Name: "Default", InternalReference: "default"},
			},
		})
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
