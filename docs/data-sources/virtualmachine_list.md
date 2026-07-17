---
page_title: "vayucloud_virtualmachine_list Data Source"
subcategory: ""
description: |-
  Lists virtual machines in a network zone. Optional filters narrow results client-side.
---

# `vayucloud_virtualmachine_list`

Returns a summary row per VM in `zone_id`. Use [`vayucloud_virtualmachine`](virtualmachine.md) when you need full detail for a known `instance_id`.

## Example Usage

```hcl
data "vayucloud_virtualmachine_list" "inventory" {
  zone_id = vayucloud_network_zone.compute.id
}

output "vm_count" {
  value = length(data.vayucloud_virtualmachine_list.inventory.virtual_machines)
}
```

### Filter by power and OS

```hcl
data "vayucloud_virtualmachine_list" "active_linux" {
  zone_id = {{zone_id}}

  filter {
    name   = "power_status"
    values = [{{power_status}}]
  }

  filter {
    name   = "os_type"
    values = [{{os_type}}]
  }
}
```

Use `power_status` and other filter values that match strings returned by your environment (case may vary).

## Argument Reference

### Required

* `zone_id` — (Number) Network zone identifier.

### Optional

* `filter` — (Block) Repeatable. Each block requires `name` and `values`. Filters apply to the returned list (logical AND across blocks per provider behavior).

## Attributes Reference

* `virtual_machines` — (List of Object) Summary objects including identifiers, names, zone, IP, power status, flavor and image references, and volume summaries as returned by the API.
* `status` — (String) API status.
* `message` — (String) API message.
* `response_code` — (Number) API response code.

This data source does not change VM power state; use [`vayucloud_virtualmachine_state`](../resources/virtualmachine_state.md) for power actions.
