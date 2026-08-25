---
page_title: "vayucloud_network_zone Resource"
subcategory: ""
description: |-
  Provides a VayuCloud network zone for virtual machine placement. Create and update operations are asynchronous where the platform requires audit polling.
---

# `vayucloud_network_zone`

Provides a network zone (logical segment with IP addressing). The provider validates `firewall_id` and `environment_id` before create.

Only `name` can be changed in place. Changing other zone inputs (environment, firewall, IP/CIDR sizing, zone type, purpose, and related fields) normally **recreates** the zone because those attributes carry `RequiresReplace` in the provider schema.

**Dual stack:** there is no separate `dual_stack_mode` argument in Terraform. If you set `no_of_v6_ips`, the provider sends **`dualStackMode`: `yes`** to the API together with IPv6 sizing; otherwise it sends **`no`**.

**Data plane:** `data_plane` is `Auto IPAM` (default) or **`Data Plane CIDR`**. For Auto IPAM, `no_of_ips` is required. For Data Plane CIDR, set **`cidr`** (the API request is built accordingly in code).

## Example Usage

```hcl
resource "vayucloud_network_zone" "example" {
  name           = "dev-zone"
  environment_id = {{environment_id}}
  firewall_id    = {{firewall_id}}
  no_of_ips      = 20
}
```

## Argument Reference

### Required

* `name` — (String) Zone name. Length 5–45. Can be updated in place.
* `environment_id` — (Number) Environment identifier. Must exist. Forces new resource if changed.
* `firewall_id` — (Number) Firewall identifier. Must exist. Forces new resource if changed.
* `no_of_ips` — (Number) Number of IP addresses. Forces new resource if changed.

### Optional

* `purpose` — (String) Zone purpose. Default `IPC`. Changing forces replacement.
* `data_plane` — (String) `Auto IPAM` or `Data Plane CIDR`. Default `Auto IPAM`. Changing forces replacement.
* `cidr` — (String) IPv4 CIDR. Required when `data_plane` is `Data Plane CIDR`. For Auto IPAM, populated from the platform after create (computed). Changing a configured value forces replacement.
* `zone_type` — (String) `overlay` or `vlan`. Default `overlay`. Changing forces replacement.
* `no_of_v6_ips` — (Number) IPv6 address count for **dual-stack** zones. When this is set (non-null), the provider enables dual stack on create; when omitted, dual stack is off.
* `ipv6_cidr` — (String) IPv6 CIDR for dual-stack configurations when you supply it.

## Attributes Reference

* `id` — (String) Zone resource identifier (from provisioning).
* `network_zone_id` — (Number) Network zone identifier returned/read from the platform (computed).
* `audit_id` — (String) Audit identifier for the latest operation.
* `status` — (String) Audit status (for example `SUCCESS`, `FAILED`, `IN_PROGRESS`).

## Import

```shell
terraform import vayucloud_network_zone.example <zone_id>
```

OpenTofu: use `tofu import` with the same arguments.

```shell
terraform import vayucloud_network_zone.example {{zone_id}}
```
