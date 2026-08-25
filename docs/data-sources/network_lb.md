---
page_title: "vayucloud_network_lb Data Source"
subcategory: ""
description: |-
  Reads a single Load Balancer by CI Master ID.
---

# `vayucloud_network_lb`

Returns current load balancer configuration for one appliance. To enumerate load balancers without a known ID, use [`vayucloud_network_lb_list`](network_lb_list.md).

**Note:** `zone_id` is only available on the [`vayucloud_network_lb`](../resources/network_lb.md) **resource** (Terraform-side preflight). The platform read API used by this data source does not return it. See the [Load balancer guide](../guides/load_balancer.md).

## Example Usage

```hcl
data "vayucloud_network_lb" "example" {
  id = {{load_balancer_id}}
}
```

## Argument Reference

### Required

* `id` — (String) Load Balancer CI Master ID.

## Attributes Reference

* `engagement_id` — (Number) Engagement identifier.
* `endpoint_id` — (Number) Endpoint identifier.
* `firewall_id` — (String) Parent firewall CI Master ID.
* `name` — (String) Load balancer CI name.
* `display_name` — (String) Display name.
* `type` — (String) Load balancer type (for example `HAProxy`).
* `bandwidth` — (String) Configured bandwidth in Mbps (for example `100Mbps`).
* `ci_status` — (String) CI status.
