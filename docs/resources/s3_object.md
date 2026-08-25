---
page_title: "vayucloud_s3_object Resource"
subcategory: ""
description: |-
  Uploads and manages an object in an S3 bucket. Operations are synchronous; body changes trigger re-upload.
---

# `vayucloud_s3_object`

Uploads or replaces an object in a bucket. Provide **exactly one** of `source`, `content`, or `content_base64`.

## Example Usage

### File upload

```hcl
resource "vayucloud_s3_object" "file" {
  domain_id   = vayucloud_s3_domain.example.id
  bucket_name = vayucloud_s3_bucket.example.bucket_name
  key         = "data/hello.txt"
  source      = "${path.module}/files/hello.txt"
  content_type = "text/plain"
}
```

### Inline content

```hcl
resource "vayucloud_s3_object" "inline" {
  domain_id   = vayucloud_s3_domain.example.id
  bucket_name = vayucloud_s3_bucket.example.bucket_name
  key         = "test/inline.txt"
  content     = "hello from terraform"
  content_type = "text/plain"
}
```

## Argument Reference

### Required

* `domain_id` — (String) Domain ID. Changing forces replacement.
* `bucket_name` — (String) Bucket name. Changing forces replacement.
* `key` — (String) Object key. Changing forces replacement.
* **Body (exactly one):**
  * `source` — (String) Local file path.
  * `content` — (String) Inline UTF-8 content.
  * `content_base64` — (String) Base64-encoded content.

### Optional

* `content_type` — (String) MIME type. Default `application/octet-stream`.

## Attributes Reference

* `id` — (String) `{domain_id}:{bucket_name}:{key}`.
* `etag` — (String) Object ETag after upload.
* `size` — (Number) Object size in bytes.
* `last_modified` — (String) Last modified timestamp.
* `source_hash` — (String) Hex SHA-256 hash of the local `source` file at last upload (for drift detection when the path is unchanged).
* `prefix`, `file_name` — (String) Parsed from `key` when applicable.
* `audit_id` — (String) When returned by the platform.

## Import

```shell
terraform import vayucloud_s3_object.example <domain_id>:<bucket_name>:<object_key>
```

Re-run apply to align body content with configuration after import.

## See also

* [`vayucloud_s3_bucket`](s3_bucket.md)
