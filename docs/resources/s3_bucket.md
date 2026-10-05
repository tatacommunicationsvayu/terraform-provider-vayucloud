---
page_title: "vayucloud_s3_bucket Resource"
subcategory: ""
description: |-
  Manages an S3 bucket in a VCS domain. Operations are synchronous; delete fails if the bucket is not empty.
---

# `vayucloud_s3_bucket`

Creates and deletes a bucket within an S3 domain. **Versioning** may be toggled in place.

**Delete** fails with *Bucket Not Empty* if objects remain — empty the bucket out of band (for example with an S3-compatible client) before destroy.

## Example Usage

```hcl
resource "vayucloud_s3_bucket" "example" {
  domain_id   = vayucloud_s3_domain.example.id
  bucket_name = "my-bucket"
}
```

## Argument Reference

### Required

* `domain_id` — (String) Parent domain ID. Changing forces replacement.
* `bucket_name` — (String) Bucket name (unique within domain). Changing forces replacement.

### Optional

* `versioning_enabled` — (Boolean) Enable versioning. Default `false`. May be updated in place.

## Attributes Reference

* `id` — (String) `{domain_id}:{bucket_name}`.
* `versioning_status` — (String) Platform versioning status.
* `created_at` — (String) Creation timestamp.
* `storage_class` — (String) Storage class.
* `audit_id` — (String) Operation tracking ID when returned.

## Import

```shell
terraform import vayucloud_s3_bucket.example <domain_id>:<bucket_name>
```

## See also

* [`vayucloud_s3_domain`](s3_domain.md)
