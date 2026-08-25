---
page_title: "vayucloud_s3_token_list Data Source"
subcategory: ""
description: |-
  Lists S3 access tokens in a domain.
---

# `vayucloud_s3_token_list`

```hcl
data "vayucloud_s3_token_list" "all" {
  domain_id = vayucloud_s3_domain.example.id
  user_name = "app-service-user"
}
```

## Argument Reference

### Required

* `domain_id` — (String) Domain identifier.

### Optional

* `user_name` — (String) Filter tokens for one user.
* `access_key_id` — (String) Filter to one access key.
* `filter` — (Block) Client-side filters.

## Attributes Reference

* `tokens` — (List of Object) `domain_id`, `user_name`, `access_key_id`, `token_expiry_date`, `token_description`.
* `status`, `message`, `response_code` — API metadata.
