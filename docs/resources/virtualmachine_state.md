---
page_title: "vayucloud_virtualmachine_state Resource"
subcategory: ""
description: |-
  Manages power and reboot actions for an existing VayuCloud virtual machine without creating or destroying the VM.
---

# `vayucloud_virtualmachine_state`

Controls power and reboot behavior for a virtual machine that already exists. This resource does **not** create or delete the VM; it only invokes platform power APIs for the given `instance_id`.

Create and update paths wait on audit completion when an action runs. Read refreshes `power_status`. Destroy removes this resource from state only and does **not** change cloud power state.

## Behavior summary

| Terraform operation | Behavior |
|---------------------|----------|
| Create | Runs the configured `action` and waits for audit completion. |
| Update | Runs when `action` or `instance_id` changes. If current power status already matches the target for the new action, the power API may be skipped. |
| Read | Refreshes `power_status` only. |
| Destroy | Drops management of this resource in Terraform; **no** power API call. |

If `action` is unchanged but power was changed outside Terraform, the next apply may not call the API until `action` changes or you adjust configuration. See `power_status` after refresh.

**Actions:** `power_off`, `power_on`, `suspend`, `hard_reboot`, `soft_reboot`, `resume`.

## Example Usage

```hcl
resource "vayucloud_virtualmachine" "app" {
  # ...
}

resource "vayucloud_virtualmachine_state" "power" {
  instance_id = tonumber(vayucloud_virtualmachine.app.id)
  action      = "power_off"
}
```

You may set `instance_id` to a numeric VM ID when the instance is not managed in the same configuration.

## Argument Reference

### Required

* `instance_id` — (Number) Virtual machine identifier. Changing forces replacement.
* `action` — (String) One of `power_off`, `power_on`, `suspend`, `hard_reboot`, `soft_reboot`, `resume`.

## Attributes Reference

* `id` — (String) Same value as `instance_id` (string form), computed.
* `power_status` — (String) Current power status after read or after an action.
* `audit_id` — (String) Audit identifier for the last action that invoked the API (may reflect a no-op when skipped).
* `status` — (String) Status of the last audited action.

## Import

Import is not typically used for this resource; prefer declaring `instance_id` and `action` in configuration.
