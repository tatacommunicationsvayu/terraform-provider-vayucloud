---
page_title: "vayucloud_s3_domain Data Source"
subcategory: ""
description: |-
  Reads a single S3 (VCS) domain by domain ID, engagement ID, and firewall ID.
---

# `vayucloud_s3_domain`

Returns details for one S3 domain. To list domains, use [`vayucloud_s3_domain_list`](s3_domain_list.md).

## Example Usage

```hcl
data "vayucloud_s3_domain" "example" {
  s3_domain_id  = vayucloud_s3_domain.managed.id
  engagement_id = vayucloud_s3_domain.managed.engagement_id
  firewall_id   = vayucloud_s3_domain.managed.firewall_id
}
```

## Argument Reference

### Required

* `s3_domain_id` — (String) Domain identifier.
* `engagement_id` — (Number) Engagement identifier.
* `firewall_id` — (Number) Firewall identifier.

## Attributes Reference

* `id` — (String) Computed placeholder.
* `domain_name_fqdn` — (String) FQDN.
* `quota`, `quota_unit` — Domain quota.
* `storage_class`, `variant` — Domain settings.
* `endpoint_id` — (Number) Endpoint identifier.
* `domain_access_ip`, `domain_access_public_ip` — Access IPs.
* `status`, `message`, `response_code` — API metadata.
