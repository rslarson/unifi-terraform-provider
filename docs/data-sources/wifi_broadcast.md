---
page_title: "unifi_wifi_broadcast Data Source - UniFi"
subcategory: ""
description: |-
  Fetches details of a UniFi WiFi broadcast.
---

# unifi_wifi_broadcast (Data Source)

Fetches details of an existing UniFi WiFi broadcast (SSID) by ID.

## Example Usage

```terraform
data "unifi_wifi_broadcast" "main" {
  site_id = local.site_id
  id      = "existing-broadcast-id"
}

output "ssid_name" {
  value = data.unifi_wifi_broadcast.main.name
}
```

## Schema

### Required

- `id` (String) - The ID of the WiFi broadcast.
- `site_id` (String) - The site ID.

### Read-Only

- `type` (String) - Broadcast type.
- `name` (String) - SSID name.
- `enabled` (Boolean) - Whether the broadcast is enabled.
- `security_type` (String) - Security configuration type.
- `network_type` (String) - Network assignment type.
- `network_id` (String) - Associated network ID.
- `client_isolation_enabled` (Boolean) - Whether client isolation is enabled.
- `hide_name` (Boolean) - Whether the SSID is hidden.
- `multicast_to_unicast_conversion_enabled` (Boolean) - Whether multicast to unicast conversion is enabled.
- `uapsd_enabled` (Boolean) - Whether U-APSD is enabled.
