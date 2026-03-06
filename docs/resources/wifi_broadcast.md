---
page_title: "unifi_wifi_broadcast Resource - UniFi"
subcategory: ""
description: |-
  Manages a UniFi WiFi broadcast (SSID).
---

# unifi_wifi_broadcast (Resource)

Manages a UniFi WiFi broadcast (SSID). Configure wireless networks with security settings, network assignment, and advanced options like client isolation and U-APSD.

## Example Usage

```terraform
# WPA2/WPA3 Personal WiFi on a specific network
resource "unifi_wifi_broadcast" "home" {
  site_id       = local.site_id
  type          = "STANDARD"
  name          = "Home WiFi"
  enabled       = true
  security_type = "WPA2_WPA3_PERSONAL"
  passphrase    = var.wifi_password
  network_type  = "SPECIFIC"
  network_id    = unifi_network.main.id
}

# IoT-optimized WiFi with client isolation
resource "unifi_wifi_broadcast" "iot" {
  site_id                  = local.site_id
  type                     = "IOT_OPTIMIZED"
  name                     = "IoT Devices"
  enabled                  = true
  security_type            = "WPA2_PERSONAL"
  passphrase               = var.iot_password
  network_type             = "SPECIFIC"
  network_id               = unifi_network.iot.id
  client_isolation_enabled = true
}

# Open guest WiFi on the native network
resource "unifi_wifi_broadcast" "guest" {
  site_id       = local.site_id
  type          = "STANDARD"
  name          = "Guest"
  enabled       = true
  security_type = "OPEN"
  network_type  = "NATIVE"
}
```

## Schema

### Required

- `site_id` (String) - The site ID where the WiFi broadcast is managed. Changing this forces a new resource.
- `type` (String) - Broadcast type. Valid values: `STANDARD`, `IOT_OPTIMIZED`.
- `name` (String) - Name of the WiFi broadcast (SSID name).
- `enabled` (Boolean) - Whether the WiFi broadcast is enabled.
- `security_type` (String) - Security type. Valid values: `OPEN`, `WPA2_PERSONAL`, `WPA3_PERSONAL`, `WPA2_WPA3_PERSONAL`, `WPA2_ENTERPRISE`, `WPA3_ENTERPRISE`, `WPA2_WPA3_ENTERPRISE`.
- `network_type` (String) - Network assignment type. Valid values: `NATIVE`, `SPECIFIC`.

### Optional

- `passphrase` (String, Sensitive) - WiFi passphrase. Required for personal security types.
- `network_id` (String) - Network ID when `network_type` is `SPECIFIC`.
- `client_isolation_enabled` (Boolean) - Whether client isolation is enabled. Defaults to `false`.
- `hide_name` (Boolean) - Whether to hide the SSID name. Defaults to `false`.
- `multicast_to_unicast_conversion_enabled` (Boolean) - Whether multicast to unicast conversion is enabled. Defaults to `false`.
- `uapsd_enabled` (Boolean) - Whether U-APSD (Unscheduled Automatic Power Save Delivery) is enabled. Defaults to `false`.

### Read-Only

- `id` (String) - The ID of the WiFi broadcast.

## Import

Import using a composite ID of `site_id/broadcast_id`:

```shell
terraform import unifi_wifi_broadcast.example site-abc123/wifi-def456
```
