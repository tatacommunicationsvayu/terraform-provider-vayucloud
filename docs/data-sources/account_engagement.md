---
page_title: "vayucloud_account_engagement Data Source"
subcategory: ""
description: |-
  Lists account engagements for the authenticated user. Optional filter blocks narrow results after the API returns data.
---

# `vayucloud_account_engagement`

Lists engagements available to the current credentials. Use optional `filter` blocks to restrict rows by attribute (for example engagement name).

## Example Usage

```hcl
data "vayucloud_account_engagement" "all" {}

data "vayucloud_account_engagement" "by_name" {
  filter {
    name   = "engagement_name"
    values = [{{engagement_name}}]
  }
}
```

## Argument Reference

### Optional

* `filter` — (Block) Repeatable. Each block requires:
  * `name` — (String) Attribute name on each returned object to match.
  * `values` — (List of String) Values to accept.

## Attributes Reference

* `id` — (String) Computed placeholder for Terraform.
* `engagements` — (List of Object) Each object includes `engagement_name`, `id`, `service_name`, and `customer_name`.
* `status` — (String) API status (for example `success`).
* `message` — (String) Additional API message text.
* `response_code` — (Number) API response code (`200` indicates success).

Without filters, all engagements returned by the API are exposed.
