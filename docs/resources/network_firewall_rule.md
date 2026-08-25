---
page_title: "vayucloud_network_firewall_rule Resource"
subcategory: ""
description: |-
  Manages a network firewall rule on a VayuCloud firewall. Create, update, and delete are asynchronous; the provider polls audit logs until completion.
---

# `vayucloud_network_firewall_rule`

Manages a firewall rule (allow/deny) on an existing network firewall. The provider validates `firewall_id` and zone requirements during plan when the API client is configured.

**Optional zone IDs:** `source_zone_id` and `destination_zone_id` are optional in configuration. Omit them (or set `null`) when not applicable; the provider does not populate them from the API when omitted in config, even if the platform infers a zone internally.

**Schedules:** `schedule_start_date` and `schedule_end_date` are optional (`YYYY-MM-DD`). Omit both for an always-on rule.

## Plan-time validation

`ModifyPlan` checks:

* `firewall_id` exists on the platform.
* When `source` is `zone` or `nas`, `source_zone_id` is required unless `source` is `zone` with public-IP `/32` source addresses.
* When `destination` is `zone` or `nas`, `destination_zone_id` is required unless `destination` is `zone` with public-IP `/32` destination addresses.

## In-place updates

May be updated without replacement: `rule_name`, `action`, `source_addresses`, `destination_addresses`, `services`, `schedule_start_date`, `schedule_end_date`.

Changing `firewall_id`, `source`, `source_zone_id`, `destination`, or `destination_zone_id` forces replacement.

## Example Usage

### Internet to zone (public IP on a VM)

```hcl
resource "vayucloud_network_firewall_rule" "internet_to_zone" {
  rule_name   = "ill_to_app"
  firewall_id = tonumber(vayucloud_network_firewall.bandwidth.id)
  source      = "internet"
  destination = "zone"
  action      = "allow"
  services    = ["HTTP", "HTTPS"]

  source_addresses      = ["0.0.0.0/0"]
  destination_addresses = ["${vayucloud_virtualmachine.app.public_ip.ip}/32"]
}
```

### Zone to NAS (bidirectional NFS pattern)

```hcl
resource "vayucloud_network_firewall_rule" "zone_to_nas" {
  rule_name           = "zone_to_nas"
  firewall_id         = local.firewall_id
  source              = "zone"
  source_zone_id      = local.app_zone_id
  destination         = "nas"
  destination_zone_id = local.nas_vlan_zone_id
  action              = "allow"
  services            = ["NFS"]

  source_addresses      = [local.app_zone_cidr]
  destination_addresses = [local.nas_vlan_cidr]

  schedule_start_date = "2026-06-29"
  schedule_end_date   = "2026-06-30"
}
```

### Internet to VCS (object storage)

```hcl
resource "vayucloud_network_firewall_rule" "ill_to_vcs" {
  rule_name   = "ill_to_vcs"
  firewall_id = tonumber(vayucloud_network_firewall.bandwidth.id)
  source      = "internet"
  destination = "vcs"
  action      = "allow"
  services    = ["HTTP", "HTTPS", "ALL_ICMP"]

  source_addresses      = ["0.0.0.0/0"]
  destination_addresses = ["${vayucloud_network_public_ip.vcs.public_ip}/32"]
}
```

### Internet to load balancer public IP

```hcl
resource "vayucloud_network_firewall_rule" "internet_to_lb" {
  rule_name   = "ill_to_lb"
  firewall_id = tonumber(vayucloud_network_firewall.bandwidth.id)
  source      = "internet"
  destination = "load_balancer"
  action      = "allow"
  services    = ["HTTP", "HTTPS"]

  source_addresses      = ["0.0.0.0/0"]
  destination_addresses = ["${vayucloud_network_public_ip.vs_vip.public_ip}/32"]
}
```

## Argument Reference

### Required

* `rule_name` — (String) Rule name. Alphanumeric, `_`, and `-` only.
* `firewall_id` — (Number) Firewall resource ID. Changing forces replacement.
* `source` — (String) `internet`, `zone`, or `nas`. Changing forces replacement.
* `destination` — (String) `internet`, `zone`, `nas`, `vcs`, or `load_balancer`. Changing forces replacement.
* `action` — (String) `allow` or `deny`.
* `source_addresses` — (List of String) Source CIDRs (for example `0.0.0.0/0`, `10.0.0.1/32`).
* `destination_addresses` — (List of String) Destination CIDRs.
* `services` — (List of String) Service names (for example `HTTP`, `HTTPS`, `SSH`, `NFS`, `ALL_ICMP`).

### Optional

* `source_zone_id` — (Number) Required when `source` is `zone` or `nas` (with exceptions above). Omit when not used. Changing forces replacement.
* `destination_zone_id` — (Number) Required when `destination` is `zone` or `nas` (with exceptions above). Omit when not used. Changing forces replacement.
* `schedule_start_date` — (String) Schedule start (`YYYY-MM-DD`). Omit for no schedule.
* `schedule_end_date` — (String) Schedule end (`YYYY-MM-DD`). Omit for no schedule.

## Attributes Reference

* `id` — (String) Rule identifier on the firewall.
* `audit_id` — (String) Audit ID from the last async operation.
* `status` — (String) Audit completion status.

## Import

Import ID format: `firewall_id,rule_id`

```shell
terraform import vayucloud_network_firewall_rule.example 408628,22
```

After import, align configuration with state (especially optional zone IDs and schedules) to avoid unwanted replacement.

## See also

* [`vayucloud_network_firewall`](network_firewall.md) — Parent firewall.
* [`vayucloud_network_firewall_rule`](../data-sources/network_firewall_rule.md) — Read one rule by ID.
* [`vayucloud_network_firewall_rule_list`](../data-sources/network_firewall_rule_list.md) — List rules on a firewall.
* [`vayucloud_network_public_ip`](network_public_ip.md) — Public IP targets for `internet` rules.
* Example: `examples/IaaS/complete/fw_vm_nas_complete/`
