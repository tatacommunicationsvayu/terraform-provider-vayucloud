---
page_title: "vayucloud_virtualmachine_list_security_group_rules Data Source"
subcategory: ""
description: |-
  Lists security groups and rules associated with a virtual machine, grouped by port. Optional filters narrow results client-side.
---

# `vayucloud_virtualmachine_list_security_group_rules`

Returns security groups attached to a virtual machine, grouped by network port:

```
ports[]
  ├─ port_id / port_name / port_ip
  └─ security_groups[]
       ├─ id / name
       └─ rules[]
```

Optional `filter` blocks match each port and security-group attachment using `port_id`, `port_name`, `port_ip`, `security_group_id`, or `security_group_name`.

## Example Usage

```hcl
data "vayucloud_virtualmachine_list_security_group_rules" "example" {
  instance_id = {{instance_id}}
}
```

## Argument Reference

### Required

* `instance_id` — (Number) Virtual machine instance ID.

### Optional

* `filter` — (Block list) Client-side filters. Each block has:
  * `name` — (String) Attribute name.
  * `values` — (List of String) Values to match.

## Attributes Reference

* `id` — (String) Placeholder identifier (`instance_id`).
* `ports` — (List of Object) VM ports with attached security groups. Each object includes:
  * `port_id` — (String) Port UUID.
  * `port_name` — (String) Port display name.
  * `port_ip` — (String) Primary IP on the port.
  * `security_groups` — (List of Object) Groups attached to this port. Each object includes:
    * `id` — (String) Security group UUID.
    * `name` — (String) Security group name.
    * `rules` — (List of Object) Rules on the group. Each object includes:
      * `id` — (String) Rule UUID.
      * `protocol` — (String) Protocol when set.
      * `ether_type` — (String) Ether type.
      * `direction` — (String) `ingress` or `egress`.
      * `port_range_min` — (Number) Minimum port when set.
      * `port_range_max` — (Number) Maximum port when set.
      * `remote_ip_prefix` — (String) Remote CIDR when set.
      * `remote_group_id` — (String) Remote security group UUID when set.
      * `normalized_cidr` — (String) Normalized remote CIDR from the platform.
      * `description` — (String) Rule description.
      * `created_at` — (String) Creation timestamp (RFC3339).
      * `updated_at` — (String) Last update timestamp (RFC3339).

## See also

* [`vayucloud_virtualmachine_security_group_association`](../resources/virtualmachine_security_group_association.md) — Attach groups to a VM.
* [`vayucloud_security_group`](../resources/security_group.md) — Manage a security group.
