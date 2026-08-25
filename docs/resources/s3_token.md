---
page_title: "vayucloud_s3_token Resource"
subcategory: ""
description: |-
  Manages an S3 access token for a domain user. Synchronous create/delete; secret is only available at create.
---

# `vayucloud_s3_token`

Creates an access key / secret pair for an S3 user.

* **`secret_access_key`** is only returned at **create** and preserved in Terraform state. It is **not** recoverable on import or read.
* **No update API** — changing expiry or description forces replacement.
* **AI_STANDARD domains:** Only `user_name = "root"` is allowed on create; delete is blocked (rotate by creating a new token on STANDARD domains).

## Example Usage

```hcl
resource "vayucloud_s3_token" "app" {
  domain_id         = vayucloud_s3_domain.example.id
  user_name         = vayucloud_s3_user.app.user_name
  token_expiry_date = "2027-06-23"
  token_description = "Terraform-managed access token"
}
```

## Argument Reference

### Required

* `domain_id` — (String) Domain ID. Changing forces replacement.
* `user_name` — (String) Parent user name. Changing forces replacement.

### Optional

* `token_expiry_date` — (String) Expiry date `YYYY-MM-DD`. Changing forces replacement.
* `token_description` — (String) Description (required on create for `STANDARD` storage class). Changing forces replacement.

## Attributes Reference

* `id` — (String) `{domain_id}:{user_name}:{access_key_id}`.
* `access_key_id` — (String) Access key ID.
* `secret_access_key` — (String, sensitive) Secret key (create-only).
* `audit_id` — (String) When returned by the platform.

## Import

```shell
terraform import vayucloud_s3_token.example <domain_id>:<user_name>:<access_key_id>
```

`secret_access_key` will be unknown after import. Rotate the token if you need a new secret in state.

## See also

* [`vayucloud_s3_user`](s3_user.md)
