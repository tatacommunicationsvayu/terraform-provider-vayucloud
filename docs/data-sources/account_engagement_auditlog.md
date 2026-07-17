---
page_title: "vayucloud_account_engagement_auditlog Data Source"
subcategory: ""
description: |-
  Lists audit log entries for an engagement. Supports server-side request parameters and client-side filter blocks.
---

# `vayucloud_account_engagement_auditlog`

Returns audit history for an engagement. Use optional attributes such as `start_date`, `end_date`, and `action` to limit what the API returns; use repeatable `filter` blocks for additional client-side matching on the result set.

## Example Usage

```hcl
data "vayucloud_account_engagement_auditlog" "recent" {
  engagement_id = {{engagement_id}}
  start_date      = {{start_date}}
  end_date        = {{end_date}}
}

data "vayucloud_account_engagement_auditlog" "filtered" {
  engagement_id = {{engagement_id}}

  filter {
    name   = "action"
    values = [{{action}}]
  }
}
```

## Argument Reference

### Required

* `engagement_id` — (Number) Engagement identifier.

### Optional

* `start_date` — (String) Start of date range (server-side filter).
* `end_date` — (String) End of date range (server-side filter).
* `action` — (List of String) Action names (server-side filter).
* `created_by` — (String) User filter (server-side).
* `request_status` — (String) Request status filter (server-side).
* `audit_id` — (String) Specific audit identifier (server-side).
* `resource_category` — (String) Resource category filter (server-side).
* `resource_id` — (String) Resource identifier filter (server-side).
* `filter` — (Block) Repeatable client-side filter; each block requires `name` and `values`.

## Attributes Reference

* `id` — (String) Computed placeholder for Terraform.
* `audit_logs` — (List of Object) Each entry includes `audit_id`, `engagement_id`, `resource_type`, `resource_id`, `resource_category`, `action`, `description`, `created_by`, `created_time`, `updated_by`, `updated_time`, `status`, `snow_ticket_id`, `cluster_name`, and `cluster_status`.
* `total_pages` — (Number) Page count from the API.
* `total_elements` — (Number) Total matching elements.
* `api_status` — (String) Overall API status string.

Prefer server-side parameters for large tenants to reduce payload size.
