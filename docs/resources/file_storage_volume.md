---
page_title: "vayucloud_file_storage_volume Resource"
subcategory: ""
description: |-
  Manages a NAS file storage volume. Create, grow-only resize, and delete are asynchronous.
---

# `vayucloud_file_storage_volume`

Creates and manages a NAS volume on a file server. **Size can only increase** in place; shrinking errors at apply.

The provider validates the parent file server and NAS ordering at plan time. Out-of-band size growth in the portal is adopted on refresh/plan.

## Example Usage

```hcl
resource "vayucloud_file_storage_volume" "nas" {
  engagement_id     = var.engagement_id
  endpoint_id       = var.endpoint_id
  file_server_id    = vayucloud_file_server.nas.id
  name              = "nfsVolume1"
  size_gb           = 30
  file_storage_type = "NFS"
  usage_type        = "reserved"
  pricing_model     = "reserved_1"
}
```

## Argument Reference

### Required

* `engagement_id` — (Number) Engagement identifier. Changing forces replacement.
* `endpoint_id` — (Number) Endpoint identifier. Changing forces replacement.
* `file_server_id` — (String) Parent vServer ID. Changing forces replacement.
* `name` — (String) Volume name. Changing forces replacement.
* `size_gb` — (Number) Size in GB. May be increased in place only.
* `usage_type` — (String) For example `ppu`, `reserved`. Changing forces replacement.
* `pricing_model` — (String) For example `hourly`, `reserved_1`. Changing forces replacement.

### Optional

* `volume_type` — (String) Default `Flex Volume`. Changing forces replacement.
* `file_storage_type` — (String) For example `NFS`. Changing forces replacement.
* `iops` — (Number) IOPS. Changing forces replacement.

## Attributes Reference

* `id` — (String) Volume identifier.
* `audit_id` — (String) Audit ID from async operations.
* `status` — (String) Audit completion status.

## Import

```shell
terraform import vayucloud_file_storage_volume.example <engagement_id>,<file_server_id>,<volume_id>,<name>
```

Legacy `file_server_id,volume_id` is supported; set `name` and `engagement_id` in configuration when using legacy import.

## See also

* [`vayucloud_file_server`](file_server.md)
* [`vayucloud_file_storage_export_policy`](file_storage_export_policy.md)
