---
page_title: "NAS File Storage"
description: |-
  VayuCloud NAS file servers, volumes, export policies, VLAN discovery, and zone extension.
---

# NAS File Storage

The NAS workflow is file server, volume, export policy, and optional NAS zone extension. Use the firewall-to-NAS rules documented in the workflow guide to control access.

## Prerequisites and use cases

Confirm the engagement and endpoint. Use a file server for a NAS virtual server, volumes for capacity, and export policies to attach client IPs.

## Resources

- [File server resource](../resources/file_server.md)
- [File storage volume resource](../resources/file_storage_volume.md)
- [File storage export policy resource](../resources/file_storage_export_policy.md)
- [Extend NAS zone resource](../resources/extend_nas_zone.md)

## Data sources

- [File server](../data-sources/file_server.md)
- [File server list](../data-sources/file_server_list.md)
- [File storage](../data-sources/file_storage.md)
- [NAS VLAN zone](../data-sources/nas_vlan_zone.md)

## Behavior notes

File-server and volume operations may wait for audit completion. Volume size is grow-only. When you set `iops` on a NAS volume explicitly, use the same **1**, **3**, or **5** IOPS/GB tiers as compute block storage—not absolute IOPS values. Omit `iops` to let the NAS order set the platform default. Export-policy IDs are derived from the attachment and may be synthetic; do not treat them as platform resource IDs.
