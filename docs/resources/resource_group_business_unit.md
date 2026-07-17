---
page_title: "vayucloud_resource_group_business_unit Resource"
subcategory: ""
description: |-
  Provides a business unit under a VayuCloud firewall. Create and update operations are asynchronous where the platform requires audit polling.
---

# `vayucloud_resource_group_business_unit`

Provides a business unit scoped to a firewall. The provider validates `firewall_id` before create and appends the configured provider username to the `users` list.

Only the `business_unit` name can be updated in place. Changing `firewall_id` forces replacement.

## Example Usage

```hcl
resource "vayucloud_resource_group_business_unit" "example" {
  firewall_id   = {{firewall_id}}
  business_unit = "Engineering"
  users         = ["admin@example.com"]
}
```

## Argument Reference

### Required

* `firewall_id` — (Number) Firewall identifier. Must exist. Forces new resource if changed.
* `business_unit` — (String) Business unit name. Length 5–45; alphanumeric, `_`, and `-` allowed. Can be updated in place.

### Optional

* `users` — (List of String) Additional user email addresses. The provider username is always included by the platform.

## Attributes Reference

* `id` — (String) Business unit resource identifier.
* `audit_id` — (String) Audit identifier for the latest operation.
* `status` — (String) Audit status (for example `SUCCESS`, `FAILED`, `IN_PROGRESS`).

## Import

```shell
terraform import vayucloud_resource_group_business_unit.example <business_unit_id>
```

OpenTofu: use `tofu import` with the same arguments.

```shell
terraform import vayucloud_resource_group_business_unit.example {{business_unit_id}}
```
