---
page_title: "vayucloud_virtualmachine_blockstorage Resource"
subcategory: ""
description: |-
  Attaches, resizes, or removes an extra block storage volume on a VayuCloud virtual machine. Create, resize, and delete use platform validation and audit polling.
---

# `vayucloud_virtualmachine_blockstorage`

Manages **one** attachable block volume for an existing virtual machine: **create** runs attach-volume validation (also at plan time for new attachments), then attaches the volume and waits for audit completion; **update** increases `size` using the same validate-and-resize flow as [`vayucloud_virtualmachine`](virtualmachine.md); **delete** removes the volume from the instance.

This resource is for **additional** disks. The VM’s root disk remains on `vayucloud_virtualmachine` (`root_disk_size` / `root_disk_id`). Changing `instance_id`, `name`, or `iops` forces replacement because the platform treats them as part of the attachment identity.

## Example Usage

```hcl
resource "vayucloud_virtualmachine_blockstorage" "data_disk" {
  instance_id = vayucloud_virtualmachine.app.id
  name        = "app_data_vol"
  size        = 100
  iops        = 3
}
```

## Argument Reference

### Required

* `instance_id` — (Number) Virtual machine instance identifier. Changing forces replacement.
* `name` — (String) Volume name. Between 5 and 45 characters; alphanumeric, spaces, and underscores only. Changing forces replacement.
* `size` — (Number) Size in GB. Must be between **10** and **5000** and a **multiple of 10**. Can be increased in place; decreasing size is not supported by this update path.
* `iops` — (Number) IOPS tier: **1**, **3**, or **5**. Changing forces replacement.

### Optional / computed

* `disk_type` — (String) Returned by the API (for example `SSD`). The provider currently constrains input to `SSD` when set explicitly; otherwise it is populated after create/read.

## Attributes Reference

In addition to the arguments above, the following attributes are exported:

* `id` — (Number) Volume (disk) identifier after attach.
* `created_date` — (String) Creation timestamp from the API.
* `audit_id` — (String) Audit ID from the last completed async operation.
* `status` — (String) Status from that operation.

## Import

Import volumes using **`instance_id,volume_id`** (comma-separated integers):

```text
terraform import vayucloud_virtualmachine_blockstorage.data_disk 12345,67890
```

After import, run `terraform plan` to reconcile optional/computed fields.
