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
# WPA2/WPA3 Personal WiFi using write-only passphrase (recommended)
resource "unifi_wifi_broadcast" "home" {
  site_id                = local.site_id
  type                   = "STANDARD"
  name                   = "Home WiFi"
  enabled                = true
  security_type          = "WPA2_WPA3_PERSONAL"
  passphrase_wo          = var.wifi_password
  passphrase_wo_version  = 1  # increment when password changes
  network_type           = "SPECIFIC"
  network_id             = unifi_network.main.id
}

# IoT-optimized WiFi with client isolation
resource "unifi_wifi_broadcast" "iot" {
  site_id                  = local.site_id
  type                     = "IOT_OPTIMIZED"
  name                     = "IoT Devices"
  enabled                  = true
  security_type            = "WPA2_PERSONAL"
  passphrase_wo            = var.iot_password
  passphrase_wo_version    = 1
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

## Write-Only Passphrase

This resource supports a write-only passphrase via `passphrase_wo` and `passphrase_wo_version`. When using these attributes, the passphrase value is **never stored** in the Terraform state or plan files — only the version number is persisted. This is the recommended approach for managing WiFi passwords.

To rotate a passphrase, update the `passphrase_wo` value and increment `passphrase_wo_version`. Terraform detects the version change and re-applies the passphrase.

~> **Note:** `passphrase_wo` requires Terraform 1.11 or later. For older versions, use the `passphrase` attribute instead.

## Schema

### Required

- `site_id` (String) - The site ID where the WiFi broadcast is managed. Changing this forces a new resource.
- `type` (String) - Broadcast type. Valid values: `STANDARD`, `IOT_OPTIMIZED`.
- `name` (String) - Name of the WiFi broadcast (SSID name).
- `enabled` (Boolean) - Whether the WiFi broadcast is enabled.
- `security_type` (String) - Security type. Valid values: `OPEN`, `WPA2_PERSONAL`, `WPA3_PERSONAL`, `WPA2_WPA3_PERSONAL`, `WPA2_ENTERPRISE`, `WPA3_ENTERPRISE`, `WPA2_WPA3_ENTERPRISE`.
- `network_type` (String) - Network assignment type. Valid values: `NATIVE`, `SPECIFIC`.

### Optional

- `passphrase` (String, Sensitive) - WiFi passphrase. Required for personal security types. Must be 8–63 printable ASCII characters per IEEE 802.11i. **This value is stored in the Terraform state.** Conflicts with `passphrase_wo`.
- `passphrase_wo` (String, Write-Only) - Write-only WiFi passphrase. Required for personal security types. Must be 8–63 printable ASCII characters per IEEE 802.11i. **This value is never stored in the Terraform state or plan files.** Use with `passphrase_wo_version`. Conflicts with `passphrase`. Requires Terraform >= 1.11.
- `passphrase_wo_version` (Number) - An integer that tracks changes to `passphrase_wo`. Increment this value to signal that the passphrase has changed and should be re-applied.
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
