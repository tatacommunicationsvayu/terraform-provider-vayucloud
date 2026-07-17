---
page_title: "vayucloud_virtualmachine Resource"
subcategory: ""
description: |-
  Provides a VayuCloud virtual machine. Create and delete are asynchronous; the provider polls audit logs. New resources run pre-create validation at plan time. Flavor and disk sizes can be updated in place.
---

# `vayucloud_virtualmachine`

Provides a virtual machine. The platform returns an audit identifier for create; the provider polls until provisioning completes.

Prerequisites: valid `zone_id`, `image_id`, and `flavor_id` (typically from `vayucloud_virtualmachine_image` and `vayucloud_virtualmachine_flavor` data sources).

For **new** resources, the provider calls the platform **pre-create validate** API during `terraform plan` when the instance `id` is not yet known, so invalid combinations fail early.

## In-place updates

The following can be changed **without** replacing the VM (other arguments still force replacement as documented):

* **`flavor_id`** — Resize flavor: validate API, then resize API, then audit and action-state.  
* **`root_disk_size`** — Root volume resize (requires valid `root_disk_id` in state from a previous read/apply).  
* **`additional_disk` sizes** — Only when the number of additional disks in plan matches state **and** each entry corresponds by index; per-disk size increases run validate + resize + wait.

Adding or removing `additional_disk` blocks or reordering disks is **not** handled by this update path; use replacement or separate resources as appropriate.

## Example Usage

### Basic virtual machine

Use image and flavor data sources to resolve identifiers:

```hcl
data "vayucloud_virtualmachine_image" "os" {
  zone_id = {{zone_id}}
  filter {
    name   = "name"
    values = [{{image_name}}]
  }
}

data "vayucloud_virtualmachine_flavor" "size" {
  zone_id = {{zone_id}}
  filter {
    name   = "name"
    values = [{{flavor_name}}]
  }
}

resource "vayucloud_virtualmachine" "example" {
  name       = "app-vm"
  vm_purpose = "WEB"
  image_id   = data.vayucloud_virtualmachine_image.os.images[0].id
  flavor_id  = data.vayucloud_virtualmachine_flavor.size.flavors[0].id
  zone_id    = {{zone_id}}
  iops       = 1

  is_kdump_or_page_enabled = "No"

  usage_type    = "ppu"
  pricing_model = "hourly"

  root_disk_size = 150
}
```

### Public IP (optional)

When `assign_public_ip` is `yes`, the create request includes public IP assignment, retention on termination, and pricing model.

```hcl
resource "vayucloud_virtualmachine" "with_public_ip" {
  name       = "app-vm"
  vm_purpose = "WEB"
  image_id   = data.vayucloud_virtualmachine_image.os.images[0].id
  flavor_id  = data.vayucloud_virtualmachine_flavor.size.flavors[0].id
  zone_id    = {{zone_id}}
  iops       = 1

  is_kdump_or_page_enabled = "No"

  public_ip = {
    assign_public_ip                 = "yes"
    retain_public_ip_on_termination   = "no"
    public_ip_pricing_model          = "daily"
  }
}
```

After create, `public_ip.ip` is populated from the platform when a public IP exists. Schema defaults in the provider: `assign_public_ip` → `no`; `retain_public_ip_on_termination` → **`yes`**; `public_ip_pricing_model` → `daily`. If you omit `public_ip.public_ip_pricing_model` but set `assign_public_ip` to yes, the create request uses the top-level `pricing_model` value for the public IP when the API expects it.

### Root partitions and additional disks

```hcl
resource "vayucloud_virtualmachine" "with_disks" {
  name       = "app-vm"
  vm_purpose = "WEB"
  image_id   = data.vayucloud_virtualmachine_image.os.images[0].id
  flavor_id  = data.vayucloud_virtualmachine_flavor.size.flavors[0].id
  zone_id    = {{zone_id}}
  iops       = 1

  is_kdump_or_page_enabled = "No"

  usage_type    = "ppu"
  pricing_model = "hourly"

  root_disk_size = 150

  root_disk_partitions = [
    { partition = "/root", size = 72 },
    { partition = "/boot", size = 2 },
    { partition = "/usr",  size = 10 },
    { partition = "/swap", size = 16 }
  ]

  additional_disk = [
    { size = 100, iops = 1 }
  ]
}
```

## Argument Reference

### Required

* `name` — (String) Virtual machine name. Changing forces replacement.
* `vm_purpose` — (String) Purpose code (for example `WEB`, `DB`, `OTHERS`). Changing forces replacement.
* `image_id` — (Number) Image identifier. Changing forces replacement.
* `flavor_id` — (Number) Flavor identifier. **May be changed in place** to resize the instance flavor (async).
* `zone_id` — (Number) Network zone identifier. Changing forces replacement.
* `iops` — (Number) IOPS for the root disk. Changing forces replacement.
* `is_kdump_or_page_enabled` — (String) `yes` / `Yes` or `no` / `No` per OS requirements (for example `Yes` for Windows / RHEL / SUSE, `No` for Ubuntu / Rocky). Changing forces replacement.

### Optional

* `usage_type` — (String) Usage type (for example `ppu`, `reserved`). Default `ppu`. Changing forces replacement.
* `pricing_model` — (String) Pricing model (for example `hourly`, `daily`, `monthly`, `reserved_1`). Default `daily`. Changing forces replacement.
* `root_disk_size` — (Number) Root disk size in GB. May be computed if omitted; can be increased in place when `root_disk_id` is known.
* `root_disk_id` — (Number) Root disk identifier. Computed after create/read from the platform; used for root volume resize.
* `root_disk_partitions` — (List of Object) Partition layout. Each element supports:
  * `partition` — (String) Exactly one of `/root`, `/boot`, `/usr`, `/swap` (Linux-style) or `/cDrive`, `/page` (Windows-style).
  * `size` — (Number) Size in GB.
* `public_ip` — (Object, optional) Public IP settings. Attributes:
  * `assign_public_ip` — (String) `yes` / `Yes` or `no` / `No`. Default `no`. When `yes`, create sends public IP options to the API.
  * `retain_public_ip_on_termination` — (String) `yes` / `Yes` or `no` / `No`. Default **`yes`**. When the VM is deleted, this maps to the platform delete policy (`yes` / unknown → retain public IP where supported; explicit `no` → release).
  * `public_ip_pricing_model` — (String) When set explicitly, must be one of `daily`, `monthly`, `reserved_1`, `reserved_2`, `reserved_3` (per provider validation). Default `daily`. If unset while `assign_public_ip` is yes, the VM’s `pricing_model` is sent instead.
  * `ip` — (String) Assigned public IP. Computed from the platform after apply when applicable.
* `additional_disk` — (List of Object) Extra disks. Each element supports:
  * `size` — (Number) Disk size in GB. Required in configuration.
  * `iops` — (Number) IOPS. Required in configuration.
  * `id`, `name`, `disk_type`, `created_date` — (Number / String) Computed after apply when returned by the platform.

## Attributes Reference

* `id` — (String) Virtual machine instance (resource) identifier.
* `power_status` — (String) Power status from the platform.
* `audit_id` — (String) Audit identifier for the last provisioning operation.
* `status` — (String) Audit completion status (for example success or failure).

Nested `additional_disk` entries also export populated computed fields where the API returns them.

## Import

```shell
terraform import vayucloud_virtualmachine.example <vm_id>
```

OpenTofu: use `tofu import` with the same identifier.

```shell
terraform import vayucloud_virtualmachine.example {{vm_id}}
```

The import ID is the VM instance ID (same as `id` after create).
