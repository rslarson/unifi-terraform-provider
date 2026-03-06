---
page_title: "unifi_firewall_zone Resource - UniFi"
subcategory: ""
description: |-
  Manages a UniFi custom firewall zone.
---

# unifi_firewall_zone (Resource)

Manages a UniFi custom firewall zone. Firewall zones group networks together for use in firewall rules.

## Example Usage

```terraform
resource "unifi_firewall_zone" "iot_zone" {
  site_id     = local.site_id
  name        = "IoT Zone"
  network_ids = [unifi_network.iot.id]
}

resource "unifi_firewall_zone" "trusted" {
  site_id = local.site_id
  name    = "Trusted Networks"
  network_ids = [
    unifi_network.corporate.id,
    unifi_network.management.id,
  ]
}
```

## Schema

### Required

- `site_id` (String) - The site ID where the firewall zone is managed. Changing this forces a new resource.
- `name` (String) - Name of the firewall zone.
- `network_ids` (List of String) - List of network IDs associated with this firewall zone.

### Read-Only

- `id` (String) - The ID of the firewall zone.

## Import

Import using a composite ID of `site_id/zone_id`:

```shell
terraform import unifi_firewall_zone.example site-abc123/zone-def456
```
