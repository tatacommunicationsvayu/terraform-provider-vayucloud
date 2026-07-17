---
page_title: "vayucloud_virtualmachine_flavor Data Source"
subcategory: ""
description: |-
  Lists virtual machine flavors available for a zone. Optional filter blocks narrow results client-side.
---

# `vayucloud_virtualmachine_flavor`

Returns size and pricing options for `zone_id`. Each flavor includes vCPU, memory, and related metadata from the platform.

## Example Usage

```hcl
data "vayucloud_virtualmachine_flavor" "all" {
  zone_id = {{zone_id}}
}

data "vayucloud_virtualmachine_flavor" "by_name" {
  zone_id = {{zone_id}}

  filter {
    name   = "name"
    values = ["B6"]
  }
}
```

## Argument Reference

### Required

* `zone_id` — (String) Network zone identifier.

### Optional

* `filter` — (Block) Repeatable. Each block requires `name` and `values`.

## Attributes Reference

* `id` — (String) Computed placeholder for Terraform.
* `flavors` — (List of Object) Each object includes:
  * `id` — (Number) Flavor identifier.
  * `name` — (String) Flavor name.
  * `artifact_type`, `os_model` — (String) Classification fields from the API.
  * `vcpu`, `vram`, `vgpu`, `vdisk_l`, `vdisk_w` — (Number) Capacity fields.
  * `p2r_pricing_model` — (Object) Nested `ppu` and `reserved` strings for pricing tiers.
  * `root_storage_partition_linux` — (Object) Optional Linux partition hints (`usr`, `swap`, `root`, `boot`, `kdump`).
  * `root_storage_partition_windows` — (Object) Optional Windows partition hints (`c_drive`, `page`).
* `status` — (String) API status.
* `message` — (String) API message.
* `response_code` — (Number) API response code (`0` indicates success).

Use `flavors[*].id` when setting `flavor_id` on [`vayucloud_virtualmachine`](../resources/virtualmachine.md).
