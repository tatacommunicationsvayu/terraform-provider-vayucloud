---
page_title: "vayucloud_s3_object Data Source"
subcategory: ""
description: |-
  Reads S3 object metadata (not body) by domain, bucket, and key.
---

# `vayucloud_s3_object`

Returns object metadata only — **does not download object content**.

```hcl
data "vayucloud_s3_object" "example" {
  domain_id   = vayucloud_s3_domain.example.id
  bucket_name = "my-bucket"
  key         = "data/hello.txt"
}
```

## Argument Reference

### Required

* `domain_id` — (String) Domain identifier.
* `bucket_name` — (String) Bucket name.
* `key` — (String) Object key.

## Attributes Reference

* `etag`, `size`, `content_type`, `last_modified` — Object metadata.
* `prefix`, `file_name` — Parsed from `key` when applicable.
* `status`, `message`, `response_code` — API metadata.
