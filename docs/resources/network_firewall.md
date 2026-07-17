---
page_title: "vayucloud_network_firewall Resource"
subcategory: ""
description: |-
  Provides a VayuCloud network firewall. Create and update operations are asynchronous; the provider polls audit logs until completion.
---

# `vayucloud_network_firewall`

Provides a network firewall in VayuCloud. The platform returns an audit identifier immediately; the provider polls the audit log until the operation finishes.

The provider verifies that `engagement_id` and `endpoint_id` exist before create. On **create**, you must supply **at least one** of `internet_bandwidth` or `minimum_commitment` (the provider errors if both are absent). The effective access mode is derived when `access_type` is not set: omitting or leaving `minimum_commitment` empty selects **Bandwidth** (you must then set `internet_bandwidth`); omitting `internet_bandwidth` selects **DataTransfer** (you must then set `minimum_commitment`). In **Bandwidth** mode, `firewall_throughput` must be greater than or equal to `internet_bandwidth` (client validation). If `access_type` is set explicitly, it must match the corresponding field (`Bandwidth` → `internet_bandwidth` required; `DataTransfer` → `minimum_commitment` required).

Updates to `firewall_display_name`, throughput, and bandwidth-related fields follow the resource update path in code. Changing `engagement_id`, `endpoint_id`, or `firewall_type` forces replacement.

## Example Usage

### Bandwidth-based internet

```hcl
resource "vayucloud_network_firewall" "example" {
  engagement_id         = {{engagement_id}}
  endpoint_id           = {{endpoint_id}}
  firewall_display_name = "my-firewall"
  firewall_throughput   = "10Mbps"
  internet_bandwidth    = "10Mbps"
}
```

### Data transfer (minimum commitment)

```hcl
resource "vayucloud_network_firewall" "datatransfer" {
  engagement_id         = {{engagement_id}}
  endpoint_id           = {{endpoint_id}}
  firewall_display_name = "my-firewall-dt"
  firewall_throughput    = "10Mbps"
  minimum_commitment     = "500GB"
}
```

Omitting optional arguments applies schema defaults where documented (for example `hypervisor`, booleans, pricing defaults in Terraform — the create request includes pricing fields only when explicitly set in configuration).

## Argument Reference

The following arguments are supported:

### Required

* `engagement_id` — (Number) Engagement identifier. Must exist in the platform.
* `endpoint_id` — (Number) Endpoint identifier for the engagement. Must exist.
* `firewall_display_name` — (String) Display name. Length 5–45. Allowed characters: letters, digits, `_`, `-`.
* `firewall_throughput` — (String) Throughput in the form `XMbps` (for example `10Mbps`). Validated against bandwidth or minimum commitment depending on access mode.

### Conditionally required on create

At least one of the following must be provided; see behavior summary above:

* `internet_bandwidth` — (String) Internet bandwidth in the form `XMbps`. Used when access type resolves to **Bandwidth**.
* `minimum_commitment` — (String) Minimum commitment in the form `XMB`, `XGB`, or `XTB` (for example `500GB`). Used when access type resolves to **DataTransfer**.

### Optional

* `access_type` — (String) `Bandwidth` or `DataTransfer` (case-insensitive). When set, you must supply the matching bandwidth or commitment field (see behavioral summary above).
* `firewall_type` — (String) Firewall product type. Default `VFAAS`. Changing this forces a new resource.
* `is_internet_enabled` — (Boolean) Whether internet access is enabled. Default `true`.
* `non_distributed_enabled` — (Boolean) Whether non-distributed mode is enabled. Default `true`.
* `firewall_pricing_model` — (String) One of `daily`, `monthly`, `reserved_1`, `reserved_3`, `reserved_5` (case-insensitive). Default `daily`. Sent to the API only when set in configuration.
* `internet_pricing_model` — (String) Same values as `firewall_pricing_model`. Default `daily`. Sent to the API only when set in configuration.
* `hypervisor` — (String) `KVM` or `VCD_ESXI`. Default `KVM`.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` — (String) Firewall **resource** identifier returned from the audit **`resourceId`** after create (used for updates, read, and delete). This is separate from `audit_id`.
* `audit_id` — (String) Audit identifier for the latest completed operation.
* `status` — (String) Audit status (for example `SUCCESS`, `FAILED`, `IN_PROGRESS`).

## Import

Import is supported using the firewall `id`:

```shell
terraform import vayucloud_network_firewall.example <firewall_id>
```

OpenTofu: use the same syntax with `tofu import`.

```shell
terraform import vayucloud_network_firewall.example {{firewall_id}}
```
