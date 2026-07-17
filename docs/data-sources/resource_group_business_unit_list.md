---
page_title: "vayucloud_resource_group_business_unit_list Data Source"
subcategory: ""
description: |-
  Lists business units for a network firewall. Optional filters narrow results client-side.
---

# `vayucloud_resource_group_business_unit_list`

Returns every business unit attached to `firewall_id`. Use [`vayucloud_resource_group_business_unit`](resource_group_business_unit.md) to read one business unit by ID.

## Example Usage

```hcl
data "vayucloud_resource_group_business_unit_list" "all" {
  firewall_id = vayucloud_network_firewall.main.id
}

output "business_unit_names" {
  value = [for bu in data.vayucloud_resource_group_business_unit_list.all.business_units : bu.business_unit]
}
```

### Filter by name

```hcl
data "vayucloud_resource_group_business_unit_list" "one" {
  firewall_id = {{network_firewall_id}}

  filter {
    name   = "business_unit"
    values = [{{business_unit_name}}]
  }
}
```

## Argument Reference

### Required

* `firewall_id` — (Number) Firewall resource identifier.

### Optional

* `filter` — (Block) Repeatable. Each block requires `name` and `values`.

## Attributes Reference

* `business_units` — (List of Object) Objects include `resource_group_business_unit_id`, `business_unit`, `users`, and `status` as returned by the API.
* `status` — (String) API status.
* `message` — (String) API message.
* `response_code` — (Number) API response code.
