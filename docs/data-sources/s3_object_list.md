---
page_title: "vayucloud_s3_object_list Data Source"
subcategory: ""
description: |-
  Lists objects in an S3 bucket.
---

# `vayucloud_s3_object_list`

```hcl
data "vayucloud_s3_object_list" "all" {
  domain_id   = vayucloud_s3_domain.example.id
  bucket_name = "my-bucket"
  prefix      = "data/"
}
```

## Argument Reference

### Required

* `domain_id` — (String) Domain identifier.
* `bucket_name` — (String) Bucket name.

### Optional

* `prefix` — (String) Key prefix filter.
* `filter` — (Block) Additional client-side filters.

## Attributes Reference

* `objects` — (List of Object) Metadata per object (`key`, `etag`, `size`, `last_modified`, etc.).
* `status`, `message`, `response_code` — API metadata.
