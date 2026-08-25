---
page_title: "vayucloud_s3_domain Resource"
subcategory: ""
description: |-
  Manages an S3 (VCS) domain on a firewall. Create and delete are asynchronous (~30 minutes typical); quota may be updated in place.
---

# `vayucloud_s3_domain`

Creates a private **S3 domain** (object storage) mapped to a firewall. The provider resolves `engagement_id` and `endpoint_id` from `firewall_id` at plan time.

Domain creation is **long-running** (often ~30 minutes; longer when firewall mapping is required). Child resources (`s3_bucket`, `s3_user`, etc.) should `depends_on` the domain until `status` is complete.

Only **`quota`** may be updated in place. Other arguments force replacement.

## Example Usage

```hcl
resource "vayucloud_s3_domain" "example" {
  domain_name   = "my-vcs-domain"
  quota         = 1
  storage_class = "STANDARD"
  variant       = "value"
  firewall_id   = tonumber(vayucloud_network_firewall.bandwidth.id)
  pricing_model = "daily"
}
```

### Geo-resilient variant

```hcl
resource "vayucloud_s3_domain" "geo" {
  domain_name            = "geo-domain"
  quota                  = 10
  storage_class          = "STANDARD"
  variant                = "geoResilient"
  firewall_id            = tonumber(vayucloud_network_firewall.bandwidth.id)
  secondary_endpoint_id  = var.secondary_endpoint_id
}
```

## Argument Reference

### Required

* `domain_name` — (String) Short domain name (platform limits apply). Changing forces replacement.
* `quota` — (Number) Storage quota. **May be updated in place.**
* `storage_class` — (String) For example `STANDARD`, `AI_STANDARD`. Changing forces replacement.
* `variant` — (String) For example `value`, `geoResilient`. Changing forces replacement.
* `firewall_id` — (Number) IPC firewall with S3 mapping. Changing forces replacement.

### Optional

* `pricing_model` — (String) `daily` (default), `monthly`, `reserved_1`, `reserved_3`, `reserved_5`. Changing forces replacement.
* `secondary_endpoint_id` — (Number) Required when `variant` is `geoResilient`. Changing forces replacement.

## Attributes Reference

* `id` — (String) Domain identifier.
* `engagement_id` — (Number) Resolved from firewall.
* `endpoint_id` — (Number) Resolved from firewall.
* `domain_name_fqdn` — (String) Fully qualified domain name.
* `domain_access_ip` — (String) Private access IP on the firewall (for public IP association).
* `domain_access_public_ip` — (String) Public IP when assigned by the platform.
* `quota_unit` — (String) Quota unit (for example `GB`).
* `audit_id` — (String) Audit ID from async operations.
* `status` — (String) Audit completion status.

## Import

```shell
terraform import vayucloud_s3_domain.example <domain_id>
```

Set `firewall_id` (and other required arguments) in configuration after import.

## See also

* [`vayucloud_s3_bucket`](s3_bucket.md), [`vayucloud_s3_user`](s3_user.md), [`vayucloud_s3_token`](s3_token.md)
* [`vayucloud_network_public_ip`](network_public_ip.md) — NAT to `domain_access_ip`
* Example: `examples/IaaS/s3-complete/`, `examples/IaaS/complete/fw_vm_nas_complete/`
