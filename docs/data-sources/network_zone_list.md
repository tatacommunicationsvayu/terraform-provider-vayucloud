---
page_title: "vayucloud_network_zone_list Data Source"
subcategory: ""
description: |-
  Lists network zones for an environment. Optional filter blocks narrow results client-side.
---

# `vayucloud_network_zone_list`

Returns all zones in a given environment. Use [`vayucloud_network_zone`](network_zone.md) to read one zone by ID.

## Example Usage

```hcl
data "vayucloud_network_zone_list" "in_env" {
  environment_id = vayucloud_resource_group_environment.production.id
}

output "first_zone_id" {
  value = data.vayucloud_network_zone_list.in_env.network_zones[0].network_zone_id
}
```

### Filter by zone type

```hcl
data "vayucloud_network_zone_list" "overlay_only" {
  environment_id = {{environment_id}}

  filter {
    name   = "zone_type"
    values = [{{zone_type}}]
  }
}
```

## Argument Reference

### Required

* `environment_id` — (Number) Environment resource identifier.

### Optional

* `filter` — (Block) Repeatable. Each block requires `name` and `values`.

## Attributes Reference

* `id` — (String) Computed placeholder for Terraform.
* `network_zones` — (List of Object) Objects typically include `network_zone_id`, `name`, `environment_id`, `firewall_id`, `no_of_ips`, `purpose`, `zone_type`, `no_of_v6_ips`, and `ipv6_cidr`.
* `status` — (String) API status.
* `message` — (String) API message.
* `response_code` — (Number) API response code.

Use `network_zone_id` values with [`vayucloud_virtualmachine_image`](virtualmachine_image.md), [`vayucloud_virtualmachine_flavor`](virtualmachine_flavor.md), and [`vayucloud_virtualmachine`](../resources/virtualmachine.md).
