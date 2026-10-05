---
page_title: "NAS Operations Guide"
subcategory: "Guides"
description: |-
  End-to-end guide for VayuCloud NAS: file server (vserver), volumes, export policies, NAS VLAN zones, extend NAS zone, and zone↔NAS firewall rules.
---

# NAS Operations Guide

This guide walks through **NAS file storage** in the VayuCloud provider: create a file server (vserver), provision volumes, attach client IPs for NFS/CIFS export, wire the NAS VLAN zone to the vserver, and open **zone ↔ NAS** traffic with firewall rules.

For per-attribute documentation, open the linked resource and data source pages. For the perimeter-to-VM foundation, see [Firewall to VM guide](firewall_to_vm.md). For the full dependency chain, see [Resources and data sources overview](../resourcesAndDatasource.md).

## Conventions

| Kind | Behavior |
|------|----------|
| **Resources** | Create, grow, and destroy NAS infrastructure. Most operations poll audit logs until completion. |
| **Data sources** | Read-only discovery (existing vservers, NAS VLAN zones, volume details). |
| **`engagement_id` / `endpoint_id`** | Required on file server and volume resources; match the firewall or account context. |
| **Export policy drift** | Set `engagement_id` and `file_server_id` on export policies so refresh detects portal-side detach. |
| **Provider login** | `VAYU_USERNAME` / `VAYU_PASSWORD` environment variables, or `username` / `password` in the provider block |

### Required before NAS volume create

| ID | Purpose |
|----|---------|
| `engagement_id` | Account context for NAS APIs |
| `endpoint_id` | Location for the vserver |
| `file_server_id` | Parent vserver from `vayucloud_file_server` |

For VM mount access you also need the VM private IP (`vayucloud_virtualmachine.ip`) and matching **firewall rules** between the app zone and NAS VLAN zone.

### What you configure

| Item | How you provide it |
|------|-------------------|
| vServer | `vayucloud_file_server` — `vserver_name`, `file_storage_type` (`NAS-NFS` or `CIFS`) |
| Volume | `vayucloud_file_storage_volume` — size, IOPS, pricing |
| Client mount | `vayucloud_file_storage_export_policy` — `client_ip` per VM or subnet |
| NAS VLAN wiring | `vayucloud_nas_vlan_zone` data source + optional `vayucloud_extend_nas_zone` |
| Network path | `vayucloud_network_firewall_rule` with `source`/`destination` `zone` or `nas` |

## Resources in this module

| Resource | Description | Async | In-place updates | Documentation |
|----------|-------------|:-----:|------------------|----------------|
| `vayucloud_file_server` | NAS vserver | Yes | None (no update API) | [File server](../resources/file_server.md) |
| `vayucloud_file_storage_volume` | NAS volume on vserver | Yes | **Grow** `size_gb` only | [File storage volume](../resources/file_storage_volume.md) |
| `vayucloud_file_storage_export_policy` | Attach client IP to export | Yes | None | [File storage export policy](../resources/file_storage_export_policy.md) |
| `vayucloud_extend_nas_zone` | Wire NAS VLAN zone to vserver | No | `is_firewall_configured`, `ticket_id` | [Extend NAS zone](../resources/extend_nas_zone.md) |
| `vayucloud_network_firewall_rule` | Zone ↔ NAS NFS/CIFS path | Yes | Addresses, services, schedules | [Network firewall rule](../resources/network_firewall_rule.md) |

**Replace-only fields (common pitfalls):**

| Resource | Changing these forces replacement |
|----------|-----------------------------------|
| `vayucloud_file_server` | `engagement_id`, `endpoint_id`, `vserver_name`, `file_storage_type` |
| `vayucloud_file_storage_volume` | `file_server_id`, `name`, pricing/usage/type fields (not `size_gb` grow) |
| `vayucloud_file_storage_export_policy` | `file_storage_volume_id`, `name`, `client_ip` |
| `vayucloud_extend_nas_zone` | `zone_id`, `vserver_id` |

## Data sources in this module

| Data source | Use for | Documentation |
|-------------|---------|----------------|
| `vayucloud_file_server` / `vayucloud_file_server_list` | Existing vservers | [File server](../data-sources/file_server.md) |
| `vayucloud_file_storage` | Read one volume by name | [File storage](../data-sources/file_storage.md) |
| `vayucloud_nas_vlan_zone` | Discover existing NAS VLAN zone IDs | [NAS VLAN zone](../data-sources/nas_vlan_zone.md) |
| `vayucloud_virtualmachine` / `vayucloud_virtualmachine_list` | Client IPs for export policy | [Virtual machine list](../data-sources/virtualmachine_list.md) |
| `vayucloud_network_zone_list` | App zone CIDR for firewall rules | [Network zone list](../data-sources/network_zone_list.md) |

## Dependency order

NAS sits beside the VM stack: you need account context, a vserver, volumes, export clients, optional VLAN extension, and firewall rules.

| Step | What | Terraform |
|------|------|-----------|
| 1 | Account context | `engagement_id`, `endpoint_id` from firewall or data sources |
| 2 | Discover NAS VLAN zone | `vayucloud_nas_vlan_zone` — reuse `nas_vlan_zone_ids` when present |
| 3 | **Create file server** | `vayucloud_file_server` |
| 4 | **Extend NAS zone** *(when VLAN zone exists)* | `vayucloud_extend_nas_zone` — links zone to vserver |
| 5 | **Create volume** | `vayucloud_file_storage_volume` |
| 6 | **Attach client IP** | `vayucloud_file_storage_export_policy` — one block per client |
| 7 | **Firewall rules** | `zone` ↔ `nas` rules with zone IDs and CIDRs |
| 8 | Mount on VM | OS-level NFS/CIFS mount using export path from platform |

```text
engagement / endpoint
        │
        ▼
  nas_vlan_zone (data) ──► extend_nas_zone (optional)
        │
        ▼
   file_server (vserver)
        │
        ▼
 file_storage_volume
        │
        ▼
file_storage_export_policy ◄── VM ip (client_ip)
        │
        ▼
network_firewall_rule (zone ↔ nas, NFS)
        │
        ▼
   VM mount (OS)
```

## Complete example (NFS)

Repository examples:

| Path | Purpose |
|------|---------|
| `examples/IaaS/resources/vayucloud_file_server/` | Create vserver |
| `examples/IaaS/resources/vayucloud_file_storage_volume/` | Create volume |
| `examples/IaaS/resources/vayucloud_file_storage_export_policy/` | Attach client IP |
| `examples/IaaS/resources/vayucloud_extend_nas_zone/` | Extend NAS VLAN zone + data source |
| `examples/IaaS/resources/vayucloud_network_firewall_rule/` | Zone ↔ NAS rule |

```hcl
data "vayucloud_nas_vlan_zone" "stack" {
  engagement_id = var.engagement_id
  endpoint_id   = var.endpoint_id
}

locals {
  nas_vlan_zone_id = tonumber(tolist(data.vayucloud_nas_vlan_zone.stack.nas_vlan_zone_ids)[0])
  app_zone_cidr    = "10.10.0.0/24"   # from vayucloud_network_zone or portal
  nas_vlan_cidr    = "10.20.0.0/24"   # NAS VLAN CIDR for the engagement
}

resource "vayucloud_file_server" "nas" {
  engagement_id     = var.engagement_id
  endpoint_id       = var.endpoint_id
  vserver_name      = "nfsFileServer1"
  file_storage_type = "NAS-NFS"
}

resource "vayucloud_extend_nas_zone" "nas" {
  zone_id                = local.nas_vlan_zone_id
  vserver_id             = tonumber(vayucloud_file_server.nas.id)
  is_firewall_configured = false

  depends_on = [vayucloud_file_server.nas]
}

resource "vayucloud_file_storage_volume" "data" {
  engagement_id     = var.engagement_id
  endpoint_id       = var.endpoint_id
  file_server_id    = vayucloud_file_server.nas.id
  name              = "app-data-vol"
  size_gb           = 100
  file_storage_type = "NFS"
  usage_type        = "reserved"
  pricing_model     = "reserved_1"
  iops              = 3000
}

resource "vayucloud_file_storage_export_policy" "app" {
  file_storage_volume_id = vayucloud_file_storage_volume.data.id
  name                   = vayucloud_file_storage_volume.data.name
  client_ip              = vayucloud_virtualmachine.app.ip

  engagement_id  = vayucloud_file_storage_volume.data.engagement_id
  file_server_id = vayucloud_file_storage_volume.data.file_server_id
}

resource "vayucloud_network_firewall_rule" "zone_to_nas" {
  rule_name           = "app_zone_to_nas"
  firewall_id         = var.firewall_id
  source              = "zone"
  source_zone_id      = vayucloud_network_zone.app.id
  destination         = "nas"
  destination_zone_id = local.nas_vlan_zone_id
  action              = "allow"
  services            = ["NFS"]

  source_addresses      = [local.app_zone_cidr]
  destination_addresses = [local.nas_vlan_cidr]
}

resource "vayucloud_network_firewall_rule" "nas_to_zone" {
  rule_name           = "nas_to_app_zone"
  firewall_id         = var.firewall_id
  source              = "nas"
  source_zone_id      = local.nas_vlan_zone_id
  destination         = "zone"
  destination_zone_id = vayucloud_network_zone.app.id
  action              = "allow"
  services            = ["NFS"]

  source_addresses      = [local.nas_vlan_cidr]
  destination_addresses = [local.app_zone_cidr]
}
```

Bidirectional NFS typically needs **two** rules (zone→NAS and NAS→zone). Adjust `services` to `CIFS`-appropriate names when using `file_storage_type = "CIFS"`.

## Operational notes

### NAS VLAN zone discovery

Before creating a duplicate NAS VLAN `vayucloud_network_zone`, call `vayucloud_nas_vlan_zone`. When `is_nas_vlan_zone` is true, reuse IDs from `nas_vlan_zone_ids` and call `vayucloud_extend_nas_zone` to bind the vserver.

### Volume resize

Only **increase** `size_gb` on `vayucloud_file_storage_volume`. Shrinking fails at apply. Out-of-band growth in the portal is adopted on refresh.

### Export policy and drift

Without `engagement_id` and `file_server_id`, refresh cannot detect portal-side client detach. Set both (same as the parent volume) so a missing client triggers **recreate** on the next plan.

Synthetic `id` format: `{file_storage_volume_id}/{client_ip}`.

### Extend NAS zone destroy

Destroy on `vayucloud_extend_nas_zone` calls **deconfigureNASZone** only — it does **not** delete the underlying network zone or vserver.

### Firewall rule validation

When `source` or `destination` is `zone` or `nas`, provide `source_zone_id` or `destination_zone_id` and CIDRs that match the zones. Plan-time validation runs against the firewall and zone IDs.

### Enable timing

| Operation | Typical duration | Notes |
|-----------|------------------|-------|
| File server create | Minutes | Async audit polling |
| Volume create | Minutes | Async audit polling |
| Export attach/detach | Minutes | Async audit polling |
| Extend NAS zone | Seconds | Synchronous API |
| Volume grow | Minutes | Async audit polling |

### Import existing infrastructure

| Resource | Import ID | After import |
|----------|-----------|--------------|
| `vayucloud_file_server` | `<vserver_id>` or `<engagement_id>/<vserver_id>` | Set `engagement_id`, `endpoint_id` for destroy |
| `vayucloud_file_storage_volume` | `<engagement_id>,<file_server_id>,<volume_id>,<name>` | Confirm size and pricing fields |
| `vayucloud_file_storage_export_policy` | `<volume_id>,<volume_name>,<client_ip>` | Set drift-detection fields |
| `vayucloud_extend_nas_zone` | `<zone_id>,<vserver_id>` | Confirm `is_firewall_configured` |

## Asynchronous operations

| Resource | Polling scope |
|----------|----------------|
| `vayucloud_file_server` | Create, delete |
| `vayucloud_file_storage_volume` | Create, resize (grow), delete |
| `vayucloud_file_storage_export_policy` | Attach, detach |
| `vayucloud_network_firewall_rule` | Create, update, delete |

Synchronous (no audit polling): `vayucloud_extend_nas_zone`.

## See also

- [VayuCloud provider](../index.md) — Authentication and installation.
- [Resources and data sources overview](../resourcesAndDatasource.md) — Full platform dependency order.
- [Firewall to VM guide](firewall_to_vm.md) — Firewall, zone, and VM prerequisites.
- [File server](../resources/file_server.md) — vServer resource reference.
- [File storage volume](../resources/file_storage_volume.md) — Volume sizing and import.
- [File storage export policy](../resources/file_storage_export_policy.md) — Client attach and drift.
- [Extend NAS zone](../resources/extend_nas_zone.md) — VLAN zone wiring.
- [NAS VLAN zone](../data-sources/nas_vlan_zone.md) — Zone discovery.
