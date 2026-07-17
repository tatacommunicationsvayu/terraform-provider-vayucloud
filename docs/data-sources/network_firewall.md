---
page_title: "vayucloud_network_firewall Data Source"
subcategory: ""
description: |-
  Reads a single network firewall by resource identifier.
---

# `vayucloud_network_firewall`

Returns current firewall configuration for one appliance. To enumerate firewalls without a known ID, use [`vayucloud_network_firewall_list`](network_firewall_list.md).

## Example Usage

```hcl
data "vayucloud_network_firewall" "example" {
  network_firewall_id = {{network_firewall_id}}
}
```

## Argument Reference

### Required

* `network_firewall_id` — (String) Firewall resource identifier.

## Attributes Reference

* `id` — (String) Computed placeholder for Terraform.
* `firewall_display_name` — (String) Display name.
* `firewall_throughput` — (String) Throughput (for example `2Mbps`).
* `internet_bandwidth` — (String) Internet bandwidth.
* `access_type` — (String) `Bandwidth` or `DataTransfer`.
* `minimum_commitment` — (String) Minimum commitment when applicable (for example `500GB`).
* `firewall_pricing_model` — (String) Firewall pricing model.
* `internet_pricing_model` — (String) Internet pricing model.
* `engagement_id` — (Number) Engagement identifier.
* `endpoint_id` — (Number) Endpoint identifier.
* `hypervisor` — (String) `KVM` or `VCD_ESXI`.
* `status` — (String) API status (for example `success`).
* `message` — (String) API message text.
* `response_code` — (Number) API response code (`0` indicates success).
* `raw_response` — (String) Raw JSON response for troubleshooting.
