---
page_title: "vayucloud_s3_bucket Data Source"
subcategory: ""
description: |-
  Reads a single S3 bucket by domain ID and bucket name.
---

# `vayucloud_s3_bucket`

```hcl
data "vayucloud_s3_bucket" "example" {
  domain_id   = vayucloud_s3_domain.example.id
  bucket_name = "my-bucket"
}
```

## Argument Reference

### Required

* `domain_id` — (String) Domain identifier.
* `bucket_name` — (String) Bucket name.

## Attributes Reference

* `id` — (String) `{domain_id}:{bucket_name}`.
* `versioning_status`, `created_at`, `storage_class` — Bucket metadata.
* `status`, `message`, `response_code` — API metadata.
