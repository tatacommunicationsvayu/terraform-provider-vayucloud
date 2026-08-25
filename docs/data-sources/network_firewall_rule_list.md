---
page_title: "vayucloud_network_firewall_rule_list Data Source"
subcategory: ""
description: |-
  Lists network firewall rules for a firewall via the action-state API (module=firewallRule, action=list). Optional filter blocks narrow results client-side.
---

# `vayucloud_network_firewall_rule_list`

Returns all firewall rules on a firewall. Use [`vayucloud_network_firewall_rule`](network_firewall_rule.md) to read one rule by ID.

## Example Usage

```hcl
data "vayucloud_network_firewall_rule_list" "on_firewall" {
  firewall_id = {{firewall_id}}
}

output "rule_names" {
  value = [for r in data.vayucloud_network_firewall_rule_list.on_firewall.rules : r.rule_name]
}
```

### Filter by action

```hcl
data "vayucloud_network_firewall_rule_list" "allow_rules" {
  firewall_id = {{firewall_id}}

  filter {
    name   = "action"
    values = ["allow"]
  }
}
```

### Filter by rule name

```hcl
data "vayucloud_network_firewall_rule_list" "named" {
  firewall_id = {{firewall_id}}

  filter {
    name   = "rule_name"
    values = [{{rule_name}}]
  }
}
```

## Argument Reference

### Required

* `firewall_id` — (Number) Firewall resource ID to list rules for.

### Optional

* `filter` — (Block) Repeatable client-side filter. Each block requires `name` and `values`. Multiple filters use **AND** logic; multiple values use **OR** logic with case-insensitive exact matching.

  Supported filter names: `id`, `rule_name`, `firewall_id`, `source`, `action`, `source_zone_id`, `destination`, `destination_zone_id`, `schedule_start_date`, `schedule_end_date`, `status`.

## Attributes Reference

* `id` — (String) Same as `firewall_id` as a string.
* `rules` — (List of Object) Each object includes:
  * `id` — (String) Rule identifier.
  * `rule_name` — (String) Rule name.
  * `firewall_id` — (Number) Firewall resource ID.
  * `source` — (String) Source type.
  * `action` — (String) `allow` or `deny`.
  * `source_zone_id` — (Number) Source zone ID when applicable.
  * `source_addresses` — (List of String) Source CIDRs.
  * `destination` — (String) Destination type.
  * `destination_zone_id` — (Number) Destination zone ID when applicable.
  * `destination_addresses` — (List of String) Destination CIDRs.
  * `schedule_start_date` — (String) Schedule start when set.
  * `schedule_end_date` — (String) Schedule end when set.
  * `services` — (List of String) Service names.
  * `status` — (String) Rule status from the platform.
* `status` — (String) Top-level API response status.
* `message` — (String) API message text.
* `response_code` — (Number) API response code.

## See also

* [`vayucloud_network_firewall_rule`](network_firewall_rule.md) — Read one rule by ID.
* [`vayucloud_network_firewall_rule`](../resources/network_firewall_rule.md) — Create and manage rules.
