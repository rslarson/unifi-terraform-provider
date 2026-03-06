---
page_title: "unifi_network Data Source - UniFi"
subcategory: ""
description: |-
  Fetches details of a UniFi network.
---

# unifi_network (Data Source)

Fetches details of an existing UniFi network by ID.

## Example Usage

```terraform
data "unifi_network" "default" {
  site_id = local.site_id
  id      = "existing-network-id"
}

output "network_name" {
  value = data.unifi_network.default.name
}
```

## Schema

### Required

- `id` (String) - The ID of the network.
- `site_id` (String) - The site ID where the network exists.

### Read-Only

- `name` (String) - Name of the network.
- `management` (String) - Management type: `UNMANAGED`, `GATEWAY`, or `SWITCH`.
- `enabled` (Boolean) - Whether the network is enabled.
- `vlan_id` (Number) - VLAN ID.
- `trusted_dhcp_server_ip_addresses` (List of String) - List of trusted DHCP server IP addresses.
