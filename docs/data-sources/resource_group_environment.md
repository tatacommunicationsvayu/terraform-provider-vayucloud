---
page_title: "vayucloud_resource_group_environment Data Source"
subcategory: ""
description: |-
  Reads a single resource group environment by identifier.
---

# `vayucloud_resource_group_environment`

Returns details for one environment. To list environments in a business unit, use [`vayucloud_resource_group_environment_list`](resource_group_environment_list.md).

## Example Usage

```hcl
data "vayucloud_resource_group_environment" "example" {
  resource_group_environment_id = {{resource_group_environment_id}}
}
```

## Argument Reference

### Required

* `resource_group_environment_id` — (String) Environment resource identifier.

## Attributes Reference

* `id` — (String) Computed placeholder for Terraform.
* `environment` — (String) Environment name.
* `business_unit_id` — (Number) Parent business unit identifier.
* `status` — (String) API status.
* `message` — (String) API message.
* `response_code` — (Number) API response code (`0` indicates success).
