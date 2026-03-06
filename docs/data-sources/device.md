---
page_title: "unifi_device Data Source - UniFi"
subcategory: ""
description: |-
  Fetches details of an adopted UniFi device.
---

# unifi_device (Data Source)

Fetches details of an adopted UniFi device by ID. Use this to read device properties like firmware version, state, and uplink information.

## Example Usage

```terraform
data "unifi_device" "gateway" {
  site_id = local.site_id
  id      = "device-uuid"
}

output "gateway_firmware" {
  value = data.unifi_device.gateway.firmware_version
}

output "gateway_state" {
  value = data.unifi_device.gateway.state
}
```

## Schema

### Required

- `id` (String) - The device ID.
- `site_id` (String) - The site ID.

### Read-Only

- `mac_address` (String) - MAC address of the device.
- `ip_address` (String) - IP address of the device.
- `name` (String) - Name of the device.
- `model` (String) - Device model identifier.
- `supported` (Boolean) - Whether the device is supported.
- `state` (String) - Device state. Possible values: `ONLINE`, `OFFLINE`, `PENDING_ADOPTION`, `UPDATING`, `GETTING_READY`, `ADOPTING`, `DELETING`, `CONNECTION_INTERRUPTED`, `ISOLATED`.
- `firmware_version` (String) - Current firmware version.
- `firmware_updatable` (Boolean) - Whether the firmware can be updated.
- `adopted_at` (String) - Timestamp when the device was adopted.
- `provisioned_at` (String) - Timestamp when the device was last provisioned.
- `configuration_id` (String) - Configuration identifier.
- `uplink_device_id` (String) - ID of the parent/uplink device.
