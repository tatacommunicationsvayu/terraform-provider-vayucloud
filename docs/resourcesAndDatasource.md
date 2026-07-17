---
page_title: "Resources and Data Sources"
subcategory: ""
description: |-
  Reference for VayuCloud provider resources and data sources: summaries, dependency order, asynchronous behavior, and links to full documentation.
---

# Resources and Data Sources

This document summarizes the managed resources and read-only data sources in the VayuCloud provider. For provider configuration (authentication, timeouts), see the [VayuCloud provider](index.md) page. For per-attribute documentation, open the linked resource or data source guide.

## Conventions

| Kind | Behavior |
|------|----------|
| **Resources** | Create, update, and destroy infrastructure. Participate in dependency ordering during apply. |
| **Data sources** | Read-only; evaluated at plan and refresh to supply values to the configuration. |
| **`*_list` data sources** | Return collections. Many support optional `filter` blocks. The corresponding name **without** `_list` reads a **single** object by identifier. |

## Resources

| Resource | Description | Async | Documentation |
|----------|-------------|:-----:|----------------|
| `vayucloud_network_firewall` | Network firewall; throughput, internet bandwidth, optional pricing models and hypervisor; validates engagement and endpoint; in-place updates for display name, throughput, and bandwidth | Yes | [Network firewall](resources/network_firewall.md) |
| `vayucloud_network_c2s_vpn` | C2S VPN on a firewall; pricing model and users; validates `firewall_id`; async create and user updates | Yes | [Network C2S VPN](resources/network_c2s_vpn.md) |
| `vayucloud_network_c2s_vpn_user` | VPN users only for an existing C2S VPN; add, remove, reset passwords; validates `firewall_id` at plan | Yes | [Network C2S VPN user](resources/network_c2s_vpn_user.md) |
| `vayucloud_resource_group_business_unit` | Business unit under a firewall; validates `firewall_id`; in-place name updates | Yes | [Resource group business unit](resources/resource_group_business_unit.md) |
| `vayucloud_resource_group_environment` | Environment within a business unit; validates firewall and business unit; in-place name updates | Yes | [Resource group environment](resources/resource_group_environment.md) |
| `vayucloud_network_zone` | Network zone for VMs; Auto IPAM or CIDR, IPv4 or dual-stack, overlay or VLAN; validates firewall and environment; in-place name updates | Yes | [Network zone](resources/network_zone.md) |
| `vayucloud_network_public_ip` | Associate public IP with private IP on zone, VM, firewall, bare metal, or load balancer; validates zone/VM IDs at plan; `retain_on_dissociate` in-place; identity and pricing replace-only | Yes | [Network public IP](resources/network_public_ip.md) |
| `vayucloud_keypair` | Key pair (generate new or upload existing); RSA; private key available when generated | No | [Keypair](resources/keypair.md) |
| `vayucloud_virtualmachine` | Virtual machine; disks, image, and flavor; pre-create validate at plan; in-place flavor resize and disk size updates | Yes | [Virtual machine](resources/virtualmachine.md) |
| `vayucloud_virtualmachine_blockstorage` | Attach or resize extra block volume on a VM; validates attach at plan for new volumes; `size` increase in place; import `instance_id,volume_id` | Yes | [Virtual machine block storage](resources/virtualmachine_blockstorage.md) |
| `vayucloud_virtualmachine_state` | Power actions on an existing VM (`power_off`, `power_on`, `suspend`, `hard_reboot`, `soft_reboot`, `resume`) | Yes | [Virtual machine state](resources/virtualmachine_state.md) |

Where **Async** is `Yes`, the provider polls audit logs until the operation completes when the platform requires it.

## Data sources

| Data source | Description | Documentation |
|-------------|-------------|----------------|
| `vayucloud_account_engagement` | Engagements for the authenticated user; optional `filter` | [Account engagement](data-sources/account_engagement.md) |
| `vayucloud_account_location` | Endpoints (locations) for an engagement; optional `filter` | [Account location](data-sources/account_location.md) |
| `vayucloud_network_firewall` | Single firewall by resource ID | [Network firewall](data-sources/network_firewall.md) |
| `vayucloud_network_firewall_list` | Firewalls for an engagement and endpoint; optional `filter` | [Network firewall list](data-sources/network_firewall_list.md) |
| `vayucloud_network_c2s_vpn` | C2S VPN state for a firewall (action-state read) | [Network C2S VPN](data-sources/network_c2s_vpn.md) |
| `vayucloud_network_c2s_vpn_users` | C2S VPN usernames for a firewall | [Network C2S VPN users](data-sources/network_c2s_vpn_users.md) |
| `vayucloud_network_zone` | Single zone by resource ID | [Network zone](data-sources/network_zone.md) |
| `vayucloud_network_zone_list` | Zones in an environment; optional `filter` | [Network zone list](data-sources/network_zone_list.md) |
| `vayucloud_network_public_ips` | Public IP list for a firewall (`firewall-ci`) or engagement; exactly one filter | [Network public IPs](data-sources/network_public_ips.md) |
| `vayucloud_resource_group_business_unit` | Single business unit by resource ID | [Resource group business unit](data-sources/resource_group_business_unit.md) |
| `vayucloud_resource_group_business_unit_list` | Business units on a firewall; optional `filter` | [Resource group business unit list](data-sources/resource_group_business_unit_list.md) |
| `vayucloud_resource_group_environment` | Single environment by resource ID | [Resource group environment](data-sources/resource_group_environment.md) |
| `vayucloud_resource_group_environment_list` | Environments in a business unit; optional `filter` | [Resource group environment list](data-sources/resource_group_environment_list.md) |
| `vayucloud_keypair` | Key pairs for an engagement | [Keypair](data-sources/keypair.md) |
| `vayucloud_virtualmachine` | Single VM by instance ID | [Virtual machine](data-sources/virtualmachine.md) |
| `vayucloud_virtualmachine_blockstorage` | One attached volume by instance ID and volume ID | [Virtual machine block storage](data-sources/virtualmachine_blockstorage.md) |
| `vayucloud_virtualmachine_list` | VMs in a zone; optional `filter` | [Virtual machine list](data-sources/virtualmachine_list.md) |
| `vayucloud_virtualmachine_image` | Image catalog for a zone; optional `filter` | [Virtual machine image](data-sources/virtualmachine_image.md) |
| `vayucloud_virtualmachine_flavor` | Flavor catalog for a zone; optional `filter` | [Virtual machine flavor](data-sources/virtualmachine_flavor.md) |
| `vayucloud_account_engagement_auditlog` | Audit log entries for an engagement | [Account engagement audit log](data-sources/account_engagement_auditlog.md) |
| `vayucloud_auditlog_details` | Single audit entry by audit ID | [Audit log details](data-sources/auditlog_details.md) |

## Dependency order

In VayuCloud, pieces stack **from the network inward**: each step needs what came before. You do not need to know automation tool syntax to follow the idea.

| Step | What you are creating (plain language) |
|------|----------------------------------------|
| 1 | **Network firewall** — perimeter and connectivity for your network |
| 2 | **Business unit** — organizational grouping under that firewall |
| 3 | **Environment** — a named stage (for example *Production*) inside the business unit |
| 4 | **Network zone** — the network segment where virtual machines are placed |
| 5 | **Virtual machine** — OS image, size, and disks in that zone |
| 6 | *(Optional)* **Extra block volume** — attach additional disks with `vayucloud_virtualmachine_blockstorage` |
| 7 | *(Optional)* **Power actions** — start, stop, or reboot an existing VM |
| 8 | *(Optional)* **SSH key pair** — create or upload keys when you need them for access (often before or alongside the VM, depending on your process) |

**Using Terraform or OpenTofu**, the usual resource order matches the table: `vayucloud_network_firewall` → `vayucloud_resource_group_business_unit` → `vayucloud_resource_group_environment` → `vayucloud_network_zone` → `vayucloud_virtualmachine`; then optionally `vayucloud_virtualmachine_blockstorage` for extra disks, `vayucloud_virtualmachine_state` for power, and `vayucloud_keypair` for keys. Exact argument names are in each resource’s documentation.

**Already provisioned in the cloud?** You can **look up** existing firewalls, zones, or VMs with **data sources** instead of creating new **resources**—your configuration only needs to respect the same logical order when something depends on something else.

See also the friendly walkthrough on the [VayuCloud provider](index.md) page.

## Asynchronous operations

These resources wait on audit completion when the API returns an audit identifier:

| Resource | Polling scope |
|----------|----------------|
| `vayucloud_network_firewall` | Create, update, delete |
| `vayucloud_network_c2s_vpn` | Create, update, delete |
| `vayucloud_network_c2s_vpn_user` | Create, update, delete (user mutations) |
| `vayucloud_network_zone` | Create, update, delete |
| `vayucloud_network_public_ip` | Create, destroy (dissociate); in-place state update for `retain_on_dissociate` only |
| `vayucloud_resource_group_business_unit` | Create, update, delete |
| `vayucloud_resource_group_environment` | Create, update, delete |
| `vayucloud_virtualmachine` | Create, delete, update (flavor resize, root and additional disk resize) |
| `vayucloud_virtualmachine_blockstorage` | Create (attach), delete (detach), update (size increase) |
| `vayucloud_virtualmachine_state` | When a power `action` is applied |

The provider performs polling and status checks; no separate wait logic is required in configuration.

## See also

- [VayuCloud provider](index.md) — Authentication, installation, and provider arguments.
