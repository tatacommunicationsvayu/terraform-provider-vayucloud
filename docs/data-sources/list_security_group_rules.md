---
page_title: "vayucloud_list_security_group_rules Data Source"
subcategory: ""
description: |-
  Lists security group rules for a security group on a firewall. Optional filters narrow results client-side.
---

# `vayucloud_list_security_group_rules`

Returns rules for one security group. Use optional `filter` blocks to restrict rows (for example by `protocol`, `direction`, or `id`).

## Example Usage

```hcl
data "vayucloud_list_security_group_rules" "example" {
  firewall_id       = {{firewall_id}}
  security_group_id = {{security_group_id}}
}
```

## Argument Reference

### Required

* `firewall_id` — (Number) Firewall identifier that owns the security group.
* `security_group_id` — (String) Security group UUID whose rules should be listed.

### Optional

* `filter` — (Block list) Client-side filters. Each block has:
  * `name` — (String) Attribute name.
  * `values` — (List of String) Values to match.

## Attributes Reference

* `id` — (String) Placeholder identifier (`firewall_id/security_group_id`).
* `rules` — (List of Object) Rules returned by the API. Each object includes:
  * `id` — (String) Rule UUID.
  * `security_group_id` — (String) Parent security group UUID.
  * `protocol` — (String) Protocol (for example `tcp`, `udp`, `icmp`).
  * `ether_type` — (String) Ether type (`IPv4` / `IPv6`).
  * `direction` — (String) `ingress` or `egress`.
  * `port_range_min` — (Number) Minimum port when set.
  * `port_range_max` — (Number) Maximum port when set.
  * `remote_ip_prefix` — (String) Remote CIDR when using a CIDR remote.
  * `remote_group_id` — (String) Remote security group UUID when using a group remote.
  * `description` — (String) Rule description.

## See also

* [`vayucloud_list_security_group`](list_security_group.md) — List security groups on a firewall.
* [`vayucloud_security_group`](../resources/security_group.md) — Manage a security group and rules.
