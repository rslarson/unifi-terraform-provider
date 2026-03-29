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

// ACL rule types.
const (
	AclTypeIPv4 = "IPV4"
	AclTypeMAC  = "MAC"
)

// ACL/Firewall action types.
const (
	ActionAllow  = "ALLOW"
	ActionBlock  = "BLOCK"
	ActionReject = "REJECT"
)

// Firewall IP protocol scope types.
const (
	IPVersionIPv4       = "IPV4"
	IPVersionIPv6       = "IPV6"
	IPVersionIPv4AndV6  = "IPV4_AND_IPV6"
)

// DHCP configuration modes.
const (
	DhcpModeServer = "SERVER"
	DhcpModeRelay  = "RELAY"
)

// Traffic matching list types.
const (
	TrafficMatchingPorts          = "PORTS"
	TrafficMatchingIPv4Addresses  = "IPV4_ADDRESSES"
	TrafficMatchingIPv6Addresses  = "IPV6_ADDRESSES"
)

// DNS policy types.
const (
	DnsPolicyARecord       = "A_RECORD"
	DnsPolicyAAAARecord    = "AAAA_RECORD"
	DnsPolicyCNAMERecord   = "CNAME_RECORD"
	DnsPolicyMXRecord      = "MX_RECORD"
	DnsPolicyTXTRecord     = "TXT_RECORD"
	DnsPolicySRVRecord     = "SRV_RECORD"
	DnsPolicyForwardDomain = "FORWARD_DOMAIN"
)

// ACL filter types.
const (
	AclFilterIPAddressesOrSubnets = "IP_ADDRESSES_OR_SUBNETS"
	AclFilterNetworks             = "NETWORKS"
	AclFilterPorts                = "PORTS"
	AclFilterMacAddresses         = "MAC_ADDRESSES"
)

// Firewall traffic filter types.
const (
	TrafficFilterNetwork    = "NETWORK"
	TrafficFilterIPAddress  = "IP_ADDRESS"
	TrafficFilterMacAddress = "MAC_ADDRESS"
	TrafficFilterPort       = "PORT"
)

// IPsec filter values.
const (
	IpsecMatchEncrypted    = "MATCH_ENCRYPTED"
	IpsecMatchNotEncrypted = "MATCH_NOT_ENCRYPTED"
)

// Broadcasting device filter types.
const (
	DeviceFilterDevices    = "DEVICES"
	DeviceFilterDeviceTags = "DEVICE_TAGS"
)

// Traffic matching item types.
const (
	TrafficItemIPAddress       = "IP_ADDRESS"
	TrafficItemSubnet          = "SUBNET"
	TrafficItemIPAddressRange  = "IP_ADDRESS_RANGE"
	TrafficItemPortNumber      = "PORT_NUMBER"
	TrafficItemPortNumberRange = "PORT_NUMBER_RANGE"
)

// Blackout schedule day types.
const (
	BlackoutAllDay    = "ALL_DAY"
	BlackoutTimeRange = "TIME_RANGE"
)

// Multicast/client filtering actions.
const (
	FilterActionAllow = "ALLOW"
	FilterActionBlock = "BLOCK"
)

// mDNS proxy modes.
const (
	MdnsProxyAuto   = "AUTO"
	MdnsProxyCustom = "CUSTOM"
)

// ============================================================================
// Client infrastructure
// ============================================================================

// Client manages communication with the UniFi Network API.
type Client struct {
	baseURL    string
	apiKey     string
	hostID     string
	httpClient *http.Client
}

// NewClient creates a new UniFi API client.
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

// NewClientForTesting creates a client with a custom base URL, intended for
// unit tests that need to point the client at an httptest.Server.
func NewClientForTesting(apiKey, hostID, baseURL string) *Client {
	c := NewClient(apiKey, hostID)
	c.baseURL = baseURL
	return c
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

// EntityMetadata holds API metadata for managed entities.
type EntityMetadata struct {
	Origin string `json:"origin"`
}

func (c *Client) buildURL(path string) string {
	return fmt.Sprintf("%s/%s/connector/consoles/%s/%s/%s",
		c.baseURL, apiVersion, c.hostID, apiVersion, path)
}

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

func (c *Client) doList(ctx context.Context, path string, result interface{}) error {
	return c.do(ctx, http.MethodGet, path+"?limit="+defaultPageSize, nil, result)
}

// Generic CRUD helpers.

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

// ============================================================================
// Site operations
// ============================================================================

type Site struct {
	ID                string `json:"id"`
	InternalReference string `json:"internalReference"`
	Name              string `json:"name"`
}

func (c *Client) ListSites(ctx context.Context) ([]Site, error) {
	return doListAll[Site](c, ctx, "sites")
}

// ============================================================================
// Network operations
// ============================================================================

type Network struct {
	ID           string          `json:"id,omitempty"`
	Name         string          `json:"name"`
	Management   string          `json:"management"`
	Enabled      bool            `json:"enabled"`
	VlanID       int             `json:"vlanId"`
	Metadata     *EntityMetadata `json:"metadata,omitempty"`
	DhcpGuarding *DhcpGuarding   `json:"dhcpGuarding,omitempty"`

	// Gateway and Switch managed fields.
	IsolationEnabled      *bool       `json:"isolationEnabled,omitempty"`
	CellularBackupEnabled *bool       `json:"cellularBackupEnabled,omitempty"`
	IPv4Configuration     *IPv4Config `json:"ipv4Configuration,omitempty"`

	// Gateway only fields.
	InternetAccessEnabled *bool       `json:"internetAccessEnabled,omitempty"`
	MdnsForwardingEnabled *bool       `json:"mdnsForwardingEnabled,omitempty"`
	ZoneID                string      `json:"zoneId,omitempty"`
	IPv6Configuration     *IPv6Config `json:"ipv6Configuration,omitempty"`

	// Switch only fields.
	DeviceID string `json:"deviceId,omitempty"`
}

type DhcpGuarding struct {
	TrustedDhcpServerIPAddresses []string `json:"trustedDhcpServerIpAddresses"`
}

type IPv4Config struct {
	AutoScaleEnabled        bool              `json:"autoScaleEnabled"`
	HostIPAddress           string            `json:"hostIpAddress"`
	PrefixLength            int               `json:"prefixLength"`
	AdditionalHostIPSubnets []string          `json:"additionalHostIpSubnets,omitempty"`
	DhcpConfiguration       *DhcpConfig       `json:"dhcpConfiguration,omitempty"`
	NatOutboundIPConfig     []NatOutboundConfig `json:"natOutboundIpAddressConfiguration,omitempty"`
}

type DhcpConfig struct {
	Mode               string     `json:"mode"`
	// Server mode fields.
	IPAddressRange     *DhcpRange `json:"ipAddressRange,omitempty"`
	GatewayIPOverride  string     `json:"gatewayIpAddressOverride,omitempty"`
	DnsServerOverride  []string   `json:"dnsServerIpAddressesOverride,omitempty"`
	LeaseTimeSeconds   *int       `json:"leaseTimeSeconds,omitempty"`
	DomainName         string     `json:"domainName,omitempty"`
	// Relay mode fields.
	ServerIPAddresses  []string   `json:"dhcpServerIpAddresses,omitempty"`
}

type DhcpRange struct {
	Start string `json:"start"`
	Stop  string `json:"stop"`
}

type NatOutboundConfig struct {
	Type           string `json:"type"`
	WanInterfaceID string `json:"wanInterfaceId"`
}

type IPv6Config struct {
	InterfaceType           string                   `json:"interfaceType"`
	ClientAddressAssignment *IPv6ClientAssignment     `json:"clientAddressAssignment,omitempty"`
	RouterAdvertisement     *IPv6RouterAdvertisement  `json:"routerAdvertisement,omitempty"`
	DnsServerOverride       []string                  `json:"dnsServerIpAddressesOverride,omitempty"`
	AdditionalHostIPSubnets []string                  `json:"additionalHostIpSubnets,omitempty"`
}

type IPv6ClientAssignment struct {
	DhcpConfiguration *IPv6DhcpConfig `json:"dhcpConfiguration,omitempty"`
	SlaacEnabled      bool            `json:"slaacEnabled"`
}

type IPv6DhcpConfig struct {
	IPAddressSuffixRange *IPv6SuffixRange `json:"ipAddressSuffixRange,omitempty"`
	LeaseTimeSeconds     *int             `json:"leaseTimeSeconds,omitempty"`
}

type IPv6SuffixRange struct {
	Start string `json:"start"`
	Stop  string `json:"stop"`
}

type IPv6RouterAdvertisement struct {
	Priority string `json:"priority"`
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

// ============================================================================
// WiFi Broadcast operations
// ============================================================================

type WifiBroadcast struct {
	ID                                  string                    `json:"id,omitempty"`
	Type                                string                    `json:"type"`
	Name                                string                    `json:"name"`
	Enabled                             bool                      `json:"enabled"`
	SecurityConfiguration               *SecurityConfiguration    `json:"securityConfiguration"`
	Network                             *BroadcastNetwork         `json:"network"`
	ClientIsolationEnabled              bool                      `json:"clientIsolationEnabled"`
	HideName                            bool                      `json:"hideName"`
	MulticastToUnicastConversionEnabled bool                      `json:"multicastToUnicastConversionEnabled"`
	UapsdEnabled                        bool                      `json:"uapsdEnabled"`
	Metadata                            *EntityMetadata           `json:"metadata,omitempty"`

	// Required fields added per API spec.
	BasicDataRateKbps     *BasicDataRateKbps     `json:"basicDataRateKbpsByFrequencyGHz,omitempty"`
	ClientFilteringPolicy *ClientFilteringPolicy  `json:"clientFilteringPolicy,omitempty"`
	BlackoutSchedule      *BlackoutSchedule       `json:"blackoutScheduleConfiguration,omitempty"`

	// Optional shared fields.
	BroadcastingDeviceFilter  *BroadcastingDeviceFilter  `json:"broadcastingDeviceFilter,omitempty"`
	MulticastFilteringPolicy  *MulticastFilteringPolicy   `json:"multicastFilteringPolicy,omitempty"`
	MdnsProxyConfiguration    *MdnsProxyConfiguration     `json:"mdnsProxyConfiguration,omitempty"`

	// STANDARD type only fields.
	BroadcastingFrequenciesGHz []float64 `json:"broadcastingFrequenciesGHz,omitempty"`
	BandSteeringEnabled        *bool     `json:"bandSteeringEnabled,omitempty"`
	MloEnabled                 *bool     `json:"mloEnabled,omitempty"`
	ArpProxyEnabled            *bool     `json:"arpProxyEnabled,omitempty"`
	BssTransitionEnabled       *bool     `json:"bssTransitionEnabled,omitempty"`
	AdvertiseDeviceName        *bool     `json:"advertiseDeviceName,omitempty"`
}

type SecurityConfiguration struct {
	Type       string `json:"type"`
	Passphrase string `json:"passphrase,omitempty"`
}

type BroadcastNetwork struct {
	Type      string `json:"type"`
	NetworkID string `json:"networkId,omitempty"`
}

type BasicDataRateKbps struct {
	GHz24 int `json:"2.4"`
	GHz5  int `json:"5"`
}

type ClientFilteringPolicy struct {
	Action           string   `json:"action"`
	MacAddressFilter []string `json:"macAddressFilter"`
}

type BlackoutSchedule struct {
	Days []BlackoutDay `json:"days"`
}

type BlackoutDay struct {
	Type       string      `json:"type"`
	Day        string      `json:"day"`
	TimeRanges []TimeRange `json:"timeRanges,omitempty"`
}

type TimeRange struct {
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
}

type BroadcastingDeviceFilter struct {
	Type         string   `json:"type"`
	DeviceIDs    []string `json:"deviceIds,omitempty"`
	DeviceTagIDs []string `json:"deviceTagIds,omitempty"`
}

type MulticastFilteringPolicy struct {
	Action string `json:"action"`
}

type MdnsProxyConfiguration struct {
	Mode string `json:"mode"`
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

// ============================================================================
// Firewall Zone operations
// ============================================================================

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

// ============================================================================
// Firewall Policy operations
// ============================================================================

type FirewallPolicy struct {
	ID                    string              `json:"id,omitempty"`
	Enabled               bool                `json:"enabled"`
	Name                  string              `json:"name"`
	Description           string              `json:"description,omitempty"`
	Action                *FirewallAction     `json:"action"`
	Source                *FirewallEndpoint   `json:"source"`
	Destination           *FirewallEndpoint   `json:"destination"`
	IPProtocolScope       *IPProtocolScope    `json:"ipProtocolScope"`
	ConnectionStateFilter []string            `json:"connectionStateFilter,omitempty"`
	IpsecFilter           string              `json:"ipsecFilter,omitempty"`
	LoggingEnabled        bool                `json:"loggingEnabled"`
	Schedule              *FirewallSchedule   `json:"schedule,omitempty"`
	Metadata              *EntityMetadata     `json:"metadata,omitempty"`
}

type FirewallAction struct {
	Type               string `json:"type"`
	AllowReturnTraffic *bool  `json:"allowReturnTraffic,omitempty"`
}

type FirewallEndpoint struct {
	ZoneID        string                `json:"zoneId"`
	TrafficFilter *FirewallTrafficFilter `json:"trafficFilter,omitempty"`
}

type FirewallTrafficFilter struct {
	Type         string   `json:"type"`
	NetworkID    string   `json:"networkId,omitempty"`
	IPAddresses  []string `json:"ipAddresses,omitempty"`
	MacAddresses []string `json:"macAddresses,omitempty"`
	PortFilter   *PortFilter `json:"portFilter,omitempty"`
}

type PortFilter struct {
	Type          string     `json:"type"`
	MatchOpposite bool       `json:"matchOpposite"`
	Items         []PortItem `json:"items,omitempty"`
}

type PortItem struct {
	Type       string `json:"type"`
	PortNumber *int   `json:"portNumber,omitempty"`
	StartPort  *int   `json:"startPort,omitempty"`
	EndPort    *int   `json:"endPort,omitempty"`
}

type IPProtocolScope struct {
	IPVersion string                `json:"ipVersion"`
	Protocol  *ProtocolFilter       `json:"protocol,omitempty"`
}

type ProtocolFilter struct {
	Type string `json:"type"`
}

type FirewallSchedule struct {
	Mode       string      `json:"mode"`
	TimeFilter *TimeFilter `json:"timeFilter,omitempty"`
	DaysOfWeek []string    `json:"daysOfWeek,omitempty"`
}

type TimeFilter struct {
	StartTime string `json:"startTime"`
	StopTime  string `json:"stopTime"`
}

type FirewallPolicyOrdering struct {
	OrderedFirewallPolicyIDs *PolicyOrderingIDs `json:"orderedFirewallPolicyIds"`
}

type PolicyOrderingIDs struct {
	BeforeSystemDefined []string `json:"beforeSystemDefined"`
	AfterSystemDefined  []string `json:"afterSystemDefined"`
}

func (c *Client) CreateFirewallPolicy(ctx context.Context, siteID string, policy *FirewallPolicy) (*FirewallPolicy, error) {
	return doCreate(c, ctx, fmt.Sprintf("sites/%s/firewall/policies", siteID), policy)
}

func (c *Client) GetFirewallPolicy(ctx context.Context, siteID, policyID string) (*FirewallPolicy, error) {
	return doGet[FirewallPolicy](c, ctx, fmt.Sprintf("sites/%s/firewall/policies/%s", siteID, policyID))
}

func (c *Client) UpdateFirewallPolicy(ctx context.Context, siteID, policyID string, policy *FirewallPolicy) (*FirewallPolicy, error) {
	return doUpdate(c, ctx, fmt.Sprintf("sites/%s/firewall/policies/%s", siteID, policyID), policy)
}

func (c *Client) DeleteFirewallPolicy(ctx context.Context, siteID, policyID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("sites/%s/firewall/policies/%s", siteID, policyID), nil, nil)
}

func (c *Client) ListFirewallPolicies(ctx context.Context, siteID string) ([]FirewallPolicy, error) {
	return doListAll[FirewallPolicy](c, ctx, fmt.Sprintf("sites/%s/firewall/policies", siteID))
}

func (c *Client) GetFirewallPolicyOrdering(ctx context.Context, siteID string) (*FirewallPolicyOrdering, error) {
	return doGet[FirewallPolicyOrdering](c, ctx, fmt.Sprintf("sites/%s/firewall/policies/ordering", siteID))
}

func (c *Client) UpdateFirewallPolicyOrdering(ctx context.Context, siteID string, ordering *FirewallPolicyOrdering) (*FirewallPolicyOrdering, error) {
	return doUpdate(c, ctx, fmt.Sprintf("sites/%s/firewall/policies/ordering", siteID), ordering)
}

// ============================================================================
// ACL Rule operations
// ============================================================================

type AclRule struct {
	ID                    string           `json:"id,omitempty"`
	Type                  string           `json:"type"`
	Enabled               bool             `json:"enabled"`
	Name                  string           `json:"name"`
	Description           string           `json:"description,omitempty"`
	Action                string           `json:"action"`
	SourceFilter          *AclFilter       `json:"sourceFilter,omitempty"`
	DestinationFilter     *AclFilter       `json:"destinationFilter,omitempty"`
	ProtocolFilter        []string         `json:"protocolFilter,omitempty"`
	EnforcingDeviceFilter *AclDeviceFilter `json:"enforcingDeviceFilter,omitempty"`
	NetworkIDFilter       string           `json:"networkIdFilter,omitempty"`
	Metadata              *EntityMetadata  `json:"metadata,omitempty"`
}

type AclFilter struct {
	Type                 string   `json:"type"`
	IPAddressesOrSubnets []string `json:"ipAddressesOrSubnets,omitempty"`
	NetworkIDs           []string `json:"networkIds,omitempty"`
	PortNumbers          []int    `json:"portNumbers,omitempty"`
	PortFilter           []int    `json:"portFilter,omitempty"`
	MacAddresses         []string `json:"macAddresses,omitempty"`
}

type AclDeviceFilter struct {
	Type      string   `json:"type"`
	DeviceIDs []string `json:"deviceIds"`
}

type AclRuleOrdering struct {
	OrderedAclRuleIDs []string `json:"orderedAclRuleIds"`
}

func (c *Client) CreateAclRule(ctx context.Context, siteID string, rule *AclRule) (*AclRule, error) {
	return doCreate(c, ctx, fmt.Sprintf("sites/%s/acl-rules", siteID), rule)
}

func (c *Client) GetAclRule(ctx context.Context, siteID, ruleID string) (*AclRule, error) {
	return doGet[AclRule](c, ctx, fmt.Sprintf("sites/%s/acl-rules/%s", siteID, ruleID))
}

func (c *Client) UpdateAclRule(ctx context.Context, siteID, ruleID string, rule *AclRule) (*AclRule, error) {
	return doUpdate(c, ctx, fmt.Sprintf("sites/%s/acl-rules/%s", siteID, ruleID), rule)
}

func (c *Client) DeleteAclRule(ctx context.Context, siteID, ruleID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("sites/%s/acl-rules/%s", siteID, ruleID), nil, nil)
}

func (c *Client) ListAclRules(ctx context.Context, siteID string) ([]AclRule, error) {
	return doListAll[AclRule](c, ctx, fmt.Sprintf("sites/%s/acl-rules", siteID))
}

func (c *Client) GetAclRuleOrdering(ctx context.Context, siteID string) (*AclRuleOrdering, error) {
	return doGet[AclRuleOrdering](c, ctx, fmt.Sprintf("sites/%s/acl-rules/ordering", siteID))
}

func (c *Client) UpdateAclRuleOrdering(ctx context.Context, siteID string, ordering *AclRuleOrdering) (*AclRuleOrdering, error) {
	return doUpdate(c, ctx, fmt.Sprintf("sites/%s/acl-rules/ordering", siteID), ordering)
}

// ============================================================================
// Traffic Matching List operations
// ============================================================================

type TrafficMatchingList struct {
	ID       string                `json:"id,omitempty"`
	Type     string                `json:"type"`
	Name     string                `json:"name"`
	Items    []TrafficMatchingItem `json:"items"`
	Metadata *EntityMetadata       `json:"metadata,omitempty"`
}

type TrafficMatchingItem struct {
	Type       string `json:"type"`
	Value      string `json:"value,omitempty"`
	Subnet     string `json:"subnet,omitempty"`
	StartIP    string `json:"start,omitempty"`
	EndIP      string `json:"end,omitempty"`
	PortNumber *int   `json:"portNumber,omitempty"`
	StartPort  *int   `json:"startPort,omitempty"`
	EndPort    *int   `json:"endPort,omitempty"`
}

func (c *Client) CreateTrafficMatchingList(ctx context.Context, siteID string, list *TrafficMatchingList) (*TrafficMatchingList, error) {
	return doCreate(c, ctx, fmt.Sprintf("sites/%s/traffic-matching-lists", siteID), list)
}

func (c *Client) GetTrafficMatchingList(ctx context.Context, siteID, listID string) (*TrafficMatchingList, error) {
	return doGet[TrafficMatchingList](c, ctx, fmt.Sprintf("sites/%s/traffic-matching-lists/%s", siteID, listID))
}

func (c *Client) UpdateTrafficMatchingList(ctx context.Context, siteID, listID string, list *TrafficMatchingList) (*TrafficMatchingList, error) {
	return doUpdate(c, ctx, fmt.Sprintf("sites/%s/traffic-matching-lists/%s", siteID, listID), list)
}

func (c *Client) DeleteTrafficMatchingList(ctx context.Context, siteID, listID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("sites/%s/traffic-matching-lists/%s", siteID, listID), nil, nil)
}

func (c *Client) ListTrafficMatchingLists(ctx context.Context, siteID string) ([]TrafficMatchingList, error) {
	return doListAll[TrafficMatchingList](c, ctx, fmt.Sprintf("sites/%s/traffic-matching-lists", siteID))
}

// ============================================================================
// DNS Policy operations
// ============================================================================

type DnsPolicy struct {
	ID          string          `json:"id,omitempty"`
	Type        string          `json:"type"`
	Enabled     bool            `json:"enabled"`
	Name        string          `json:"name,omitempty"`
	Domain      string          `json:"domain,omitempty"`
	IPv4Address string          `json:"ipv4Address,omitempty"`
	IPv6Address string          `json:"ipv6Address,omitempty"`
	Target      string          `json:"target,omitempty"`
	TTLSeconds  *int            `json:"ttlSeconds,omitempty"`
	Priority    *int            `json:"priority,omitempty"`
	Weight      *int            `json:"weight,omitempty"`
	Port        *int            `json:"port,omitempty"`
	TxtValue    string          `json:"txtValue,omitempty"`
	Metadata    *EntityMetadata `json:"metadata,omitempty"`
	// Forward domain fields.
	ForwardTo []string `json:"forwardTo,omitempty"`
}

func (c *Client) CreateDnsPolicy(ctx context.Context, siteID string, policy *DnsPolicy) (*DnsPolicy, error) {
	return doCreate(c, ctx, fmt.Sprintf("sites/%s/dns/policies", siteID), policy)
}

func (c *Client) GetDnsPolicy(ctx context.Context, siteID, policyID string) (*DnsPolicy, error) {
	return doGet[DnsPolicy](c, ctx, fmt.Sprintf("sites/%s/dns/policies/%s", siteID, policyID))
}

func (c *Client) UpdateDnsPolicy(ctx context.Context, siteID, policyID string, policy *DnsPolicy) (*DnsPolicy, error) {
	return doUpdate(c, ctx, fmt.Sprintf("sites/%s/dns/policies/%s", siteID, policyID), policy)
}

func (c *Client) DeleteDnsPolicy(ctx context.Context, siteID, policyID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("sites/%s/dns/policies/%s", siteID, policyID), nil, nil)
}

func (c *Client) ListDnsPolicies(ctx context.Context, siteID string) ([]DnsPolicy, error) {
	return doListAll[DnsPolicy](c, ctx, fmt.Sprintf("sites/%s/dns/policies", siteID))
}

// ============================================================================
// Hotspot Voucher operations (Create, Read, Delete — no Update)
// ============================================================================

type HotspotVoucher struct {
	ID                   string `json:"id,omitempty"`
	Name                 string `json:"name"`
	Code                 string `json:"code,omitempty"`
	TimeLimitMinutes     int    `json:"timeLimitMinutes"`
	AuthorizedGuestLimit *int   `json:"authorizedGuestLimit,omitempty"`
	DataUsageLimitMBytes *int   `json:"dataUsageLimitMBytes,omitempty"`
	RxRateLimitKbps      *int   `json:"rxRateLimitKbps,omitempty"`
	TxRateLimitKbps      *int   `json:"txRateLimitKbps,omitempty"`
	Expired              *bool  `json:"expired,omitempty"`
	CreatedAt            string `json:"createdAt,omitempty"`
	ActivatedAt          string `json:"activatedAt,omitempty"`
	ExpiresAt            string `json:"expiresAt,omitempty"`
}

type HotspotVoucherCreateRequest struct {
	Name                 string `json:"name"`
	Count                int    `json:"count,omitempty"`
	TimeLimitMinutes     int    `json:"timeLimitMinutes"`
	AuthorizedGuestLimit *int   `json:"authorizedGuestLimit,omitempty"`
	DataUsageLimitMBytes *int   `json:"dataUsageLimitMBytes,omitempty"`
	RxRateLimitKbps      *int   `json:"rxRateLimitKbps,omitempty"`
	TxRateLimitKbps      *int   `json:"txRateLimitKbps,omitempty"`
}

func (c *Client) CreateHotspotVoucher(ctx context.Context, siteID string, req *HotspotVoucherCreateRequest) ([]HotspotVoucher, error) {
	var result []HotspotVoucher
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("sites/%s/hotspot/vouchers", siteID), req, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetHotspotVoucher(ctx context.Context, siteID, voucherID string) (*HotspotVoucher, error) {
	return doGet[HotspotVoucher](c, ctx, fmt.Sprintf("sites/%s/hotspot/vouchers/%s", siteID, voucherID))
}

func (c *Client) DeleteHotspotVoucher(ctx context.Context, siteID, voucherID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("sites/%s/hotspot/vouchers/%s", siteID, voucherID), nil, nil)
}

func (c *Client) ListHotspotVouchers(ctx context.Context, siteID string) ([]HotspotVoucher, error) {
	return doListAll[HotspotVoucher](c, ctx, fmt.Sprintf("sites/%s/hotspot/vouchers", siteID))
}

// ============================================================================
// Device operations
// ============================================================================

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

// ============================================================================
// Read-only data source types and operations
// ============================================================================

// --- Clients ---

type NetworkClient struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Name      string `json:"name"`
	IPAddress string `json:"ipAddress"`
}

func (c *Client) ListClients(ctx context.Context, siteID string) ([]NetworkClient, error) {
	return doListAll[NetworkClient](c, ctx, fmt.Sprintf("sites/%s/clients", siteID))
}

// --- WANs ---

type Wan struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (c *Client) ListWans(ctx context.Context, siteID string) ([]Wan, error) {
	return doListAll[Wan](c, ctx, fmt.Sprintf("sites/%s/wans", siteID))
}

// --- VPN Servers ---

type VpnServer struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

func (c *Client) ListVpnServers(ctx context.Context, siteID string) ([]VpnServer, error) {
	return doListAll[VpnServer](c, ctx, fmt.Sprintf("sites/%s/vpn/servers", siteID))
}

// --- VPN Tunnels ---

type VpnTunnel struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
}

func (c *Client) ListVpnTunnels(ctx context.Context, siteID string) ([]VpnTunnel, error) {
	return doListAll[VpnTunnel](c, ctx, fmt.Sprintf("sites/%s/vpn/site-to-site-tunnels", siteID))
}

// --- RADIUS Profiles ---

type RadiusProfile struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (c *Client) ListRadiusProfiles(ctx context.Context, siteID string) ([]RadiusProfile, error) {
	return doListAll[RadiusProfile](c, ctx, fmt.Sprintf("sites/%s/radius/profiles", siteID))
}

// --- Device Tags ---

type DeviceTag struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	DeviceIDs []string `json:"deviceIds"`
}

func (c *Client) ListDeviceTags(ctx context.Context, siteID string) ([]DeviceTag, error) {
	return doListAll[DeviceTag](c, ctx, fmt.Sprintf("sites/%s/device-tags", siteID))
}

// --- Pending Devices ---

type PendingDevice struct {
	MacAddress        string   `json:"macAddress"`
	IPAddress         string   `json:"ipAddress"`
	Model             string   `json:"model"`
	State             string   `json:"state"`
	Supported         bool     `json:"supported"`
	FirmwareVersion   string   `json:"firmwareVersion"`
	FirmwareUpdatable bool     `json:"firmwareUpdatable"`
	Features          []string `json:"features"`
}

func (c *Client) ListPendingDevices(ctx context.Context) ([]PendingDevice, error) {
	return doListAll[PendingDevice](c, ctx, "pending-devices")
}

// --- DPI ---

type DpiCategory struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type DpiApplication struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func (c *Client) ListDpiCategories(ctx context.Context) ([]DpiCategory, error) {
	return doListAll[DpiCategory](c, ctx, "dpi/categories")
}

func (c *Client) ListDpiApplications(ctx context.Context) ([]DpiApplication, error) {
	return doListAll[DpiApplication](c, ctx, "dpi/applications")
}
