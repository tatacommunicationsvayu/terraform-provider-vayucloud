---
page_title: "vayucloud_s3_domain_list Data Source"
subcategory: ""
description: |-
  Lists S3 (VCS) domains for an engagement and endpoint.
---

# `vayucloud_s3_domain_list`

Enumerates S3 domains. Optional `filter` blocks narrow results client-side.

## Example Usage

```hcl
data "vayucloud_s3_domain_list" "all" {
  engagement_id = var.engagement_id
  endpoint_id   = var.endpoint_id
  firewall_id   = var.firewall_id
}
```

## Argument Reference

### Required

* `engagement_id` — (Number) Engagement identifier.
* `endpoint_id` — (Number) Endpoint identifier.

### Optional

* `firewall_id` — (Number) Filter by firewall when set.
* `filter` — (Block) Optional name/value filters on the returned list.

## Attributes Reference

* `id` — (String) Computed placeholder.
* `domains` — (List of Object) Each entry includes `s3_domain_id`, `domain_name_fqdn`, `quota`, `storage_class`, `variant`, `engagement_id`, `endpoint_id`, `domain_access_ip`, `domain_access_public_ip`.
* `status`, `message`, `response_code` — API metadata.
