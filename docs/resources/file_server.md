---
page_title: "vayucloud_file_server Resource"
subcategory: ""
description: |-
  Creates a NAS file server (vserver) in VayuCloud. Create and delete are asynchronous; there is no in-place update API.
---

# `vayucloud_file_server`

Creates a NAS **vserver** for file storage. The provider validates engagement, endpoint, and storage type at plan time. Updates are a no-op (only `audit_id` / `status` refresh).

Changing `engagement_id`, `endpoint_id`, `vserver_name`, or `file_storage_type` forces replacement.

## Example Usage

```hcl
resource "vayucloud_file_server" "nas" {
  engagement_id     = var.engagement_id
  endpoint_id       = var.endpoint_id
  vserver_name      = "nfsFileServer1"
  file_storage_type = "NAS-NFS" # or CIFS
}
```

## Argument Reference

### Required

* `engagement_id` — (Number) Engagement identifier.
* `endpoint_id` — (Number) Endpoint identifier.
* `vserver_name` — (String) vServer name. Changing forces replacement.

### Optional

* `file_storage_type` — (String) `NAS-NFS` (default) or `CIFS`. Changing forces replacement.

## Attributes Reference

* `id` — (String) vServer identifier.
* `audit_id` — (String) Audit ID from create/delete.
* `status` — (String) Audit completion status.

## Import

```shell
terraform import vayucloud_file_server.example <vserver_id>
```

Composite form `engagement_id/vserver_id` is also accepted; `engagement_id` must be recoverable for destroy.

## See also

* [`vayucloud_file_storage_volume`](file_storage_volume.md)
* [`vayucloud_file_storage_export_policy`](file_storage_export_policy.md)
* [`vayucloud_nas_vlan_zone`](../data-sources/nas_vlan_zone.md)
