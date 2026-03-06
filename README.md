# Terraform Provider for UniFi

A Terraform provider for managing UniFi network infrastructure through the [UniFi Network API](https://developer.ui.com/). Connects via the cloud connector proxy -- no local console access required.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.22 (for building from source)
- A UniFi console running UniFi Network 10.x+
- A [UI Account API key](https://account.ui.com/)

## Setup

### 1. Get Your API Key

Generate an API key at [account.ui.com](https://account.ui.com/) under **Security > API Keys**.

### 2. Find Your Console Host ID

Go to [unifi.ui.com](https://unifi.ui.com/) and find the Host ID for your console in the Site Manager.

### 3. Configure the Provider

```terraform
terraform {
  required_providers {
    unifi = {
      source = "rslarson/unifi"
    }
  }
}

provider "unifi" {
  api_key = var.unifi_api_key
  host_id = var.unifi_host_id
}
```

Or use environment variables:

```shell
export UNIFI_API_KEY="your-api-key"
export UNIFI_HOST_ID="your-console-host-id"
terraform plan
```

## Resources

| Resource | Description |
|---|---|
| `unifi_network` | VLAN networks with DHCP guarding |
| `unifi_wifi_broadcast` | WiFi SSIDs with security and network assignment |
| `unifi_firewall_zone` | Custom firewall zones with network associations |

## Data Sources

| Data Source | Description |
|---|---|
| `unifi_sites` | List all sites on the console |
| `unifi_network` | Look up a network by ID |
| `unifi_wifi_broadcast` | Look up a WiFi broadcast by ID |
| `unifi_firewall_zone` | Look up a firewall zone by ID |
| `unifi_device` | Look up an adopted device by ID |

## Quick Example

```terraform
# Discover sites
data "unifi_sites" "all" {}

locals {
  site_id = data.unifi_sites.all.sites[0].id
}

# Create a VLAN network
resource "unifi_network" "iot" {
  site_id    = local.site_id
  name       = "IoT Network"
  management = "UNMANAGED"
  enabled    = true
  vlan_id    = 30
}

# Create a WiFi SSID for the network
resource "unifi_wifi_broadcast" "iot_wifi" {
  site_id       = local.site_id
  type          = "IOT_OPTIMIZED"
  name          = "IoT WiFi"
  enabled       = true
  security_type = "WPA2_WPA3_PERSONAL"
  passphrase    = var.iot_wifi_password
  network_type  = "SPECIFIC"
  network_id    = unifi_network.iot.id
}

# Create a firewall zone
resource "unifi_firewall_zone" "iot_zone" {
  site_id     = local.site_id
  name        = "IoT Zone"
  network_ids = [unifi_network.iot.id]
}
```

## Importing Existing Resources

All resources support import using a composite ID format: `site_id/resource_id`.

```shell
terraform import unifi_network.example site-abc123/net-def456
terraform import unifi_wifi_broadcast.example site-abc123/wifi-def456
terraform import unifi_firewall_zone.example site-abc123/zone-def456
```

## Building from Source

```shell
git clone https://github.com/rslarson/terraform-provider-unifi.git
cd terraform-provider-unifi
make build
```

## Development

```shell
# Run tests
make test

# Run linter
make lint

# Build
make build
```

## Documentation

- [Provider Documentation](docs/index.md)
- [UniFi Network API](https://developer.ui.com/)
- [Terraform Registry](https://registry.terraform.io/providers/rslarson/unifi/latest)
