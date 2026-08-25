---
page_title: "vayucloud_nas_vlan_zone Data Source"
subcategory: ""
description: |-
  Discovers whether a NAS VLAN zone already exists for an engagement and endpoint.
---

# `vayucloud_nas_vlan_zone`

Used before creating a new NAS VLAN `vayucloud_network_zone` — when `is_nas_vlan_zone` is true, reuse `nas_vlan_zone_ids` instead of creating a duplicate zone.

```hcl
data "vayucloud_nas_vlan_zone" "stack" {
  engagement_id = var.engagement_id
  endpoint_id   = var.endpoint_id
}
```

## Argument Reference

### Required

* `engagement_id` — (Number) Engagement identifier.
* `endpoint_id` — (Number) Endpoint identifier.

## Attributes Reference

* `is_nas_vlan_zone` — (Boolean) Whether a NAS VLAN zone already exists.
* `nas_vlan_zone_ids` — (Set of String) Existing NAS VLAN zone IDs when present.
* `raw_response` — (String) Raw API JSON.

## See also

* [`vayucloud_extend_nas_zone`](../resources/extend_nas_zone.md)
* Example: `examples/IaaS/complete/fw_vm_nas_complete/`
