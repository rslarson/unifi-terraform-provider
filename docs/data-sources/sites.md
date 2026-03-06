---
page_title: "unifi_sites Data Source - UniFi"
subcategory: ""
description: |-
  Fetches all sites managed by the UniFi console.
---

# unifi_sites (Data Source)

Fetches all sites managed by the UniFi console. Use this to discover site IDs for other resources and data sources.

## Example Usage

```terraform
data "unifi_sites" "all" {}

# Use the first site's ID
locals {
  site_id = data.unifi_sites.all.sites[0].id
}

output "sites" {
  value = data.unifi_sites.all.sites
}
```

## Schema

### Read-Only

- `sites` (List of Object) - List of sites. Each site has:
  - `id` (String) - Site ID.
  - `internal_reference` (String) - Internal reference identifier.
  - `name` (String) - Site name.
