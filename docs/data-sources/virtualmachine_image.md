---
page_title: "vayucloud_virtualmachine_image Data Source"
subcategory: ""
description: |-
  Lists virtual machine images (templates) available for a zone. Optional type, OS, and filter arguments narrow results.
---

# `vayucloud_virtualmachine_image`

Returns deployable images for `zone_id`. The provider validates that the zone exists before calling the templates API.

Optional arguments `type` and `os_make` restrict results server-side. Repeatable `filter` blocks apply additional client-side matching on the returned list.

## Example Usage

```hcl
data "vayucloud_virtualmachine_image" "all" {
  zone_id = {{zone_id}}
}

data "vayucloud_virtualmachine_image" "ubuntu" {
  zone_id = {{zone_id}}
  type    = {{type}}
  os_make = {{os_make}}
}
```

## Argument Reference

### Required

* `zone_id` — (String) Network zone identifier.

### Optional

* `type` — (String) Hypervisor or image type filter (for example `KVM`, `VCD_ESXI`).
* `os_make` — (String) Operating system make filter (for example `Ubuntu`, `Windows`).
* `filter` — (Block) Repeatable. Each block requires `name` and `values`.

## Attributes Reference

* `id` — (String) Computed placeholder for Terraform.
* `images` — (List of Object) Each object includes `id`, `os_type`, `name`, `os_make`, `os_model`, `os_version`, and `os_service_pack`.
* `status` — (String) API status.
* `message` — (String) API message.
* `response_code` — (Number) API response code (`0` indicates success).

Use `images[*].id` when setting `image_id` on [`vayucloud_virtualmachine`](../resources/virtualmachine.md).
