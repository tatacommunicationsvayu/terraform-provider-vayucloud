---
page_title: "vayucloud_s3_user_list Data Source"
subcategory: ""
description: |-
  Lists S3 users in a domain.
---

# `vayucloud_s3_user_list`

```hcl
data "vayucloud_s3_user_list" "all" {
  domain_id = vayucloud_s3_domain.example.id
}
```

## Argument Reference

### Required

* `domain_id` — (String) Domain identifier.

### Optional

* `user_name` — (String) Server-side filter by user name.
* `filter` — (Block) Client-side filters.

## Attributes Reference

* `users` — (List of Object) `id`, `domain_id`, `user_name`.
* `status`, `message`, `response_code` — API metadata.
