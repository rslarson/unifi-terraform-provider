---
page_title: "unifi_firewall_zone Data Source - UniFi"
subcategory: ""
description: |-
  Fetches details of a UniFi firewall zone.
---

# unifi_firewall_zone (Data Source)

Fetches details of an existing UniFi firewall zone by ID.

## Example Usage

```terraform
data "unifi_firewall_zone" "default" {
  site_id = local.site_id
  id      = "existing-zone-id"
}

output "zone_networks" {
  value = data.unifi_firewall_zone.default.network_ids
}
```

## Schema

### Required

- `id` (String) - The ID of the firewall zone.
- `site_id` (String) - The site ID.

### Read-Only

- `name` (String) - Name of the firewall zone.
- `network_ids` (List of String) - Network IDs associated with this zone.
