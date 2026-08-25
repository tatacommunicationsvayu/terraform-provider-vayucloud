---
page_title: "vayucloud_s3_user Data Source"
subcategory: ""
description: |-
  Reads a single S3 user by domain ID and user name.
---

# `vayucloud_s3_user`

```hcl
data "vayucloud_s3_user" "example" {
  domain_id = vayucloud_s3_domain.example.id
  user_name = "app-service-user"
}
```

## Argument Reference

### Required

* `domain_id` — (String) Domain identifier.
* `user_name` — (String) User name.

## Attributes Reference

* `id` — (String) `{domain_id}:{user_name}`.
* `status`, `message`, `response_code` — API metadata.
