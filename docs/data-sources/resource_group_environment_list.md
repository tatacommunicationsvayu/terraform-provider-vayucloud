---
page_title: "vayucloud_resource_group_environment_list Data Source"
subcategory: ""
description: |-
  Lists environments for a business unit. Optional filters narrow results client-side.
---

# `vayucloud_resource_group_environment_list`

Returns all environments under `business_unit_id`. Use [`vayucloud_resource_group_environment`](resource_group_environment.md) to read one environment by ID.

## Example Usage

```hcl
data "vayucloud_resource_group_environment_list" "all" {
  business_unit_id = vayucloud_resource_group_business_unit.team.id
}

output "environment_names" {
  value = [for e in data.vayucloud_resource_group_environment_list.all.environments : e.environment]
}
```

### Filter by status

```hcl
data "vayucloud_resource_group_environment_list" "active" {
  business_unit_id = {{business_unit_id}}

  filter {
    name   = "status"
    values = [{{status}}]
  }
}
```

Match filter values to strings returned by your tenant; API values may vary.

## Argument Reference

### Required

* `business_unit_id` — (Number) Business unit resource identifier.

### Optional

* `filter` — (Block) Repeatable. Each block requires `name` and `values`.

## Attributes Reference

* `environments` — (List of Object) Objects typically include `resource_group_environment_id`, `environment`, `business_unit_id`, and `status`.
* `status` — (String) API status.
* `message` — (String) API message.
* `response_code` — (Number) API response code.

List order follows the API; use `filter` or `for` expressions to select a stable named environment.
