---
page_title: "vayucloud_resource_group_environment Resource"
subcategory: ""
description: |-
  Provides an environment within a VayuCloud business unit. Provisioning is asynchronous; the provider waits on audit completion.
---

# `vayucloud_resource_group_environment`

Provides an environment (for example development or production) within a business unit. Requires an existing firewall and business unit.

The provider validates `firewall_id` and `business_unit_id` before create, update, and delete. The `environment` name can be updated in place. Changing `firewall_id` forces replacement.

## Example Usage

```hcl
resource "vayucloud_resource_group_environment" "production" {
  firewall_id      = tonumber(vayucloud_network_firewall.example.id)
  business_unit_id = tonumber(vayucloud_resource_group_business_unit.example.id)
  environment      = "Production"
}
```

Multiple environments under the same business unit:

```hcl
resource "vayucloud_resource_group_environment" "development" {
  firewall_id      = tonumber(vayucloud_network_firewall.example.id)
  business_unit_id = tonumber(vayucloud_resource_group_business_unit.example.id)
  environment      = "Development"
}

resource "vayucloud_resource_group_environment" "production" {
  firewall_id      = tonumber(vayucloud_network_firewall.example.id)
  business_unit_id = tonumber(vayucloud_resource_group_business_unit.example.id)
  environment      = "Production"
}
```

## Argument Reference

### Required

* `firewall_id` — (Number) Firewall identifier. Must reference an existing firewall. Forces new resource if changed.
* `business_unit_id` — (Number) Business unit identifier. Must exist.
* `environment` — (String) Environment name. Length 5–45; alphanumeric with `_` and `-` allowed. Can be updated in place.

## Attributes Reference

* `id` — (String) Environment resource identifier.
* `audit_id` — (String) Audit identifier for the create or update operation.
* `status` — (String) Final audit status (for example `SUCCESS`, `FAILED`).

## Import

```shell
terraform import vayucloud_resource_group_environment.example <environment_id>
```

OpenTofu: use `tofu import` with the same arguments.

```shell
terraform import vayucloud_resource_group_environment.example {{environment_id}}
```
