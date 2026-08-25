---
page_title: "vayucloud_file_storage Data Source"
subcategory: ""
description: |-
  Reads a NAS file storage volume by engagement, file server, and volume name.
---

# `vayucloud_file_storage`

Looks up a single NAS volume and its export clients.

```hcl
data "vayucloud_file_storage" "example" {
  engagement_id  = var.engagement_id
  file_server_id = vayucloud_file_server.nas.id
  name           = "nfsVolume1"
}
```

## Argument Reference

### Required

* `engagement_id` — (Number) Engagement identifier.
* `file_server_id` — (String) vServer identifier.
* `name` — (String) Volume name.

## Attributes Reference

* `volume_id` — (String) Volume identifier.
* `file_server_name` — (String) Parent vServer name.
* `size_gb` — (Number) Volume size.
* `clients` — (List) Export client entries when returned.
* `raw_response` — (String) Raw API JSON.

## See also

* [`vayucloud_file_storage_volume`](../resources/file_storage_volume.md)
