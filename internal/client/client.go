package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultBaseURL  = "https://api.ui.com"
	apiVersion      = "v1"
	defaultPageSize = "200"
)

// Management types for networks.
const (
	ManagementUnmanaged = "UNMANAGED"
	ManagementGateway   = "GATEWAY"
	ManagementSwitch    = "SWITCH"
)

// WiFi broadcast types.
const (
	BroadcastTypeStandard     = "STANDARD"
	BroadcastTypeIoTOptimized = "IOT_OPTIMIZED"
)

// WiFi security types.
const (
	SecurityOpen               = "OPEN"
	SecurityWPA2Personal       = "WPA2_PERSONAL"
	SecurityWPA3Personal       = "WPA3_PERSONAL"
	SecurityWPA2WPA3Personal   = "WPA2_WPA3_PERSONAL"
	SecurityWPA2Enterprise     = "WPA2_ENTERPRISE"
	SecurityWPA3Enterprise     = "WPA3_ENTERPRISE"
	SecurityWPA2WPA3Enterprise = "WPA2_WPA3_ENTERPRISE"
)

// Network assignment types for WiFi broadcasts.
const (
	NetworkTypeNative   = "NATIVE"
	NetworkTypeSpecific = "SPECIFIC"
)

// Client manages communication with the UniFi Network API.
type Client struct {
	baseURL    string
	apiKey     string
	hostID     string
	httpClient *http.Client
}

// NewClient creates a new UniFi API client.
// The hostID is the console's Host ID used for cloud connector proxying.
func NewClient(apiKey, hostID string) *Client {
	return &Client{
		baseURL: defaultBaseURL,
		apiKey:  apiKey,
		hostID:  hostID,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// APIError represents an error response from the UniFi API.
type APIError struct {
	StatusCode  int    `json:"statusCode"`
	StatusName  string `json:"statusName"`
	Code        string `json:"code"`
	Message     string `json:"message"`
	Timestamp   string `json:"timestamp"`
	RequestPath string `json:"requestPath"`
	RequestID   string `json:"requestId"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("UniFi API error %d (%s): %s [code=%s, requestId=%s]",
		e.StatusCode, e.StatusName, e.Message, e.Code, e.RequestID)
}

// IsNotFound returns true if the error is a 404 Not Found API error.
func IsNotFound(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}

// PaginatedResponse wraps paginated list responses.
type PaginatedResponse[T any] struct {
	Offset     int64 `json:"offset"`
	Limit      int32 `json:"limit"`
	Count      int32 `json:"count"`
	TotalCount int64 `json:"totalCount"`
	Data       []T   `json:"data"`
}

// buildURL constructs the proxied API URL via cloud connector.
func (c *Client) buildURL(path string) string {
	return fmt.Sprintf("%s/%s/connector/consoles/%s/%s/%s",
		c.baseURL, apiVersion, c.hostID, apiVersion, path)
}

func (c *Client) do(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	fullURL := c.buildURL(path)

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr APIError
		if err := json.Unmarshal(respBody, &apiErr); err != nil {
			return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
		}
		return &apiErr
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unmarshaling response: %w", err)
		}
	}

	return nil
}

// doList performs a paginated GET request, collecting all pages.
func (c *Client) doList(ctx context.Context, path string, result interface{}) error {
	fullURL := c.buildURL(path)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")

	q := req.URL.Query()
	q.Set("limit", defaultPageSize)
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr APIError
		if err := json.Unmarshal(body, &apiErr); err != nil {
			return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
		}
		return &apiErr
	}

	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("unmarshaling response: %w", err)
	}

	return nil
}

// --- Site operations ---

type Site struct {
	ID                string `json:"id"`
	InternalReference string `json:"internalReference"`
	Name              string `json:"name"`
}

func (c *Client) ListSites(ctx context.Context) ([]Site, error) {
	var resp PaginatedResponse[Site]
	if err := c.doList(ctx, "sites", &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// --- Network operations ---

type Network struct {
	ID           string          `json:"id,omitempty"`
	Name         string          `json:"name"`
	Management   string          `json:"management"`
	Enabled      bool            `json:"enabled"`
	VlanID       int             `json:"vlanId"`
	Metadata     *EntityMetadata `json:"metadata,omitempty"`
	DhcpGuarding *DhcpGuarding   `json:"dhcpGuarding,omitempty"`
}

type EntityMetadata struct {
	Origin string `json:"origin"`
}

type DhcpGuarding struct {
	TrustedDhcpServerIPAddresses []string `json:"trustedDhcpServerIpAddresses"`
}

func (c *Client) CreateNetwork(ctx context.Context, siteID string, network *Network) (*Network, error) {
	var result Network
	path := fmt.Sprintf("sites/%s/networks", siteID)
	err := c.do(ctx, http.MethodPost, path, network, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetNetwork(ctx context.Context, siteID, networkID string) (*Network, error) {
	var result Network
	path := fmt.Sprintf("sites/%s/networks/%s", siteID, networkID)
	err := c.do(ctx, http.MethodGet, path, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) UpdateNetwork(ctx context.Context, siteID, networkID string, network *Network) (*Network, error) {
	var result Network
	path := fmt.Sprintf("sites/%s/networks/%s", siteID, networkID)
	err := c.do(ctx, http.MethodPut, path, network, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) DeleteNetwork(ctx context.Context, siteID, networkID string) error {
	path := fmt.Sprintf("sites/%s/networks/%s", siteID, networkID)
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

func (c *Client) ListNetworks(ctx context.Context, siteID string) ([]Network, error) {
	var resp PaginatedResponse[Network]
	path := fmt.Sprintf("sites/%s/networks", siteID)
	if err := c.doList(ctx, path, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// --- WiFi Broadcast operations ---

type WifiBroadcast struct {
	ID                                  string                 `json:"id,omitempty"`
	Type                                string                 `json:"type"`
	Name                                string                 `json:"name"`
	Enabled                             bool                   `json:"enabled"`
	SecurityConfiguration               *SecurityConfiguration `json:"securityConfiguration"`
	Network                             *BroadcastNetwork      `json:"network"`
	ClientIsolationEnabled              bool                   `json:"clientIsolationEnabled"`
	HideName                            bool                   `json:"hideName"`
	MulticastToUnicastConversionEnabled bool                   `json:"multicastToUnicastConversionEnabled"`
	UapsdEnabled                        bool                   `json:"uapsdEnabled"`
	Metadata                            *EntityMetadata        `json:"metadata,omitempty"`
}

type SecurityConfiguration struct {
	Type       string `json:"type"`
	Passphrase string `json:"passphrase,omitempty"`
}

type BroadcastNetwork struct {
	Type      string `json:"type"`
	NetworkID string `json:"networkId,omitempty"`
}

func (c *Client) CreateWifiBroadcast(ctx context.Context, siteID string, broadcast *WifiBroadcast) (*WifiBroadcast, error) {
	var result WifiBroadcast
	path := fmt.Sprintf("sites/%s/wifi/broadcasts", siteID)
	err := c.do(ctx, http.MethodPost, path, broadcast, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetWifiBroadcast(ctx context.Context, siteID, broadcastID string) (*WifiBroadcast, error) {
	var result WifiBroadcast
	path := fmt.Sprintf("sites/%s/wifi/broadcasts/%s", siteID, broadcastID)
	err := c.do(ctx, http.MethodGet, path, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) UpdateWifiBroadcast(ctx context.Context, siteID, broadcastID string, broadcast *WifiBroadcast) (*WifiBroadcast, error) {
	var result WifiBroadcast
	path := fmt.Sprintf("sites/%s/wifi/broadcasts/%s", siteID, broadcastID)
	err := c.do(ctx, http.MethodPut, path, broadcast, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) DeleteWifiBroadcast(ctx context.Context, siteID, broadcastID string) error {
	path := fmt.Sprintf("sites/%s/wifi/broadcasts/%s", siteID, broadcastID)
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// --- Firewall Zone operations ---

type FirewallZone struct {
	ID         string          `json:"id,omitempty"`
	Name       string          `json:"name"`
	NetworkIDs []string        `json:"networkIds"`
	Metadata   *EntityMetadata `json:"metadata,omitempty"`
}

func (c *Client) CreateFirewallZone(ctx context.Context, siteID string, zone *FirewallZone) (*FirewallZone, error) {
	var result FirewallZone
	path := fmt.Sprintf("sites/%s/firewall/zones", siteID)
	err := c.do(ctx, http.MethodPost, path, zone, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetFirewallZone(ctx context.Context, siteID, zoneID string) (*FirewallZone, error) {
	var result FirewallZone
	path := fmt.Sprintf("sites/%s/firewall/zones/%s", siteID, zoneID)
	err := c.do(ctx, http.MethodGet, path, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) UpdateFirewallZone(ctx context.Context, siteID, zoneID string, zone *FirewallZone) (*FirewallZone, error) {
	var result FirewallZone
	path := fmt.Sprintf("sites/%s/firewall/zones/%s", siteID, zoneID)
	err := c.do(ctx, http.MethodPut, path, zone, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) DeleteFirewallZone(ctx context.Context, siteID, zoneID string) error {
	path := fmt.Sprintf("sites/%s/firewall/zones/%s", siteID, zoneID)
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// --- Device operations ---

type Device struct {
	ID                string          `json:"id"`
	MacAddress        string          `json:"macAddress"`
	IPAddress         string          `json:"ipAddress"`
	Name              string          `json:"name"`
	Model             string          `json:"model"`
	Supported         bool            `json:"supported"`
	State             string          `json:"state"`
	FirmwareVersion   string          `json:"firmwareVersion"`
	FirmwareUpdatable bool            `json:"firmwareUpdatable"`
	AdoptedAt         string          `json:"adoptedAt"`
	ProvisionedAt     string          `json:"provisionedAt"`
	ConfigurationID   string          `json:"configurationId"`
	Uplink            *DeviceUplink   `json:"uplink,omitempty"`
	Features          *DeviceFeatures `json:"features,omitempty"`
}

type DeviceUplink struct {
	DeviceID string `json:"deviceId"`
}

type DeviceFeatures struct {
	Switching   *json.RawMessage `json:"switching,omitempty"`
	AccessPoint *json.RawMessage `json:"accessPoint,omitempty"`
}

func (c *Client) GetDevice(ctx context.Context, siteID, deviceID string) (*Device, error) {
	var result Device
	path := fmt.Sprintf("sites/%s/devices/%s", siteID, deviceID)
	err := c.do(ctx, http.MethodGet, path, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) ListDevices(ctx context.Context, siteID string) ([]Device, error) {
	var resp PaginatedResponse[Device]
	path := fmt.Sprintf("sites/%s/devices", siteID)
	if err := c.doList(ctx, path, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}
