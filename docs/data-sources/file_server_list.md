---
page_title: "vayucloud_file_server_list Data Source"
subcategory: ""
description: |-
  Lists NAS file servers for an engagement and endpoint.
---

# `vayucloud_file_server_list`

```hcl
data "vayucloud_file_server_list" "all" {
  engagement_id = var.engagement_id
  endpoint_id   = var.endpoint_id
}
```

## Argument Reference

### Required

* `engagement_id` — (Number) Engagement identifier.
* `endpoint_id` — (Number) Endpoint identifier.

## Attributes Reference

* `vservers` — (List of Object) `vserver_id`, `vserver_name`, `engagement_id`, `endpoint_id`, `file_storage_type`, `description`.
* `status`, `message`, `response_code`, `raw_response` — API metadata.
