---
page_title: "vayucloud_network_firewall_rule Data Source"
subcategory: ""
description: |-
  Reads a network firewall rule by firewall ID and rule ID via the action-state API (module=firewallRule, action=read).
---

# `vayucloud_network_firewall_rule`

Returns the current state of one firewall rule. This uses the same read path as the [`vayucloud_network_firewall_rule`](../resources/network_firewall_rule.md) resource refresh.

## Example Usage

```hcl
data "vayucloud_network_firewall_rule" "example" {
  firewall_id = {{firewall_id}}
  id          = {{rule_id}}
}
```

### Look up a rule by name from the list data source

```hcl
data "vayucloud_network_firewall_rule_list" "all" {
  firewall_id = {{firewall_id}}
}

locals {
  rule_id = one([
    for r in data.vayucloud_network_firewall_rule_list.all.rules :
    r.id if r.rule_name == {{rule_name}}
  ])
}

data "vayucloud_network_firewall_rule" "by_name" {
  firewall_id = {{firewall_id}}
  id          = local.rule_id
}
```

## Argument Reference

### Required

* `firewall_id` — (Number) Firewall resource ID the rule belongs to.
* `id` — (String) Rule identifier on the firewall (same as the resource `id` attribute).

## Attributes Reference

* `rule_name` — (String) Rule name from the platform.
* `source` — (String) Source type (`internet`, `zone`, or `nas`).
* `action` — (String) Rule action (`allow` or `deny`).
* `source_zone_id` — (Number) Source zone ID when applicable; `null` otherwise.
* `source_addresses` — (List of String) Source CIDRs.
* `destination` — (String) Destination type (`internet`, `zone`, `nas`, `vcs`, or `load_balancer`).
* `destination_zone_id` — (Number) Destination zone ID when applicable; `null` otherwise.
* `destination_addresses` — (List of String) Destination CIDRs.
* `schedule_start_date` — (String) Schedule start (`YYYY-MM-DD`) when set; `null` for always-on rules.
* `schedule_end_date` — (String) Schedule end (`YYYY-MM-DD`) when set; `null` for always-on rules.
* `services` — (List of String) Service names (for example `HTTP`, `HTTPS`, `NFS`).
* `status` — (String) Rule status from the platform.
* `message` — (String) API message text.
* `response_code` — (Number) API response code (`0` indicates success).
* `raw_response` — (String) Raw JSON `data` payload from the action-state response (for debugging).

## See also

* [`vayucloud_network_firewall_rule_list`](network_firewall_rule_list.md) — List all rules on a firewall.
* [`vayucloud_network_firewall_rule`](../resources/network_firewall_rule.md) — Manage rules with Terraform.
