terraform {
  required_providers {
    unifi = {
      source = "rslarson/unifi"
    }
  }
}

# Configure the provider with your API key and console host ID.
# These can also be set via UNIFI_API_KEY and UNIFI_HOST_ID environment variables.
provider "unifi" {
  api_key = var.unifi_api_key
  host_id = var.unifi_host_id
}

variable "unifi_api_key" {
  type      = string
  sensitive = true
}

variable "unifi_host_id" {
  type = string
}

# Look up all sites on the console
data "unifi_sites" "all" {}

# Use the first site
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

# Create a WiFi broadcast for the IoT network.
# Uses passphrase_wo so the password is never stored in the Terraform state.
# Increment passphrase_wo_version when rotating the password.
resource "unifi_wifi_broadcast" "iot_wifi" {
  site_id               = local.site_id
  type                  = "IOT_OPTIMIZED"
  name                  = "IoT WiFi"
  enabled               = true
  security_type         = "WPA2_WPA3_PERSONAL"
  passphrase_wo         = var.iot_wifi_password
  passphrase_wo_version = 1
  network_type          = "SPECIFIC"
  network_id            = unifi_network.iot.id
}

variable "iot_wifi_password" {
  type      = string
  sensitive = true
}

# Create a firewall zone for the IoT network
resource "unifi_firewall_zone" "iot_zone" {
  site_id     = local.site_id
  name        = "IoT Zone"
  network_ids = [unifi_network.iot.id]
}

# Look up device details
# data "unifi_device" "gateway" {
#   site_id = local.site_id
#   id      = "your-device-uuid"
# }

output "iot_network_id" {
  value = unifi_network.iot.id
}

output "iot_wifi_broadcast_id" {
  value = unifi_wifi_broadcast.iot_wifi.id
}

output "sites" {
  value = data.unifi_sites.all.sites
}
