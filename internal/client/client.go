package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

// IsPersonalSecurityType returns true if the security type requires a passphrase.
func IsPersonalSecurityType(t string) bool {
	switch t {
	case SecurityWPA2Personal, SecurityWPA3Personal, SecurityWPA2WPA3Personal:
		return true
	}
	return false
}

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

// SetBaseURL overrides the default API base URL. This is intended for testing.
func (c *Client) SetBaseURL(url string) {
	c.baseURL = url
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
	var apiErr *APIError
	if errors.As(err, &apiErr) {
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

// executeRequest sends an HTTP request, handles error responses, and unmarshals the result.
func (c *Client) executeRequest(req *http.Request, result interface{}) error {
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("reading error response body: %w", err)
		}
		var apiErr APIError
		if err := json.Unmarshal(respBody, &apiErr); err != nil {
			return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
		}
		return &apiErr
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}

	return nil
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

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.executeRequest(req, result)
}

// doList performs a paginated GET request with the maximum page size.
// Note: currently fetches only the first page (up to 200 items).
func (c *Client) doList(ctx context.Context, path string, result interface{}) error {
	return c.do(ctx, http.MethodGet, path+"?limit="+defaultPageSize, nil, result)
}

// Generic CRUD helpers that eliminate per-resource boilerplate.

func doCreate[T any](c *Client, ctx context.Context, path string, body *T) (*T, error) {
	var result T
	if err := c.do(ctx, http.MethodPost, path, body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func doGet[T any](c *Client, ctx context.Context, path string) (*T, error) {
	var result T
	if err := c.do(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func doUpdate[T any](c *Client, ctx context.Context, path string, body *T) (*T, error) {
	var result T
	if err := c.do(ctx, http.MethodPut, path, body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func doListAll[T any](c *Client, ctx context.Context, path string) ([]T, error) {
	var resp PaginatedResponse[T]
	if err := c.doList(ctx, path, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// --- Site operations ---

type Site struct {
	ID                string `json:"id"`
	InternalReference string `json:"internalReference"`
	Name              string `json:"name"`
}

func (c *Client) ListSites(ctx context.Context) ([]Site, error) {
	return doListAll[Site](c, ctx, "sites")
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
	return doCreate(c, ctx, fmt.Sprintf("sites/%s/networks", siteID), network)
}

func (c *Client) GetNetwork(ctx context.Context, siteID, networkID string) (*Network, error) {
	return doGet[Network](c, ctx, fmt.Sprintf("sites/%s/networks/%s", siteID, networkID))
}

func (c *Client) UpdateNetwork(ctx context.Context, siteID, networkID string, network *Network) (*Network, error) {
	return doUpdate(c, ctx, fmt.Sprintf("sites/%s/networks/%s", siteID, networkID), network)
}

func (c *Client) DeleteNetwork(ctx context.Context, siteID, networkID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("sites/%s/networks/%s", siteID, networkID), nil, nil)
}

func (c *Client) ListNetworks(ctx context.Context, siteID string) ([]Network, error) {
	return doListAll[Network](c, ctx, fmt.Sprintf("sites/%s/networks", siteID))
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
	return doCreate(c, ctx, fmt.Sprintf("sites/%s/wifi/broadcasts", siteID), broadcast)
}

func (c *Client) GetWifiBroadcast(ctx context.Context, siteID, broadcastID string) (*WifiBroadcast, error) {
	return doGet[WifiBroadcast](c, ctx, fmt.Sprintf("sites/%s/wifi/broadcasts/%s", siteID, broadcastID))
}

func (c *Client) UpdateWifiBroadcast(ctx context.Context, siteID, broadcastID string, broadcast *WifiBroadcast) (*WifiBroadcast, error) {
	return doUpdate(c, ctx, fmt.Sprintf("sites/%s/wifi/broadcasts/%s", siteID, broadcastID), broadcast)
}

func (c *Client) DeleteWifiBroadcast(ctx context.Context, siteID, broadcastID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("sites/%s/wifi/broadcasts/%s", siteID, broadcastID), nil, nil)
}

// --- Firewall Zone operations ---

type FirewallZone struct {
	ID         string          `json:"id,omitempty"`
	Name       string          `json:"name"`
	NetworkIDs []string        `json:"networkIds"`
	Metadata   *EntityMetadata `json:"metadata,omitempty"`
}

func (c *Client) CreateFirewallZone(ctx context.Context, siteID string, zone *FirewallZone) (*FirewallZone, error) {
	return doCreate(c, ctx, fmt.Sprintf("sites/%s/firewall/zones", siteID), zone)
}

func (c *Client) GetFirewallZone(ctx context.Context, siteID, zoneID string) (*FirewallZone, error) {
	return doGet[FirewallZone](c, ctx, fmt.Sprintf("sites/%s/firewall/zones/%s", siteID, zoneID))
}

func (c *Client) UpdateFirewallZone(ctx context.Context, siteID, zoneID string, zone *FirewallZone) (*FirewallZone, error) {
	return doUpdate(c, ctx, fmt.Sprintf("sites/%s/firewall/zones/%s", siteID, zoneID), zone)
}

func (c *Client) DeleteFirewallZone(ctx context.Context, siteID, zoneID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("sites/%s/firewall/zones/%s", siteID, zoneID), nil, nil)
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
	return doGet[Device](c, ctx, fmt.Sprintf("sites/%s/devices/%s", siteID, deviceID))
}

func (c *Client) ListDevices(ctx context.Context, siteID string) ([]Device, error) {
	return doListAll[Device](c, ctx, fmt.Sprintf("sites/%s/devices", siteID))
}
