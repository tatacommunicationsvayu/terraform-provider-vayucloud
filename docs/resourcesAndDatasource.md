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
| `vayucloud_network_firewall_rule` | Firewall rule (allow/deny); source/destination types include internet, zone, NAS, VCS, load balancer; optional zone IDs and schedules; plan-time firewall and zone validation | Yes | [Network firewall rule](resources/network_firewall_rule.md) |
| `vayucloud_network_lb` | Load balancer on a firewall; enable, modify bandwidth, disable; requires `firewall_id` and `zone_id`; `firewall_id` replace-only; bandwidth in Mbps | Yes | [Network load balancer](resources/network_lb.md) |
| `vayucloud_network_lb_virtualservice` | HAProxy virtual service; listener, pool members, monitors; `load_balancer_id` and `name` replace-only; in-place updates for pool and listener settings | Yes | [Network LB virtual service](resources/network_lb_virtualservice.md) |
| `vayucloud_network_lb_ssl_profile` | SSL client certificate upload for HTTPS; create and delete only (no in-place update) | Yes | [Network LB SSL profile](resources/network_lb_ssl_profile.md) |
| `vayucloud_network_c2s_vpn` | C2S VPN on a firewall; pricing model and users; validates `firewall_id`; async create and user updates | Yes | [Network C2S VPN](resources/network_c2s_vpn.md) |
| `vayucloud_network_c2s_vpn_user` | VPN users only for an existing C2S VPN; add, remove, reset passwords; validates `firewall_id` at plan | Yes | [Network C2S VPN user](resources/network_c2s_vpn_user.md) |
| `vayucloud_resource_group_business_unit` | Business unit under a firewall; validates `firewall_id`; in-place name updates | Yes | [Resource group business unit](resources/resource_group_business_unit.md) |
| `vayucloud_resource_group_environment` | Environment within a business unit; validates firewall and business unit; in-place name updates | Yes | [Resource group environment](resources/resource_group_environment.md) |
| `vayucloud_network_zone` | Network zone for VMs; Auto IPAM or CIDR, IPv4 or dual-stack, overlay or VLAN; validates firewall and environment; in-place name updates | Yes | [Network zone](resources/network_zone.md) |
| `vayucloud_network_public_ip` | Associate public IP with private IP on zone, VM, firewall, bare metal, or load balancer; validates zone/VM IDs at plan; `retain_on_dissociate` in-place; identity and pricing replace-only | Yes | [Network public IP](resources/network_public_ip.md) |
| `vayucloud_file_server` | NAS vserver; validates engagement/endpoint at plan; no in-place update API | Yes | [File server](resources/file_server.md) |
| `vayucloud_file_storage_volume` | NAS volume; grow-only resize; validates parent file server at plan | Yes | [File storage volume](resources/file_storage_volume.md) |
| `vayucloud_file_storage_export_policy` | Attach client IP to NAS volume export; synthetic `id` at plan time | Yes | [File storage export policy](resources/file_storage_export_policy.md) |
| `vayucloud_extend_nas_zone` | Extend NAS VLAN zone for vserver; destroy deconfigures only (sync API) | No | [Extend NAS zone](resources/extend_nas_zone.md) |
| `vayucloud_s3_domain` | S3 (VCS) domain on firewall; long async create; quota update in place | Yes | [S3 domain](resources/s3_domain.md) |
| `vayucloud_s3_bucket` | Bucket in domain; versioning in place; delete blocked when not empty | No | [S3 bucket](resources/s3_bucket.md) |
| `vayucloud_s3_object` | Object upload (file, inline, or base64); body change re-uploads | No | [S3 object](resources/s3_object.md) |
| `vayucloud_s3_user` | S3 user; blocked on AI_STANDARD domains; delete cascades tokens | No | [S3 user](resources/s3_user.md) |
| `vayucloud_s3_token` | S3 access token; secret only at create; AI_STANDARD restrictions | No | [S3 token](resources/s3_token.md) |
| `vayucloud_virtualmachine` | Virtual machine; disks, image, and flavor; pre-create validate at plan; in-place flavor resize and disk size updates | Yes | [Virtual machine](resources/virtualmachine.md) |
| `vayucloud_virtualmachine_blockstorage` | Attach or resize extra block volume on a VM; validates attach at plan for new volumes; `size` increase in place; import `instance_id,volume_id` | Yes | [Virtual machine block storage](resources/virtualmachine_blockstorage.md) |
| `vayucloud_virtualmachine_state` | Power actions on an existing VM (`power_off`, `power_on`, `suspend`, `hard_reboot`, `soft_reboot`, `resume`) | Yes | [Virtual machine state](resources/virtualmachine_state.md) |
| `vayucloud_security_group` | Security group on a firewall; create, adopt by UUID, or adopt by type/resource ID; nested `rule` blocks | Yes | [Security group](resources/security_group.md) |
| `vayucloud_virtualmachine_security_group_association` | Attach security groups to a VM; detach on destroy; in-place membership updates | Yes | [Virtual machine security group association](resources/virtualmachine_security_group_association.md) |

Where **Async** is `Yes`, the provider polls audit logs until the operation completes when the platform requires it.

## Data sources

| Data source | Description | Documentation |
|-------------|-------------|----------------|
| `vayucloud_account_engagement` | Engagements for the authenticated user; optional `filter` | [Account engagement](data-sources/account_engagement.md) |
| `vayucloud_account_location` | Endpoints (locations) for an engagement; optional `filter` | [Account location](data-sources/account_location.md) |
| `vayucloud_network_firewall` | Single firewall by resource ID | [Network firewall](data-sources/network_firewall.md) |
| `vayucloud_network_firewall_list` | Firewalls for an engagement and endpoint; optional `filter` | [Network firewall list](data-sources/network_firewall_list.md) |
| `vayucloud_network_firewall_rule` | Single firewall rule by firewall ID and rule ID (action-state read) | [Network firewall rule](data-sources/network_firewall_rule.md) |
| `vayucloud_network_firewall_rule_list` | Firewall rules on a firewall; optional `filter` | [Network firewall rule list](data-sources/network_firewall_rule_list.md) |
| `vayucloud_network_lb` | Single load balancer by CI Master ID | [Network load balancer](data-sources/network_lb.md) |
| `vayucloud_network_lb_list` | Load balancers for an engagement and endpoint | [Network load balancer list](data-sources/network_lb_list.md) |
| `vayucloud_network_lb_virtualservice_options` | Virtual service wizard options (zones, protocols, algorithms, monitors); optional `filter` on zones | [Network LB virtual service options](data-sources/network_lb_virtualservice_options.md) |
| `vayucloud_network_lb_ssl_profiles` | Uploaded SSL certificate profiles on a load balancer | [Network LB SSL profiles](data-sources/network_lb_ssl_profiles.md) |
| `vayucloud_network_c2s_vpn` | C2S VPN state for a firewall (action-state read) | [Network C2S VPN](data-sources/network_c2s_vpn.md) |
| `vayucloud_network_c2s_vpn_users` | C2S VPN usernames for a firewall | [Network C2S VPN users](data-sources/network_c2s_vpn_users.md) |
| `vayucloud_network_zone` | Single zone by resource ID | [Network zone](data-sources/network_zone.md) |
| `vayucloud_network_zone_list` | Zones in an environment; optional `filter` | [Network zone list](data-sources/network_zone_list.md) |
| `vayucloud_network_public_ips` | Public IP list for a firewall (`firewall-ci`) or engagement; exactly one filter | [Network public IPs](data-sources/network_public_ips.md) |
| `vayucloud_resource_group_business_unit` | Single business unit by resource ID | [Resource group business unit](data-sources/resource_group_business_unit.md) |
| `vayucloud_resource_group_business_unit_list` | Business units on a firewall; optional `filter` | [Resource group business unit list](data-sources/resource_group_business_unit_list.md) |
| `vayucloud_resource_group_environment` | Single environment by resource ID | [Resource group environment](data-sources/resource_group_environment.md) |
| `vayucloud_resource_group_environment_list` | Environments in a business unit; optional `filter` | [Resource group environment list](data-sources/resource_group_environment_list.md) |
| `vayucloud_virtualmachine` | Single VM by instance ID | [Virtual machine](data-sources/virtualmachine.md) |
| `vayucloud_virtualmachine_blockstorage` | One attached volume by instance ID and volume ID | [Virtual machine block storage](data-sources/virtualmachine_blockstorage.md) |
| `vayucloud_virtualmachine_list` | VMs in a zone; optional `filter` | [Virtual machine list](data-sources/virtualmachine_list.md) |
| `vayucloud_virtualmachine_image` | Image catalog for a zone; optional `filter` | [Virtual machine image](data-sources/virtualmachine_image.md) |
| `vayucloud_virtualmachine_flavor` | Flavor catalog for a zone; optional `filter` | [Virtual machine flavor](data-sources/virtualmachine_flavor.md) |
| `vayucloud_account_engagement_auditlog` | Audit log entries for an engagement | [Account engagement audit log](data-sources/account_engagement_auditlog.md) |
| `vayucloud_auditlog_details` | Single audit entry by audit ID | [Audit log details](data-sources/auditlog_details.md) |
| `vayucloud_file_server` | Single NAS vserver by `vserver_id` | [File server](data-sources/file_server.md) |
| `vayucloud_file_server_list` | NAS vservers for engagement and endpoint | [File server list](data-sources/file_server_list.md) |
| `vayucloud_file_storage` | Single NAS volume by engagement, file server, and name | [File storage](data-sources/file_storage.md) |
| `vayucloud_nas_vlan_zone` | NAS VLAN zone discovery for engagement/endpoint | [NAS VLAN zone](data-sources/nas_vlan_zone.md) |
| `vayucloud_s3_domain` | Single S3 domain | [S3 domain](data-sources/s3_domain.md) |
| `vayucloud_s3_domain_list` | S3 domains for engagement/endpoint | [S3 domain list](data-sources/s3_domain_list.md) |
| `vayucloud_s3_bucket` | Single S3 bucket | [S3 bucket](data-sources/s3_bucket.md) |
| `vayucloud_s3_bucket_list` | Buckets in a domain | [S3 bucket list](data-sources/s3_bucket_list.md) |
| `vayucloud_s3_object` | S3 object metadata (no body) | [S3 object](data-sources/s3_object.md) |
| `vayucloud_s3_object_list` | Objects in a bucket | [S3 object list](data-sources/s3_object_list.md) |
| `vayucloud_s3_user` | Single S3 user | [S3 user](data-sources/s3_user.md) |
| `vayucloud_s3_user_list` | Users in a domain | [S3 user list](data-sources/s3_user_list.md) |
| `vayucloud_s3_token` | S3 token metadata | [S3 token](data-sources/s3_token.md) |
| `vayucloud_s3_token_list` | Tokens in a domain | [S3 token list](data-sources/s3_token_list.md) |
| `vayucloud_list_security_group` | Security groups on a firewall; optional `filter` | [List security group](data-sources/list_security_group.md) |
| `vayucloud_list_security_group_rules` | Rules for a security group; optional `filter` | [List security group rules](data-sources/list_security_group_rules.md) |
| `vayucloud_virtualmachine_list_security_group_rules` | Security groups and rules on a VM, grouped by port; optional `filter` | [Virtual machine list security group rules](data-sources/virtualmachine_list_security_group_rules.md) |

## Dependency order

In VayuCloud, pieces stack **from the network inward**: each step needs what came before. You do not need to know automation tool syntax to follow the idea.

| Step | What you are creating (plain language) |
|------|----------------------------------------|
| 1 | **Network firewall** — perimeter and connectivity for your network |
| 2 | **Business unit** — organizational grouping under that firewall |
| 3 | **Environment** — a named stage (for example *Production*) inside the business unit |
| 4 | **Network zone** — the network segment where virtual machines are placed |
| 5 | **Virtual machine** — OS image, size, and disks in that zone |
| 6 | *(Optional)* **Load balancer** — enable HAProxy on the firewall; set **`firewall_id`**, **`zone_id`**, and bandwidth |
| 7 | *(Optional)* **SSL profile** — upload a client certificate when using HTTPS |
| 8 | *(Optional)* **Virtual service** — listener and backend pool; `zone_id` from load balancer; pool VMs via `vayucloud_virtualmachine_list` |
| 9 | *(Optional)* **Public IP** — associate to VS `vip_ip` on the load balancer |
| 10 | *(Optional)* **Extra block volume** — attach additional disks with `vayucloud_virtualmachine_blockstorage` |
| 11 | *(Optional)* **Power actions** — start, stop, or reboot an existing VM |
| 12 | *(Optional)* **Security groups** — `vayucloud_security_group` and `vayucloud_virtualmachine_security_group_association` |
| 13 | *(Optional)* **Firewall rules** — `vayucloud_network_firewall_rule` for internet, zone, NAS, VCS, or load balancer traffic |
| 14 | *(Optional)* **NAS storage** — file server → volume → export policy; optional NAS VLAN zone and extend NAS zone |
| 15 | *(Optional)* **S3 (VCS)** — domain on firewall → bucket, user, token, objects |
| 16 | *(Optional)* **VCS public access** — public IP on `domain_access_ip` plus `ill_to_vcs` firewall rule |

**Using Terraform or OpenTofu**, the usual resource order matches the table: `vayucloud_network_firewall` → `vayucloud_resource_group_business_unit` → `vayucloud_resource_group_environment` → `vayucloud_network_zone` → `vayucloud_virtualmachine`; then optionally load balancer, SSL, virtual service, and public IP; `vayucloud_security_group` and `vayucloud_virtualmachine_security_group_association`; `vayucloud_network_firewall_rule` for access; NAS (`vayucloud_file_server`, `vayucloud_file_storage_volume`, `vayucloud_file_storage_export_policy`); VCS (`vayucloud_s3_domain` and children). See `examples/IaaS/complete/fw_vm_nas_complete/` for a combined example.

**Already provisioned in the cloud?** You can **look up** existing firewalls, zones, or VMs with **data sources** instead of creating new **resources**—your configuration only needs to respect the same logical order when something depends on something else.

For a focused HAProxy walkthrough (SSL, virtual services, pool members, VIP pinning, public IP), see the [Load balancer guide](guides/load_balancer.md). Example configurations live under `examples/IaaS/resources/`, `examples/IaaS/complete/lb_and_vs/`, and `examples/IaaS/complete/fw_vm_nas_complete/` (firewall + VM + NAS + VCS).

See also the friendly walkthrough on the [VayuCloud provider](index.md) page.

## Asynchronous operations

These resources wait on audit completion when the API returns an audit identifier:

| Resource | Polling scope |
|----------|----------------|
| `vayucloud_network_firewall` | Create, update, delete |
| `vayucloud_network_firewall_rule` | Create, update, delete |
| `vayucloud_network_lb` | Create, update (bandwidth), delete |
| `vayucloud_network_lb_virtualservice` | Create, update, delete |
| `vayucloud_network_lb_ssl_profile` | Create, delete |
| `vayucloud_network_c2s_vpn` | Create, update, delete |
| `vayucloud_network_c2s_vpn_user` | Create, update, delete (user mutations) |
| `vayucloud_network_zone` | Create, update, delete |
| `vayucloud_network_public_ip` | Create, destroy (dissociate); in-place state update for `retain_on_dissociate` only |
| `vayucloud_resource_group_business_unit` | Create, update, delete |
| `vayucloud_resource_group_environment` | Create, update, delete |
| `vayucloud_virtualmachine` | Create, delete, update (flavor resize, root and additional disk resize) |
| `vayucloud_virtualmachine_blockstorage` | Create (attach), delete (detach), update (size increase) |
| `vayucloud_virtualmachine_state` | When a power `action` is applied |
| `vayucloud_security_group` | Create, rule add/remove, delete |
| `vayucloud_virtualmachine_security_group_association` | Attach, detach, membership update |
| `vayucloud_file_server` | Create, delete |
| `vayucloud_file_storage_volume` | Create, resize (grow), delete |
| `vayucloud_file_storage_export_policy` | Attach, detach |
| `vayucloud_s3_domain` | Create, update (quota), delete |

Synchronous resources (no audit polling): `vayucloud_extend_nas_zone`, `vayucloud_s3_bucket`, `vayucloud_s3_object`, `vayucloud_s3_user`, `vayucloud_s3_token`.

The provider performs polling and status checks; no separate wait logic is required in configuration.

## See also

- [VayuCloud provider](index.md) — Authentication, installation, and provider arguments.
- [Load balancer guide](guides/load_balancer.md) — HAProxy module end-to-end.
