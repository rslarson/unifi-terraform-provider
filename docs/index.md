---
page_title: "UniFi Provider"
subcategory: ""
description: |-
  Terraform provider for managing UniFi network infrastructure via the UI API.
---

# UniFi Provider

The UniFi provider manages network infrastructure on UniFi consoles (Dream Machine, Cloud Gateway, etc.) through the [UniFi Network API](https://developer.ui.com/). It connects via the cloud connector proxy, so no local network access to the console is required.

## Supported Resources

- **Networks** - VLAN networks with DHCP guarding
- **WiFi Broadcasts** - SSIDs with security, client isolation, and network assignment
- **Firewall Zones** - Custom zones with network associations

## Prerequisites

1. A UniFi console running UniFi Network 10.x or later
2. A [UI Account API key](https://account.ui.com/) with access to the console
3. The console's Host ID (found in the UniFi Site Manager at [unifi.ui.com](https://unifi.ui.com/))

## Authentication

The provider authenticates using a UI Account API key, passed via the cloud connector at `api.ui.com`. Generate an API key at [account.ui.com](https://account.ui.com/) under **Security > API Keys**.

You can provide credentials directly in the provider block or via environment variables:

```shell
export UNIFI_API_KEY="your-api-key"
export UNIFI_HOST_ID="your-console-host-id"
```

## Example Usage

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

{{ tffile "examples/main.tf" }}

## Schema

### Optional

- `api_key` (String, Sensitive) - API key for authenticating with the UniFi API. Can also be set via the `UNIFI_API_KEY` environment variable.
- `host_id` (String) - Host ID of the UniFi console for cloud connector proxying. Can also be set via the `UNIFI_HOST_ID` environment variable.

## External Resources

- [UniFi Network API Documentation](https://developer.ui.com/)
- [UniFi Site Manager](https://unifi.ui.com/)
- [Terraform Registry](https://registry.terraform.io/providers/rslarson/unifi/latest)
