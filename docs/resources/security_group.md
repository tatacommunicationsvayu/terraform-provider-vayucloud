---
page_title: "vayucloud_security_group Resource"
subcategory: ""
description: |-
  Manages a VayuCloud security group and its rules on a firewall. Create, rule changes, and delete are asynchronous where the platform requires audit polling.
---

# `vayucloud_security_group`

Manages a security group on a firewall and optionally creates rules under it.

Choose **exactly one** creation mode:

1. **Create group** — set `name` (and optional `description`) to create a new security group.
2. **Adopt by ID** — set `id` to an existing security group UUID (group create is skipped).
3. **Adopt by type** — set `type` (`zone`, `virtualmachine`, or `firewall`) and `resource_id` to locate `SG_Zone_<id>`, `SG_VM_<id>`, or `SG_TR_<id>`.

Each `rule` block creates a security group rule. Provide either `remote_ip_prefix` or `remote_security_group` (optional when `protocol` is `icmp`).

The provider validates `firewall_id` at plan time. Changing `firewall_id`, `id`, `name`, `description`, `type`, or `resource_id` forces replacement.

`managed_security_group` is `true` when this resource created the group (destroy deletes the group). It is `false` when an existing group was adopted (destroy removes only rules created by this resource).

## Example Usage

### Create a security group and rules

```hcl
resource "vayucloud_security_group" "example" {
  firewall_id = {{firewall_id}}
  name        = "tf_sg_example"
  description = "Created by Terraform"

  rule {
    protocol         = "tcp"
    ether_type       = "ipv4"
    direction        = "ingress"
    port_range       = 80
    remote_ip_prefix = ["0.0.0.0/0"]
  }

  rule {
    protocol   = "icmp"
    ether_type = "ipv4"
    direction  = "ingress"
  }
}
```

### Adopt an existing group by UUID

```hcl
resource "vayucloud_security_group" "adopted" {
  firewall_id = {{firewall_id}}
  id          = {{security_group_id}}

  rule {
    protocol         = "tcp"
    ether_type       = "ipv4"
    direction        = "ingress"
    port_range       = 22
    remote_ip_prefix = ["10.0.0.0/8"]
  }
}
```

### Adopt a system group by type and resource ID

```hcl
resource "vayucloud_security_group" "vm_system" {
  firewall_id = {{firewall_id}}
  type        = "virtualmachine"
  resource_id = {{instance_id}}

  rule {
    protocol              = "tcp"
    ether_type            = "ipv4"
    direction             = "ingress"
    port_range            = 8080
    remote_security_group = "tf_sg_example"
  }
}
```

## Argument Reference

### Required

* `firewall_id` — (Number) Firewall identifier that owns the security group. Must exist. Changing forces replacement.

### Optional (exactly one create/adopt mode)

* `name` — (String) Security group name for create mode. Letters, digits, and underscore only. Changing forces replacement.
* `description` — (String) Description for create mode. Changing forces replacement.
* `id` — (String) Existing security group UUID for adopt-by-ID mode. Computed after create/lookup otherwise. Changing forces replacement.
* `type` — (String) Resource type for adopt-by-type mode: `zone`, `virtualmachine`, or `firewall`. Changing forces replacement.
* `resource_id` — (Number) Platform resource ID used with `type` to look up the system security group. Changing forces replacement.

### Nested `rule` blocks

Order does not matter; rules are matched by attributes.

* `protocol` — (String) `tcp`, `udp`, or `icmp`.
* `ether_type` — (String) `ipv4` or `ipv6`.
* `direction` — (String) `ingress` or `egress`.
* `port_range` — (Number) Single port (1–65535). Omit for all ports.
* `remote_ip_prefix` — (List of String) Remote CIDRs. Mutually exclusive with `remote_security_group`.
* `remote_security_group` — (String) Remote security group name. Mutually exclusive with `remote_ip_prefix`.
* `id` — (String) Rule UUID. Computed after apply.

## Attributes Reference

* `managed_security_group` — (Boolean) `true` if this resource created the group.
* `audit_id` — (String) Audit identifier for the last completed operation.
* `status` — (String) Audit status from the last completed operation.

## Import

```shell
terraform import vayucloud_security_group.example {{firewall_id}}/{{security_group_id}}
```

OpenTofu: use `tofu import` with the same identifier. After import, `managed_security_group` is `false`.

## See also

* [`vayucloud_virtualmachine_security_group_association`](virtualmachine_security_group_association.md) — Attach groups to a VM.
* [`vayucloud_list_security_group`](../data-sources/list_security_group.md) — List groups on a firewall.
