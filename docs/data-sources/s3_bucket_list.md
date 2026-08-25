---
page_title: "vayucloud_s3_bucket_list Data Source"
subcategory: ""
description: |-
  Lists buckets in an S3 domain.
---

# `vayucloud_s3_bucket_list`

```hcl
data "vayucloud_s3_bucket_list" "all" {
  domain_id = vayucloud_s3_domain.example.id
}
```

## Argument Reference

### Required

* `domain_id` — (String) Domain identifier.

### Optional

* `filter` — (Block) Client-side filters on `buckets`.

## Attributes Reference

* `buckets` — (List of Object) `id`, `domain_id`, `bucket_name`, `versioning_status`, `created_at`, `storage_class`.
* `status`, `message`, `response_code` — API metadata.
