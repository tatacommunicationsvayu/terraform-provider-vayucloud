---
page_title: "vayucloud_virtualmachine_security_group_association Resource"
subcategory: ""
description: |-
  Associates one or more security groups with a virtual machine. Attach and detach are asynchronous; the provider polls audit logs.
---

# `vayucloud_virtualmachine_security_group_association`

Attaches security groups to a virtual machine. Destroy detaches the same set. Changing `security_group_ids` detaches removed IDs and attaches newly added IDs.

The provider validates that `instance_id` exists at plan time. Changing `instance_id` forces replacement.

## Example Usage

```hcl
resource "vayucloud_virtualmachine_security_group_association" "example" {
  instance_id        = {{instance_id}}
  security_group_ids = [
    vayucloud_security_group.example.id,
  ]
}
```

## Argument Reference

### Required

* `instance_id` — (Number) Virtual machine instance ID. Changing forces replacement.
* `security_group_ids` — (List of String) Security group UUIDs to associate. At least one value; values must be unique.

## Attributes Reference

* `id` — (String) Same as `instance_id` as a string.
* `audit_id` — (String) Audit identifier from the last completed operation.
* `status` — (String) Audit status from the last completed operation.

## See also

* [`vayucloud_security_group`](security_group.md) — Create or adopt security groups and rules.
* [`vayucloud_virtualmachine_list_security_group_rules`](../data-sources/virtualmachine_list_security_group_rules.md) — Read groups and rules currently on a VM.
