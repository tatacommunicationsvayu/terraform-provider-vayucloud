---
page_title: "vayucloud_list_security_group Data Source"
subcategory: ""
description: |-
  Lists security groups on a firewall. Optional filters narrow results client-side.
---

# `vayucloud_list_security_group`

Returns security groups for a firewall. Use optional `filter` blocks to restrict rows (for example by `name`, `id`, or `description`).

## Example Usage

```hcl
data "vayucloud_list_security_group" "all" {
  firewall_id = {{firewall_id}}
}

data "vayucloud_list_security_group" "by_name" {
  firewall_id = {{firewall_id}}

  filter {
    name   = "name"
    values = [{{security_group_name}}]
  }
}
```

## Argument Reference

### Required

* `firewall_id` — (Number) Firewall identifier whose security groups should be listed.

### Optional

* `filter` — (Block list) Client-side filters. Each block has:
  * `name` — (String) Attribute name.
  * `values` — (List of String) Values to match.

## Attributes Reference

* `id` — (String) Placeholder identifier (`firewall_id`).
* `security_groups` — (List of Object) Groups returned by the API. Each object includes:
  * `id` — (String) Security group UUID.
  * `name` — (String) Name.
  * `description` — (String) Description.
  * `tenant_id` — (String) Tenant identifier.
  * `project_id` — (String) Project identifier.
  * `shared` — (Boolean) Whether the group is shared.
  * `stateful` — (Boolean) Whether the group is stateful.
  * `revision_number` — (Number) Revision number.
  * `created_at` — (String) Creation timestamp (RFC3339).
  * `updated_at` — (String) Last update timestamp (RFC3339).

## See also

* [`vayucloud_security_group`](../resources/security_group.md) — Manage a security group.
* [`vayucloud_list_security_group_rules`](list_security_group_rules.md) — List rules for one group.
