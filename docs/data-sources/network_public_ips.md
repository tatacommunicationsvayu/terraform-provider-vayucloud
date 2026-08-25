---
page_title: "vayucloud_network_public_ips Data Source"
subcategory: ""
description: |-
  Lists public IP inventory from the VayuCloud network public-ips API for a firewall or engagement.
---

# `vayucloud_network_public_ips`

Returns public IP rows from `GET .../network/public-ips` using either a **firewall** filter (`firewall-ci`) or an **engagement** filter (`engagement`). Exactly one of `firewall_id` or `engagement_id` must be set.

## Example Usage

```hcl
data "vayucloud_network_public_ips" "by_firewall" {
  firewall_id = {{firewall_id}}
}

data "vayucloud_network_public_ips" "by_engagement" {
  engagement_id = {{engagement_id}}
}

data "vayucloud_network_public_ips" "free_ips" {
  firewall_id = {{firewall_id}}

  filter {
    name   = "purpose"
    values = ["Free IP, Not associated to any"]
  }

  filter {
    name   = "is_used"
    values = ["false"]
  }
}
```

## Argument Reference

### Required (exactly one)

You must specify **one** of:

* `firewall_id` — (Number) Query parameter `firewall-ci`. Mutually exclusive with `engagement_id`.
* `engagement_id` — (Number) Query parameter `engagement`. Mutually exclusive with `firewall_id`.

### Optional

* `filter` — (Block List) Client-side filters applied after the API response. Multiple `filter` blocks are combined with **AND** logic; multiple `values` within a block use **OR** logic (case-insensitive exact match). Valid `name` values on each `public_ips` entry: `public_ip_segment`, `is_used`, `purpose`, `location`, `description`. For `is_used`, use `"true"` or `"false"`.

## Attributes Reference

* `id` — (String) Computed key for this read: `firewall-{id}` or `engagement-{id}` depending on which argument was set.
* `public_ips` — (List of object) Entries from the API `content` array. Each object includes:
  * `public_ip_segment` — (String)
  * `is_used` — (Bool)
  * `location` — (String)
  * `purpose` — (String)
  * `description` — (String)
