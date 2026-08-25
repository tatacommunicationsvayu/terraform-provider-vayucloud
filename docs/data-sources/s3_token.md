---
page_title: "vayucloud_s3_token Data Source"
subcategory: ""
description: |-
  Reads S3 token metadata by domain, user, and access key ID. Does not return the secret.
---

# `vayucloud_s3_token`

```hcl
data "vayucloud_s3_token" "example" {
  domain_id     = vayucloud_s3_domain.example.id
  user_name     = "app-service-user"
  access_key_id = "abc123..."
}
```

## Argument Reference

### Required

* `domain_id` — (String) Domain identifier.
* `user_name` — (String) User name.
* `access_key_id` — (String) Access key ID.

## Attributes Reference

* `token_expiry_date`, `token_description` — Token metadata.
* `status`, `message`, `response_code` — API metadata.

**Note:** `secret_access_key` is never returned by this data source.
