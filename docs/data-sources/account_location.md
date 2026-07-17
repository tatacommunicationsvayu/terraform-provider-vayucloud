---
page_title: "vayucloud_account_location Data Source"
subcategory: ""
description: |-
  Lists locations (endpoints) for a VayuCloud engagement. Supports optional filter blocks.
---

# `vayucloud_account_location`

Returns endpoints associated with the given `engagement_id`. Use `filter` blocks to narrow the list client-side.

## Example Usage

```hcl
data "vayucloud_account_location" "example" {
  engagement_id = {{engagement_id}}
}
```

## Argument Reference

### Required

* `engagement_id` — (Number) Engagement identifier.

### Optional

* `filter` — (Block) Repeatable. Each block requires `name` and `values` (list of strings).

## Attributes Reference

* `id` — (String) Computed placeholder for Terraform.
* `locations` — (List of Object) Each object includes `endpoint_id` and `endpoint_display_name`.
* `status` — (String) API status (for example `success`).
* `message` — (String) Additional API message text.
* `response_code` — (Number) API response code (`0` indicates success).
