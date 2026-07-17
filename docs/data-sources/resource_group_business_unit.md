---
page_title: "vayucloud_resource_group_business_unit Data Source"
subcategory: ""
description: |-
  Reads a single business unit by resource identifier.
---

# `vayucloud_resource_group_business_unit`

Returns details for one business unit. To list all business units on a firewall, use [`vayucloud_resource_group_business_unit_list`](resource_group_business_unit_list.md).

## Example Usage

```hcl
data "vayucloud_resource_group_business_unit" "example" {
  resource_group_business_unit_id = {{resource_group_business_unit_id}}
}
```

## Argument Reference

### Required

* `resource_group_business_unit_id` — (String) Business unit resource identifier.

## Attributes Reference

* `id` — (String) Computed placeholder for Terraform.
* `business_unit` — (String) Business unit name.
* `users` — (List of String) Associated user email addresses.
* `status` — (String) API status (for example `success`).
* `message` — (String) API message.
* `response_code` — (Number) API response code (`0` indicates success).
