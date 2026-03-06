---
page_title: "unifi_network Resource - UniFi"
subcategory: ""
description: |-
  Manages a UniFi network.
---

# unifi_network (Resource)

Manages a UniFi network. Supports VLAN configuration and DHCP guarding with trusted server IP addresses.

## Example Usage

```terraform
# Basic VLAN network
resource "unifi_network" "iot" {
  site_id    = local.site_id
  name       = "IoT Network"
  management = "UNMANAGED"
  enabled    = true
  vlan_id    = 30
}

# Network with DHCP guarding
resource "unifi_network" "corporate" {
  site_id    = local.site_id
  name       = "Corporate"
  management = "GATEWAY"
  enabled    = true
  vlan_id    = 10

  trusted_dhcp_server_ip_addresses = [
    "10.0.10.1",
    "10.0.10.2",
  ]
}
```

## Schema

### Required

- `site_id` (String) - The site ID where the network is managed. Changing this forces a new resource.
- `name` (String) - Name of the network.
- `management` (String) - Management type. Valid values: `UNMANAGED`, `GATEWAY`, `SWITCH`.
- `enabled` (Boolean) - Whether the network is enabled.
- `vlan_id` (Number) - VLAN ID. Must be between 2 and 4000.

### Optional

- `trusted_dhcp_server_ip_addresses` (List of String) - List of trusted DHCP server IP addresses for DHCP guarding.

### Read-Only

- `id` (String) - The ID of the network.

## Import

Import using a composite ID of `site_id/network_id`:

```shell
terraform import unifi_network.example site-abc123/net-def456
```
