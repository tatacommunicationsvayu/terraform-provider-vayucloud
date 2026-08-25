---
page_title: "vayucloud_file_server Data Source"
subcategory: ""
description: |-
  Reads a NAS file server (vserver) by vserver ID.
---

# `vayucloud_file_server`

```hcl
data "vayucloud_file_server" "example" {
  vserver_id = vayucloud_file_server.nas.id
}
```

## Argument Reference

### Required

* `vserver_id` — (String) vServer identifier.

## Attributes Reference

* `id` — (String) `fileserver-{vserver_id}`.
* `file_server_name` — (String) vServer name.
* `engagement_id`, `endpoint_id` — (Number) Parent scope.
* `file_storage_type` — (String) `NAS-NFS` or `CIFS`.
* `description` — (String) When returned by the API.

## See also

* [`vayucloud_file_server_list`](file_server_list.md)
