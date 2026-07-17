---
page_title: "vayucloud_network_zone Data Source"
subcategory: ""
description: |-
  Reads a single network zone by resource identifier.
---

# `vayucloud_network_zone`

Returns the current configuration for one zone. To list every zone in an environment, use [`vayucloud_network_zone_list`](network_zone_list.md).

## Example Usage

```hcl
data "vayucloud_network_zone" "example" {
  network_zone_id = {{network_zone_id}}
}
```

## Argument Reference

### Required

* `network_zone_id` — (String) Zone resource identifier.

## Attributes Reference

* `id` — (String) Computed placeholder for Terraform.
* `name` — (String) Zone name.
* `environment_id` — (Number) Environment identifier.
* `firewall_id` — (Number) Firewall identifier.
* `no_of_ips` — (Number) IPv4 address count.
* `purpose` — (String) Zone purpose.
* `data_plane` — (String) Data plane type (for example `Auto IPAM`).
* `zone_type` — (String) Zone type (for example `overlay`, `vlan`).
* `no_of_v6_ips` — (Number) IPv6 address count when applicable.
* `ipv6_cidr` — (String) IPv6 CIDR when applicable.
* `status` — (String) API status.
* `message` — (String) API message.
* `response_code` — (Number) API response code (`0` indicates success).
* `raw_response` — (String) Raw JSON response for troubleshooting.
