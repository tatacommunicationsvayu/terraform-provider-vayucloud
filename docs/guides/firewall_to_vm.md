---
page_title: "Firewall to VM Guide"
subcategory: "Guides"
description: |-
  End-to-end guide for VayuCloud network perimeter to workloads: firewall, resource groups, zones, VMs, firewall rules, C2S VPN, and security groups.
---

# Firewall to VM Guide

This guide walks through the **network-to-workload** path in the VayuCloud provider: stand up a firewall and organizational hierarchy, place VMs in a zone, open traffic with **firewall rules**, allow remote access with **C2S VPN**, and enforce host-level access with **security groups**.

For per-attribute documentation, open the linked resource and data source pages. For the full platform dependency chain (including load balancer, NAS, and S3), see [Resources and data sources overview](../resourcesAndDatasource.md).

## Conventions

| Kind | Behavior |
|------|----------|
| **Resources** | Create, update, and destroy network and compute infrastructure. Long-running operations poll audit logs until completion. |
| **Data sources** | Read-only discovery (engagements, endpoints, firewalls, zones, VM catalogs, existing rules). |
| **Firewall rules vs security groups** | **Firewall rules** control north-south and zone-level traffic on the perimeter appliance. **Security groups** control ingress/egress on individual VMs (OpenStack-style). Both may be required for end-to-end connectivity. |
| **Provider login** | `VAYU_USERNAME` / `VAYU_PASSWORD` environment variables, or `username` / `password` in the provider block |

### Required before VM create

| ID | Purpose |
|----|---------|
| `engagement_id` | Account / project context |
| `endpoint_id` | Location for the firewall and NAS/S3 resources |
| `firewall_id` | Perimeter firewall |
| `environment_id` | Parent environment for the VM zone |
| `zone_id` | Network segment where the VM receives its IP |
| `image_id`, `flavor_id` | OS image and size from zone catalogs |

Resolve engagement and endpoint with data sources when they are not literals. Resolve `image_id` and `flavor_id` with [`vayucloud_virtualmachine_image`](../data-sources/virtualmachine_image.md) and [`vayucloud_virtualmachine_flavor`](../data-sources/virtualmachine_flavor.md).

### What you configure

| Item | How you provide it |
|------|-------------------|
| Perimeter | `vayucloud_network_firewall` or an existing `firewall_id` |
| VM zone | `vayucloud_network_zone` or existing `zone_id` |
| VM | `vayucloud_virtualmachine` with catalog IDs and zone |
| North-south access | `vayucloud_network_firewall_rule` (internet ↔ zone, zone ↔ NAS/VCS/LB) |
| Remote users | `vayucloud_network_c2s_vpn` on the same `firewall_id` |
| Host firewall | `vayucloud_security_group` + `vayucloud_virtualmachine_security_group_association` |

## Resources in this module

| Resource | Description | Async | In-place updates | Documentation |
|----------|-------------|:-----:|------------------|----------------|
| `vayucloud_network_firewall` | Perimeter firewall; throughput and internet bandwidth | Yes | Display name, throughput, bandwidth | [Network firewall](../resources/network_firewall.md) |
| `vayucloud_resource_group_business_unit` | Business unit under firewall | Yes | Name | [Resource group business unit](../resources/resource_group_business_unit.md) |
| `vayucloud_resource_group_environment` | Environment in business unit | Yes | Name | [Resource group environment](../resources/resource_group_environment.md) |
| `vayucloud_network_zone` | VM network zone (Auto IPAM or CIDR) | Yes | Name | [Network zone](../resources/network_zone.md) |
| `vayucloud_virtualmachine` | Virtual machine | Yes | Flavor resize, disk grow | [Virtual machine](../resources/virtualmachine.md) |
| `vayucloud_network_public_ip` | Public IP on VM private IP | Yes | `retain_on_dissociate` only | [Network public IP](../resources/network_public_ip.md) |
| `vayucloud_network_firewall_rule` | Allow/deny on firewall (internet, zone, NAS, VCS, LB) | Yes | Name, action, addresses, services, schedules | [Network firewall rule](../resources/network_firewall_rule.md) |
| `vayucloud_network_c2s_vpn` | Client-to-site VPN on firewall | Yes | User list reconciliation | [Network C2S VPN](../resources/network_c2s_vpn.md) |
| `vayucloud_network_c2s_vpn_user` | VPN users only (VPN already exists) | Yes | User add/remove/password | [Network C2S VPN user](../resources/network_c2s_vpn_user.md) |
| `vayucloud_security_group` | Security group and nested rules | Yes | Rule add/remove | [Security group](../resources/security_group.md) |
| `vayucloud_virtualmachine_security_group_association` | Attach SGs to VM | Yes | Membership list | [VM security group association](../resources/virtualmachine_security_group_association.md) |

**Replace-only fields (common pitfalls):**

| Resource | Changing these forces replacement |
|----------|-----------------------------------|
| `vayucloud_network_firewall_rule` | `firewall_id`, `source`, `source_zone_id`, `destination`, `destination_zone_id` |
| `vayucloud_network_c2s_vpn` | `firewall_id`, `pricing_model` |
| `vayucloud_security_group` | `firewall_id`, `id`, `name`, `description`, `type`, `resource_id` |
| `vayucloud_virtualmachine_security_group_association` | `instance_id` |

## Data sources in this module

| Data source | Use for | Documentation |
|-------------|---------|----------------|
| `vayucloud_account_engagement` | Resolve `engagement_id` | [Account engagement](../data-sources/account_engagement.md) |
| `vayucloud_account_location` | Resolve `endpoint_id` | [Account location](../data-sources/account_location.md) |
| `vayucloud_network_firewall` / `vayucloud_network_firewall_list` | Existing firewalls | [Network firewall](../data-sources/network_firewall.md) |
| `vayucloud_network_zone` / `vayucloud_network_zone_list` | Zones in an environment | [Network zone](../data-sources/network_zone.md) |
| `vayucloud_virtualmachine_image` | OS catalog for a zone | [Virtual machine image](../data-sources/virtualmachine_image.md) |
| `vayucloud_virtualmachine_flavor` | Size catalog for a zone | [Virtual machine flavor](../data-sources/virtualmachine_flavor.md) |
| `vayucloud_network_firewall_rule_list` | Discover existing rules | [Network firewall rule list](../data-sources/network_firewall_rule_list.md) |
| `vayucloud_network_c2s_vpn` / `vayucloud_network_c2s_vpn_users` | VPN state and usernames | [Network C2S VPN](../data-sources/network_c2s_vpn.md) |
| `vayucloud_list_security_group` | Security groups on firewall | [List security group](../data-sources/list_security_group.md) |
| `vayucloud_virtualmachine_list_security_group_rules` | Effective SG rules on a VM | [VM list security group rules](../data-sources/virtualmachine_list_security_group_rules.md) |

## Dependency order

Build from the perimeter inward, then add access controls.

| Step | What | Terraform |
|------|------|-----------|
| 1 | Account context | `vayucloud_account_engagement`, `vayucloud_account_location` *(optional)* |
| 2 | **Network firewall** | `vayucloud_network_firewall` |
| 3 | **Business unit** | `vayucloud_resource_group_business_unit` |
| 4 | **Environment** | `vayucloud_resource_group_environment` |
| 5 | **Network zone** | `vayucloud_network_zone` |
| 6 | Discover catalogs | `vayucloud_virtualmachine_image`, `vayucloud_virtualmachine_flavor` |
| 7 | **Create VM** | `vayucloud_virtualmachine` |
| 8 | **Public IP** *(optional)* | `vayucloud_network_public_ip` on VM private IP |
| 9 | **Security group + rules** | `vayucloud_security_group` |
| 10 | **Attach SG to VM** | `vayucloud_virtualmachine_security_group_association` |
| 11 | **Firewall rules** | `vayucloud_network_firewall_rule` — internet ↔ zone, zone ↔ zone, etc. |
| 12 | **C2S VPN** *(optional)* | `vayucloud_network_c2s_vpn` on same `firewall_id` |

```text
engagement / endpoint
        │
        ▼
   network_firewall
        │
   ┌────┴────┬──────────────┐
   ▼         ▼              ▼
business   c2s_vpn    firewall_rules
  unit         │              │
   │           │              │
   ▼           │              │
environment    │              │
   │           │              │
   ▼           │              │
network_zone ◄─┴──────────────┘
   │
   ▼
virtualmachine ──► security_group_association ◄── security_group
   │
   ▼ (optional)
network_public_ip
```

## Complete example

Repository examples (variable pattern as other IaaS modules):

| Path | Purpose |
|------|---------|
| `examples/IaaS/resources/vayucloud_network_firewall/` | Create firewall |
| `examples/IaaS/resources/vayucloud_network_zone/` | Create zone |
| `examples/IaaS/resources/vayucloud_virtualmachine/` | Create VM |
| `examples/IaaS/resources/vayucloud_network_firewall_rule/` | Firewall rule |
| `examples/IaaS/resources/vayucloud_network_c2s_vpn/` | C2S VPN |
| `examples/IaaS/resources/vayucloud_security_group/` | Security group |
| `examples/IaaS/resources/vayucloud_virtualmachine_security_group_association/` | Attach SG to VM |

The configuration below combines firewall → zone → VM → security groups → firewall rules → VPN:

```hcl
data "vayucloud_virtualmachine_image" "app" {
  zone_id = vayucloud_network_zone.app.id
  filter {
    name   = "name"
    values = ["Ubuntu-22.04"]
  }
}

data "vayucloud_virtualmachine_flavor" "app" {
  zone_id = vayucloud_network_zone.app.id
  filter {
    name   = "name"
    values = ["m1.medium"]
  }
}

resource "vayucloud_network_zone" "app" {
  name           = "app-zone"
  environment_id = vayucloud_resource_group_environment.prod.id
  firewall_id    = vayucloud_network_firewall.perimeter.id
  no_of_ips      = 32
}

resource "vayucloud_virtualmachine" "app" {
  name                     = "app-vm-1"
  vm_purpose               = "Application Server"
  image_id                 = data.vayucloud_virtualmachine_image.app.images[0].id
  flavor_id                = data.vayucloud_virtualmachine_flavor.app.flavors[0].id
  zone_id                  = vayucloud_network_zone.app.id
  iops                     = 3000
  is_kdump_or_page_enabled = "No"
  usage_type               = "ppu"
  pricing_model            = "hourly"
  root_disk_size           = 50
}

resource "vayucloud_security_group" "web" {
  firewall_id = vayucloud_network_firewall.perimeter.id
  name        = "tf-web-sg"
  description = "HTTP/HTTPS from VPN and office"

  rule {
    protocol         = "tcp"
    ether_type       = "ipv4"
    direction        = "ingress"
    port_range       = 80
    remote_ip_prefix = ["10.8.0.0/24"] # VPN pool — adjust to your C2S range
  }

  rule {
    protocol         = "tcp"
    ether_type       = "ipv4"
    direction        = "ingress"
    port_range       = 443
    remote_ip_prefix = ["10.8.0.0/24"]
  }
}

resource "vayucloud_virtualmachine_security_group_association" "app" {
  instance_id        = vayucloud_virtualmachine.app.instance_id
  security_group_ids = [vayucloud_security_group.web.id]
}

resource "vayucloud_network_public_ip" "app" {
  resource_type           = "zone"
  resource_id             = vayucloud_network_zone.app.id
  private_ip              = vayucloud_virtualmachine.app.ip
  public_ip_pricing_model = "daily"
  retain_on_dissociate    = true
}

resource "vayucloud_network_firewall_rule" "internet_to_app" {
  rule_name   = "ill_to_app"
  firewall_id = tonumber(vayucloud_network_firewall.perimeter.id)
  source      = "internet"
  destination = "zone"
  action      = "allow"
  services    = ["HTTP", "HTTPS"]

  source_addresses      = ["0.0.0.0/0"]
  destination_addresses = ["${vayucloud_network_public_ip.app.public_ip}/32"]
}

resource "vayucloud_network_c2s_vpn" "remote" {
  firewall_id   = vayucloud_network_firewall.perimeter.id
  zone_id       = vayucloud_network_zone.app.id
  pricing_model = "daily"

  users = [
    {
      name     = "ops_user"
      password = var.vpn_user_initial_password
    },
  ]
}
```

Set `vpn_user_initial_password` via `terraform.tfvars` or `TF_VAR_vpn_user_initial_password`. Password rules: length 8–50; lowercase, uppercase, digit, and allowed special character required.

## Operational notes

### Two layers of access control

| Layer | Resource | Typical use |
|-------|----------|-------------|
| Perimeter | `vayucloud_network_firewall_rule` | Internet → public IP, zone ↔ NAS, zone ↔ VCS |
| VM host | `vayucloud_security_group` | SSH/HTTP from VPN CIDR or another SG |

A rule on the **firewall** allowing HTTP to a VM public IP does not open port 80 on the VM unless a **security group** (or platform default) also permits it. Plan both when exposing services.

### Firewall rule zone IDs

When `source` or `destination` is `zone` or `nas`, set `source_zone_id` or `destination_zone_id` unless the rule targets a **public IP `/32`** on a zone destination. Omit optional zone IDs when not applicable — the provider does not backfill them from the API.

### Internet to VM pattern

1. Associate `vayucloud_network_public_ip` with the VM private IP (`resource_type = "zone"`, `resource_id = zone_id`).
2. Create an `internet` → `zone` rule with `destination_addresses = ["<public_ip>/32"]`.
3. Open the same ports on the VM security group (from `0.0.0.0/0` or a narrower CIDR).

### C2S VPN

* One VPN per firewall. Use `vayucloud_network_c2s_vpn` for initial create (includes first user), or `vayucloud_network_c2s_vpn_user` when the VPN already exists.
* Set **`zone_id`** on `vayucloud_network_c2s_vpn` — it is validated at create but not returned by the read API; keep it in configuration after import.
* Outputs `vpn_ip`, `pre_shared_key`, and `peer_id` configure VPN clients. User passwords are sensitive in state.

### Security group adoption modes

Use exactly one mode on `vayucloud_security_group`:

1. **Create** — set `name` (+ optional `description`).
2. **Adopt by UUID** — set `id`.
3. **Adopt system group** — set `type` (`zone`, `virtualmachine`, `firewall`) and `resource_id`.

When adopting, `managed_security_group = false` and destroy removes only Terraform-managed rules, not the group itself.

### Import existing infrastructure

| Resource | Import ID | After import |
|----------|-----------|--------------|
| `vayucloud_network_firewall_rule` | `<firewall_id>,<rule_id>` | Align zone IDs, addresses, services |
| `vayucloud_network_c2s_vpn` | `<firewall_id>` | Set `pricing_model`, `users`, and `zone_id` |
| `vayucloud_security_group` | `<firewall_id>/<sg_uuid>` | Confirm adoption mode and rules |
| `vayucloud_virtualmachine` | `<instance_id>` | Set zone, image, flavor, disks |

## Asynchronous operations

| Resource | Polling scope |
|----------|----------------|
| `vayucloud_network_firewall` | Create, update, delete |
| `vayucloud_resource_group_business_unit` | Create, update, delete |
| `vayucloud_resource_group_environment` | Create, update, delete |
| `vayucloud_network_zone` | Create, update, delete |
| `vayucloud_virtualmachine` | Create, delete, flavor/disk updates |
| `vayucloud_network_public_ip` | Create, destroy |
| `vayucloud_network_firewall_rule` | Create, update, delete |
| `vayucloud_network_c2s_vpn` | Create, update, delete |
| `vayucloud_network_c2s_vpn_user` | User mutations |
| `vayucloud_security_group` | Create, rule changes, delete |
| `vayucloud_virtualmachine_security_group_association` | Attach, detach, membership update |

No separate wait logic is required in configuration; the provider polls audit logs until each operation completes.

## See also

- [VayuCloud provider](../index.md) — Authentication and installation.
- [Resources and data sources overview](../resourcesAndDatasource.md) — Full platform dependency order.
- [NAS operations guide](nas_operations.md) — File server, volume, export policy, zone↔NAS rules.
- [S3 operations guide](s3_operations.md) — VCS domain, buckets, users, tokens, public access.
- [Load balancer guide](load_balancer.md) — HAProxy on the same firewall.
- [Network firewall rule](../resources/network_firewall_rule.md) — Rule types and validation.
- [Security group](../resources/security_group.md) — Rule blocks and adoption modes.
