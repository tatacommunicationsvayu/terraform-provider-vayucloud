---
page_title: "vayucloud_s3_user Resource"
subcategory: ""
description: |-
  Manages an S3 service user in a VCS domain. Synchronous create/delete; no in-place update API.
---

# `vayucloud_s3_user`

Creates a service user for S3 API access. **No update API** — changes to `user_name` force replacement.

**AI_STANDARD domains:** Create and delete are **blocked** by the provider. Use `user_name = "root"` and manage tokens via [`vayucloud_s3_token`](s3_token.md) only.

**Delete** cascades all tokens for the user.

## Example Usage

```hcl
resource "vayucloud_s3_user" "app" {
  domain_id = vayucloud_s3_domain.example.id
  user_name = "app-service-user"
}
```

## Argument Reference

### Required

* `domain_id` — (String) Domain ID. Changing forces replacement.
* `user_name` — (String) User name. Changing forces replacement.

## Attributes Reference

* `id` — (String) `{domain_id}:{user_name}`.
* `audit_id` — (String) When returned by the platform.

## Import

```shell
terraform import vayucloud_s3_user.example <domain_id>:<user_name>
```

## See also

* [`vayucloud_s3_token`](s3_token.md)
