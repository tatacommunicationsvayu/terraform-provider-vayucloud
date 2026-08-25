---
page_title: "vayucloud_network_lb_list Data Source"
subcategory: ""
description: |-
  Lists Load Balancers for an engagement and endpoint.
---

# `vayucloud_network_lb_list`

Returns all load balancers for a given `engagement_id` and `endpoint_id`. Use [`vayucloud_network_lb`](network_lb.md) when you already have a single load balancer ID.

Resolve `engagement_id` and `endpoint_id` from [`vayucloud_account_engagement`](account_engagement.md) and [`vayucloud_account_location`](account_location.md), or read them from an existing firewall with [`vayucloud_network_firewall`](network_firewall.md).

## Example Usage

```hcl
data "vayucloud_network_firewall" "fw" {
  network_firewall_id = {{firewall_id}}
}

data "vayucloud_network_lb_list" "all" {
  engagement_id = data.vayucloud_network_firewall.fw.engagement_id
  endpoint_id   = data.vayucloud_network_firewall.fw.endpoint_id
}

output "lb_ids" {
  value = [for lb in data.vayucloud_network_lb_list.all.load_balancers : lb.id]
}
```

### Filter by firewall

```hcl
locals {
  lbs_on_firewall = [
    for lb in data.vayucloud_network_lb_list.all.load_balancers :
    lb if lb.firewall_id == {{firewall_id}}
  ]
}
```

## Argument Reference

### Required

* `engagement_id` — (Number) Engagement identifier.
* `endpoint_id` — (Number) Endpoint identifier.

## Attributes Reference

* `load_balancers` — (List of Object) One element per load balancer. Fields include:
  * `id` — (String) Load balancer CI Master ID.
  * `name` — (String) CI name.
  * `display_name` — (String) Display name.
  * `lb_type` — (String) Type (`F5`, `HAProxy`, `AVI`).
  * `bandwidth` — (String) Configured bandwidth in Mbps.
  * `firewall_id` — (String) Parent firewall CI Master ID.
  * `ci_status` — (String) CI status.
  * `engagement_id` — (Number) Engagement identifier.
  * `endpoint_id` — (Number) Endpoint identifier.
