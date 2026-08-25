---
page_title: "vayucloud_network_lb Resource"
subcategory: ""
description: |-
  Enables and manages a VayuCloud Load Balancer on a firewall. Create, update (bandwidth), and delete operations are asynchronous; the provider polls audit logs until completion.
---

# `vayucloud_network_lb`

Enables a load balancer on an existing network firewall. The platform returns an audit identifier immediately; the provider polls the audit log until the operation finishes.

**Enable timing:** First-time enable typically takes **20–30 minutes**. The platform provisions internal business unit, environment, and zone resources, internal VMs, and VIP capacity before the load balancer is ready. Plan change windows accordingly.

LB type (for example `HAProxy`, `F5`, `AVI`) is determined by the firewall hypervisor and populated after create or read. Only **`bandwidth`** can be updated in place. Changing **`firewall_id`** on an existing resource forces replacement.

**`zone_id`** is **required**. The provider validates that the zone exists on the firewall before enable. Reuse it on [`vayucloud_network_lb_virtualservice`](network_lb_virtualservice.md) and [`vayucloud_virtualmachine_list`](../data-sources/virtualmachine_list.md) for pool member discovery.

Bandwidth is always expressed in **Mbps**. Values such as `100` and `100Mbps` both mean 100 Mbps. Gbps is not supported directly — use the Mbps equivalent (for example `1000` for 1 Gbps). Maximum **1000** Mbps per platform limits.

For the full HAProxy module walkthrough (SSL, virtual services, pool members, VIP pinning, public IP), see the [Load balancer guide](../guides/load_balancer.md).

Example configuration: [`examples/IaaS/resources/vayucloud_network_lb`](../../examples/IaaS/resources/vayucloud_network_lb/resource.tf). Import helper: [`import.sh`](../../examples/IaaS/resources/vayucloud_network_lb/import.sh).

## Example Usage

```hcl
resource "vayucloud_network_lb" "example" {
  firewall_id   = {{firewall_id}}
  zone_id       = {{zone_id}}
  bandwidth     = "100"
  pricing_model = "daily"
}

data "vayucloud_virtualmachine_list" "pool_vms" {
  zone_id = vayucloud_network_lb.example.zone_id
}
```

Resolve `zone_id` from [`vayucloud_network_zone_list`](../data-sources/network_zone_list.md) or [`vayucloud_network_zone`](../data-sources/network_zone.md) for the target firewall.

### Modify bandwidth

```hcl
resource "vayucloud_network_lb" "example" {
  firewall_id   = {{firewall_id}}
  zone_id       = {{zone_id}}
  bandwidth     = "200"
  pricing_model = "daily"
}
```

## Argument Reference

### Required

* `firewall_id` — (String) Firewall CI Master ID on which to enable the load balancer. Changing an existing non-empty value forces resource replacement.
* `zone_id` — (Number) Network zone ID on the firewall for virtual service placement and pool member VM discovery. **Required.** Validated against `firewall_id` before enable.
* `bandwidth` — (String) Load balancer bandwidth in Mbps (for example `100` or `100Mbps`).

### Optional

* `pricing_model` — (String) LB pricing model sent on enable or modify (for example `daily`, `monthly`). Default `daily`.
* `engagement_id` — (Number) Engagement ID. Populated from the platform after create or read.
* `endpoint_id` — (Number) Endpoint ID. Populated from the platform after create or read.
* `name` — (String) Load balancer CI name. Populated from the platform.
* `type` — (String) Load balancer type (`F5`, `HAProxy`, `AVI`). Populated from the platform.
* `display_name` — (String) Display name. Populated from the platform.

## Attributes Reference

In addition to arguments above, the following attributes are exported:

* `id` — (String) Load Balancer CI Master ID (returned from audit after create).
* `ci_status` — (String) CI status (for example `ACTIVE`, `INACTIVE`).
* `audit_id` — (String) Audit identifier for the latest completed operation.
* `status` — (String) Audit status.
* `last_updated` — (String) Timestamp of the last update.

## Import

Import is supported using the load balancer CI Master ID:

```shell
terraform import vayucloud_network_lb.example <lb_id>
```

Optional format with engagement ID:

```shell
terraform import vayucloud_network_lb.example <lb_id>:<engagement_id>
```

OpenTofu: use the same syntax with `tofu import`.

After import, set **`zone_id`** in configuration (the platform does not return it on read). The provider refreshes `firewall_id`, `bandwidth`, and other computed fields from the platform.

## See also

* [Load balancer guide](../guides/load_balancer.md) — End-to-end HAProxy walkthrough.
* [`vayucloud_network_lb_list`](../data-sources/network_lb_list.md) — List load balancers for an engagement and endpoint.
* [`vayucloud_network_lb`](../data-sources/network_lb.md) — Read a single load balancer by ID (does not include `zone_id`).
* [`vayucloud_network_lb_virtualservice`](network_lb_virtualservice.md) — Create virtual services on the load balancer.
* [`vayucloud_network_lb_ssl_profile`](network_lb_ssl_profile.md) — Upload certificates for HTTPS.
* [`vayucloud_virtualmachine_list`](../data-sources/virtualmachine_list.md) — List VMs in the LB zone for pool members.
* [`vayucloud_network_public_ip`](network_public_ip.md) — Associate a public IP to the VS VIP.
