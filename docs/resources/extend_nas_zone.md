---
page_title: "vayucloud_extend_nas_zone Resource"
subcategory: ""
description: |-
  Extends a NAS VLAN network zone for a file server (extendNASZone). Destroy calls deconfigureNASZone only; it does not delete the zone.
---

# `vayucloud_extend_nas_zone`

Invokes the platform **extend NAS zone** API to wire a NAS VLAN zone to a vServer. Operations are **synchronous** (no audit polling); responses are stored in `raw_response`.

**Destroy** calls `deconfigureNASZone` — it does **not** delete the underlying `vayucloud_network_zone` resource.

## Example Usage

```hcl
resource "vayucloud_extend_nas_zone" "nas" {
  zone_id                = tonumber(vayucloud_network_zone.nas_vlan[0].id)
  vserver_id             = tonumber(vayucloud_file_server.nas.id)
  is_firewall_configured = false
}
```

## Argument Reference

### Required

* `zone_id` — (Number) NAS VLAN zone ID. Changing forces replacement.
* `vserver_id` — (Number) File server (vServer) ID. Changing forces replacement.

### Optional

* `is_firewall_configured` — (Boolean) Default `false`. May be updated in place (re-invokes extend API).
* `ticket_id` — (String) Optional ticket ID query parameter. May be updated in place.

## Attributes Reference

* `id` — (String) Synthetic ID `extend-nas-zone-{zone_id}-{vserver_id}`.
* `raw_response` — (String) Raw API response JSON.

## Import

```shell
terraform import vayucloud_extend_nas_zone.example <zone_id>,<vserver_id>
```

## See also

* [`vayucloud_network_zone`](network_zone.md)
* [`vayucloud_file_server`](file_server.md)
* [`vayucloud_nas_vlan_zone`](../data-sources/nas_vlan_zone.md)
