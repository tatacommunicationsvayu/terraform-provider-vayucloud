---
page_title: "VayuCloud Provider"
subcategory: ""
description: |-
  Use the VayuCloud provider to provision and manage VayuCloud infrastructure with Terraform or OpenTofu. Configure authentication, resources, and data sources from your IaC configuration.
---

# VayuCloud Provider

The VayuCloud provider is used to interact with the resources supported by the VayuCloud platform. Use it to manage network firewalls and firewall rules, load balancers and virtual services, resource groups (business units and environments), network zones, public IP associations, virtual machines, security groups, NAS file storage, S3 (VCS) object storage, and VM power operations.

The provider authenticates against the TATA Communications identity provider and uses a Bearer token for API requests. Long-running operations are handled by polling audit logs until completion, so applies can span several minutes for resources such as firewalls and virtual machines.

## Resources and data sources

Resources create, update, and destroy infrastructure. Data sources read existing data without changing it.

- **Single-object data sources** resolve one resource by identifier.
- **List data sources** (names ending in `_list`) return collections and often support optional `filter` blocks for client-side narrowing.

### Resources

| Resource | Description |
|----------|-------------|
| [`vayucloud_network_firewall`](resources/network_firewall.md) | Network firewall appliance (throughput and bandwidth) |
| [`vayucloud_network_firewall_rule`](resources/network_firewall_rule.md) | Firewall allow/deny rule (internet, zone, NAS, VCS, load balancer) |
| [`vayucloud_network_lb`](resources/network_lb.md) | Load balancer on a firewall (enable, bandwidth, disable); requires `zone_id` for VS/pool preflight |
| [`vayucloud_network_lb_virtualservice`](resources/network_lb_virtualservice.md) | HAProxy virtual service (listener, pool, monitors) |
| [`vayucloud_network_lb_ssl_profile`](resources/network_lb_ssl_profile.md) | SSL client certificate upload for HTTPS virtual services |
| [`vayucloud_network_c2s_vpn`](resources/network_c2s_vpn.md) | C2S VPN on a firewall (pricing model and users) |
| [`vayucloud_network_c2s_vpn_user`](resources/network_c2s_vpn_user.md) | C2S VPN users only (add, remove, password reset) |
| [`vayucloud_resource_group_business_unit`](resources/resource_group_business_unit.md) | Business unit under a firewall |
| [`vayucloud_resource_group_environment`](resources/resource_group_environment.md) | Environment within a business unit |
| [`vayucloud_network_zone`](resources/network_zone.md) | Network zone for virtual machine placement |
| [`vayucloud_network_public_ip`](resources/network_public_ip.md) | Associate a public IP with a zone, VM, firewall, bare metal, or load balancer |
| [`vayucloud_file_server`](resources/file_server.md) | NAS file server (vserver) |
| [`vayucloud_file_storage_volume`](resources/file_storage_volume.md) | NAS file storage volume |
| [`vayucloud_file_storage_export_policy`](resources/file_storage_export_policy.md) | NAS export client IP attachment |
| [`vayucloud_extend_nas_zone`](resources/extend_nas_zone.md) | Extend NAS VLAN zone for a vserver |
| [`vayucloud_s3_domain`](resources/s3_domain.md) | S3 (VCS) domain on a firewall |
| [`vayucloud_s3_bucket`](resources/s3_bucket.md) | S3 bucket in a domain |
| [`vayucloud_s3_object`](resources/s3_object.md) | S3 object upload |
| [`vayucloud_s3_user`](resources/s3_user.md) | S3 service user |
| [`vayucloud_s3_token`](resources/s3_token.md) | S3 access token |
| [`vayucloud_virtualmachine`](resources/virtualmachine.md) | Virtual machine |
| [`vayucloud_virtualmachine_blockstorage`](resources/virtualmachine_blockstorage.md) | Extra block volume attached to a virtual machine |
| [`vayucloud_virtualmachine_state`](resources/virtualmachine_state.md) | Power actions for an existing virtual machine |
| [`vayucloud_security_group`](resources/security_group.md) | Security group and rules on a firewall |
| [`vayucloud_virtualmachine_security_group_association`](resources/virtualmachine_security_group_association.md) | Attach security groups to a virtual machine |

### Data sources

| Data source | Description |
|-------------|-------------|
| [`vayucloud_account_engagement`](data-sources/account_engagement.md) | Engagements (by name or filter) |
| [`vayucloud_account_location`](data-sources/account_location.md) | Endpoints for an engagement |
| [`vayucloud_network_firewall`](data-sources/network_firewall.md) | Single firewall by ID |
| [`vayucloud_network_firewall_list`](data-sources/network_firewall_list.md) | Firewalls for an engagement and endpoint |
| [`vayucloud_network_firewall_rule`](data-sources/network_firewall_rule.md) | Single firewall rule by firewall ID and rule ID |
| [`vayucloud_network_firewall_rule_list`](data-sources/network_firewall_rule_list.md) | Firewall rules on a firewall |
| [`vayucloud_network_lb`](data-sources/network_lb.md) | Single load balancer by ID |
| [`vayucloud_network_lb_list`](data-sources/network_lb_list.md) | Load balancers for an engagement and endpoint |
| [`vayucloud_network_lb_virtualservice_options`](data-sources/network_lb_virtualservice_options.md) | Virtual service wizard options (zones, protocols, monitors) |
| [`vayucloud_network_lb_ssl_profiles`](data-sources/network_lb_ssl_profiles.md) | Uploaded SSL certificate profiles on a load balancer |
| [`vayucloud_network_c2s_vpn`](data-sources/network_c2s_vpn.md) | C2S VPN details for a firewall |
| [`vayucloud_network_c2s_vpn_users`](data-sources/network_c2s_vpn_users.md) | C2S VPN usernames for a firewall |
| [`vayucloud_network_zone`](data-sources/network_zone.md) | Single zone by ID |
| [`vayucloud_network_zone_list`](data-sources/network_zone_list.md) | Zones in an environment |
| [`vayucloud_network_public_ips`](data-sources/network_public_ips.md) | Public IP inventory for a firewall or engagement |
| [`vayucloud_file_server`](data-sources/file_server.md) | Single NAS vserver by ID |
| [`vayucloud_file_server_list`](data-sources/file_server_list.md) | NAS vservers for an engagement and endpoint |
| [`vayucloud_file_storage`](data-sources/file_storage.md) | Single NAS volume by name |
| [`vayucloud_nas_vlan_zone`](data-sources/nas_vlan_zone.md) | Whether a NAS VLAN zone exists for engagement/endpoint |
| [`vayucloud_s3_domain`](data-sources/s3_domain.md) | Single S3 domain |
| [`vayucloud_s3_domain_list`](data-sources/s3_domain_list.md) | S3 domains for engagement/endpoint |
| [`vayucloud_s3_bucket`](data-sources/s3_bucket.md) | Single S3 bucket |
| [`vayucloud_s3_bucket_list`](data-sources/s3_bucket_list.md) | Buckets in a domain |
| [`vayucloud_s3_object`](data-sources/s3_object.md) | S3 object metadata |
| [`vayucloud_s3_object_list`](data-sources/s3_object_list.md) | Objects in a bucket |
| [`vayucloud_s3_user`](data-sources/s3_user.md) | Single S3 user |
| [`vayucloud_s3_user_list`](data-sources/s3_user_list.md) | Users in a domain |
| [`vayucloud_s3_token`](data-sources/s3_token.md) | S3 token metadata |
| [`vayucloud_s3_token_list`](data-sources/s3_token_list.md) | Tokens in a domain |
| [`vayucloud_resource_group_business_unit`](data-sources/resource_group_business_unit.md) | Single business unit by ID |
| [`vayucloud_resource_group_business_unit_list`](data-sources/resource_group_business_unit_list.md) | Business units on a firewall |
| [`vayucloud_resource_group_environment`](data-sources/resource_group_environment.md) | Single environment by ID |
| [`vayucloud_resource_group_environment_list`](data-sources/resource_group_environment_list.md) | Environments in a business unit |
| [`vayucloud_virtualmachine`](data-sources/virtualmachine.md) | Single virtual machine by instance ID |
| [`vayucloud_virtualmachine_blockstorage`](data-sources/virtualmachine_blockstorage.md) | One attached volume on a VM by instance and volume ID |
| [`vayucloud_virtualmachine_list`](data-sources/virtualmachine_list.md) | Virtual machines in a zone |
| [`vayucloud_virtualmachine_image`](data-sources/virtualmachine_image.md) | Image catalog for a zone |
| [`vayucloud_virtualmachine_flavor`](data-sources/virtualmachine_flavor.md) | Flavor catalog for a zone |
| [`vayucloud_account_engagement_auditlog`](data-sources/account_engagement_auditlog.md) | Audit log entries |
| [`vayucloud_auditlog_details`](data-sources/auditlog_details.md) | Detailed audit log entry |
| [`vayucloud_list_security_group`](data-sources/list_security_group.md) | Security groups on a firewall |
| [`vayucloud_list_security_group_rules`](data-sources/list_security_group_rules.md) | Rules for a security group |
| [`vayucloud_virtualmachine_list_security_group_rules`](data-sources/virtualmachine_list_security_group_rules.md) | Security groups and rules on a VM, grouped by port |

For a dependency-oriented walkthrough and consolidated reference, see [Resources and data sources overview](resourcesAndDatasource.md).

### Guides

| Guide | Description |
|-------|-------------|
| [Load balancer (HAProxy)](guides/load_balancer.md) | End-to-end LB → SSL → virtual service → public IP; imports, VIP pinning, and operations |

## How the pieces fit together (high level)

Most teams build from the **outside in**: secure the network first, then organize the account, then add the network space for apps, and finally create virtual machines. You can follow the same story in one automation configuration, or skip steps when something already exists in your tenant.

1. **Sign in** — You provide a username and password (or equivalent secrets). The tool uses them to access VayuCloud on your behalf.
2. **Know your account context** — Confirm which **engagement** (account or project context) and **endpoint** (location or site) you are using. Your configuration can look these up automatically when needed.
3. **Network firewall** — Establishes the perimeter and connectivity for your network.
4. **Business unit** — Groups your resources under that firewall for organization and access.
5. **Environment** — A named stage inside the business unit (for example *Production* or *Development*).
6. **Network zone** — The segment where virtual machines get IP addresses and attach to the network.
7. **Virtual machine** — You pick an **operating system image** and a **size (flavor)**, then create the VM in that zone.
8. **Load balancer** *(optional)* — Enable HAProxy (or platform-determined type) on a firewall with **`zone_id`**, bandwidth, and pricing. See the [Load balancer guide](guides/load_balancer.md).
9. **SSL profile** *(optional)* — Upload a client certificate when the virtual service uses HTTPS.
10. **Virtual service** *(optional)* — Front-end listener and backend pool; reuse **`zone_id`** from the load balancer; discover pool VMs with `vayucloud_virtualmachine_list`.
11. **Public IP** *(optional)* — Associate a public IP to the virtual service VIP on the load balancer.
12. **Extra block storage** *(optional)* — Attach additional disks beyond the VM’s root volume when capacity or separation is needed.
13. **Power actions** *(optional)* — Start, stop, suspend, or reboot an existing VM when you need to change runtime state.
14. **Security groups** *(optional)* — Create or adopt a group and rules, then attach groups to a VM.
15. **NAS file storage** *(optional)* — File server → volume → export policy; optional NAS VLAN zone and `extend_nas_zone`; zone↔NAS firewall rules.
16. **S3 (VCS)** *(optional)* — Domain on a firewall → bucket, user, token, objects; optional public IP and `ill_to_vcs` firewall rule.

**Already have infrastructure?** If firewalls, zones, or VMs already exist in VayuCloud, your configuration can **reference** them instead of creating new ones—you do not have to build the full chain every time.

### For Terraform and OpenTofu users

The numbered flow above matches this provider in order: sign-in via the provider block → optional engagement and endpoint **data sources** → **resources** for firewall, business unit, environment, and zone → image/flavor **data sources** and the virtual machine **resource** → optional **load balancer** (with `zone_id`), **SSL profile**, and **virtual service** for HAProxy → optional **public IP** on the VS VIP → optional **virtual machine block storage** and virtual machine **state** for disks and power. Resource and data source names are listed in the tables in this document.

## Capabilities

| Capability | Details |
|------------|---------|
| Asynchronous operations | Create, update, delete, and relevant power actions wait for audit completion where the platform requires it. |
| Discovery | List and single-object data sources for engagements, endpoints, firewalls, zones, resource groups, and virtual machines. |
| Filtering | Many list data sources support `filter` blocks to narrow results after the API returns data. |
| Validation | Mutating operations typically verify referenced identifiers (engagement, endpoint, firewall, business unit, environment, zone) before proceeding. |
| Import | Supported where documented on individual resource pages (`terraform import` / `tofu import`). |

## Authentication

Configure credentials in the `provider "vayucloud"` block and/or with environment variables. **Values set in the provider block take precedence** over the environment when both are supplied.

| Setting | Argument | Environment variable |
|---------|----------|----------------------|
| Username | `username` | `VAYU_USERNAME` |
| Password | `password` | `VAYU_PASSWORD` |
| HTTP timeout in seconds (optional) | `timeout` | `VAYU_TIMEOUT` |

API endpoints are defined by the product; they are not configurable in the provider.

## Example usage

The following configuration declares the provider, resolves an engagement, and creates a firewall.

```hcl
terraform {
  required_providers {
    vayucloud = {
      source = "tatacommunicationsvayu/vayucloud"
    }
  }
}

provider "vayucloud" {
  username = var.vayucloud_username
  password = var.vayucloud_password
}

data "vayucloud_account_engagement" "by_name" {
  filter {
    name   = "engagement_name"
    values = ["My Engagement"]
  }
}

resource "vayucloud_network_firewall" "example" {
  engagement_id         = data.vayucloud_account_engagement.by_name.engagements[0].id
  endpoint_id           = {{endpoint_id}}
  firewall_display_name = "my-firewall"
  firewall_throughput   = "10Mbps"
  internet_bandwidth    = "10Mbps"
}
```

## Installation and CLI

Add the provider to `required_providers`, then initialize the working directory:

- **Terraform:** `terraform init`
- **OpenTofu:** `tofu init`

Run `terraform plan` / `apply` or `tofu plan` / `apply` as usual.

For verbose troubleshooting, set `TF_LOG=DEBUG`. To use a locally built provider binary during development, configure a **development overrides** block in the Terraform CLI configuration file (see the HashiCorp Terraform documentation for `terraform.rc` / `.terraformrc` and your operating system).

## Argument reference

| Argument | Type | Description |
|----------|------|-------------|
| `username` | String, sensitive | Required for API access unless `VAYU_USERNAME` is set. |
| `password` | String, sensitive | Required unless `VAYU_PASSWORD` is set. |
| `timeout` | Number | HTTP client timeout in **seconds**. Default `60`. May be set with `VAYU_TIMEOUT`. |

## See also

- [Resources and data sources overview](resourcesAndDatasource.md) — Dependencies and quick reference.
- [Load balancer guide](guides/load_balancer.md) — HAProxy end-to-end walkthrough.
