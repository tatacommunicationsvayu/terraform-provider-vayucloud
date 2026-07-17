---
page_title: "vayucloud_virtualmachine_blockstorage Data Source"
subcategory: ""
description: |-
  Reads one attached block storage volume on a virtual machine by instance ID and volume ID (instance detail API).
---

# `vayucloud_virtualmachine_blockstorage`

Returns attributes for a **single** attached volume on a VM. Use [`vayucloud_virtualmachine`](virtualmachine.md) if you need the full instance, including every volume in `volumes`.

Arguments use **numbers** (`instance_id`, `volume_id`), consistent with the block storage **resource**.

## Example Usage

```hcl
data "vayucloud_virtualmachine_blockstorage" "disk" {
  instance_id = {{instance_id}}
  volume_id   = {{volume_id}}
}

output "disk_size_gb" {
  value = data.vayucloud_virtualmachine_blockstorage.disk.size
}
```

## Argument Reference

### Required

* `instance_id` — (Number) Virtual machine instance identifier.
* `volume_id` — (Number) Attached volume (disk) identifier.

## Attributes Reference

* `id` — (String) Computed datastore key from the provider; do not rely on a specific formatting—use `instance_id` and `volume_id` as the stable inputs.
* `name` — (String) Volume name.
* `size` — (Number) Size in GB.
* `iops` — (Number) IOPS value for the volume.
* `disk_type` — (String) Disk type from the API (for example `SSD`, `HDD`, or `root` when applicable).
* `created_date` — (String) Volume creation date from the API.
