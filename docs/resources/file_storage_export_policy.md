---
page_title: "vayucloud_file_storage_export_policy Resource"
subcategory: ""
description: |-
  Attaches a client IP to a NAS volume export (NFS/CIFS attachClient). Create and delete are asynchronous.
---

# `vayucloud_file_storage_export_policy`

Associates a **client IP** with a NAS volume export (`attachClient` / `detachClient`). Without optional drift fields, refresh keeps Terraform state as-is. Set `engagement_id` and `file_server_id` to reconcile `client_ip` against `fetchVolumeDetails` on refresh.

`id` is computed as `{file_storage_volume_id}/{client_ip}`.

## Example Usage

```hcl
resource "vayucloud_file_storage_export_policy" "nas" {
  file_storage_volume_id = vayucloud_file_storage_volume.nas.id
  name                   = vayucloud_file_storage_volume.nas.name
  client_ip              = data.vayucloud_virtualmachine.app.ip

  engagement_id  = vayucloud_file_storage_volume.nas.engagement_id
  file_server_id = vayucloud_file_storage_volume.nas.file_server_id
}
```

## Argument Reference

### Required

* `file_storage_volume_id` — (String) NAS volume ID. Changing forces replacement.
* `name` — (String) Volume name (must match the volume). Changing forces replacement.
* `client_ip` — (String) Client IPv4 allowed to mount the export. Changing forces replacement.

### Optional

* `engagement_id` — (Number) With `file_server_id`, enables `fetchVolumeDetails` on refresh; if `client_ip` is no longer attached on the platform, the resource is removed from state so the next plan proposes recreate.
* `file_server_id` — (String) NAS vserver id paired with `engagement_id` for drift detection.
* `quota_gb` — (Number) Documentation-only; not sent to the API. Updates are blocked.

## Attributes Reference

* `id` — (String) `{file_storage_volume_id}/{client_ip}`.
* `audit_id` — (String) Audit ID from attach/detach.
* `status` — (String) Audit completion status.

## Drift detection

`attachClient` / `detachClient` have no dedicated GET API. By default, refresh keeps Terraform state as-is.

Set **`engagement_id`** and **`file_server_id`** on the resource (same values as the parent `vayucloud_file_storage_volume`). On refresh, if `client_ip` is no longer attached on the platform, the resource is removed from state and the next plan proposes **recreate** (re-attach). No extra data sources or `check` blocks are required.

## Import

```shell
terraform import vayucloud_file_storage_export_policy.example <volume_id>,<volume_name>,<client_ip>
```

## See also

* [`vayucloud_file_storage_volume`](file_storage_volume.md)
* [`vayucloud_network_firewall_rule`](network_firewall_rule.md) — Zone↔NAS NFS rules
