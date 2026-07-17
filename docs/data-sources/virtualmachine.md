---
page_title: "vayucloud_virtualmachine Data Source"
subcategory: ""
description: |-
  Reads detailed attributes for a single virtual machine by instance identifier (GET vm-instances detail API).
---

# `vayucloud_virtualmachine`

Returns full detail for one VM via the `vm-instances/{instanceId}` API. To list all VMs in a zone, use [`vayucloud_virtualmachine_list`](virtualmachine_list.md).

## Example Usage

```hcl
data "vayucloud_virtualmachine" "example" {
  instance_id = {{instance_id}}
}

output "vm_ip" {
  value = data.vayucloud_virtualmachine.example.ip
}
```

## Argument Reference

### Required

* `instance_id` — (String) Virtual machine instance identifier.

## Attributes Reference

* `id` — (String) Computed placeholder for Terraform.
* `name` — (String) Virtual machine name.
* `hostname` — (String) Hostname.
* `zone_id` — (Number) Network zone identifier.
* `ip` — (String) Primary IP address.
* `root_disk` — (Number) Root disk size in GB.
* `power_status` — (String) Power state (for example `ACTIVE`).
* `flavor_id` — (Number) Flavor identifier.
* `image_id` — (Number) Image identifier.
* `os_type` — (String) Operating system family.
* `os_version` — (String) Operating system version.
* `os_model` — (String) OS model string.
* `os_make` — (String) OS vendor or make.
* `os_service_pack` — (String) Service pack when applicable.
* `pricing_model` — (String) Pricing model (for example `hourly`).
* `vcpu` — (Number) vCPU count.
* `vram` — (Number) Memory in MB.
* `volumes` — (List of Object) Attached volumes; each object includes `id`, `name`, `size`, `disk_type`, `created_date`, and `iops` as returned by the API.
* `status` — (String) API status (for example `success`).
* `message` — (String) API message.
* `response_code` — (Number) API response code (the provider documents `200` as success in schema; confirm against your environment).
