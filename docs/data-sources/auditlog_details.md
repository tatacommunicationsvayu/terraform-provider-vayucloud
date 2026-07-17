---
page_title: "vayucloud_auditlog_details Data Source"
subcategory: ""
description: |-
  Returns full detail for a single audit log entry by audit identifier.
---

# `vayucloud_auditlog_details`

Reads one audit record, including inputs, outputs, comments, and metadata. Use [`vayucloud_account_engagement_auditlog`](account_engagement_auditlog.md) to discover `audit_id` values.

## Example Usage

```hcl
data "vayucloud_auditlog_details" "example" {
  audit_id = {{audit_id}}
}
```

## Argument Reference

### Required

* `audit_id` — (String) Audit entry identifier.

## Attributes Reference

* `id` — (String) Computed placeholder for Terraform.
* `updated_time` — (String) Last update time.
* `resource_id` — (String) Related resource identifier.
* `updated_by` — (String) User who last updated the entry.
* `comments` — (List of Object) Each object includes `updated_time`, `comment_type`, `updated_by`, `comment_id`, and `comment`.
* `response_status` — (String) Response status.
* `resource_category` — (String) Resource category.
* `output` — (String) Operation output payload.
* `input` — (String) Operation input payload.
* `environment` — (String) Environment context when present.
* `created_by` — (String) Creating user.
* `action` — (String) Recorded action.
* `created_time` — (String) Creation time.
* `engagement_id` — (String) Engagement identifier.
* `resource_type` — (String) Resource type.
* `status` — (String) Entry status.
