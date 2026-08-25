---
page_title: "vayucloud_network_lb_virtualservice Resource"
subcategory: ""
description: |-
  Manages an HAProxy Virtual Service on a VayuCloud Load Balancer. Create, update, and delete operations are asynchronous where the platform requires audit polling.
---

# `vayucloud_network_lb_virtualservice`

Manages an HAProxy virtual service (listener, pool, and health monitors) on an existing load balancer.

For the full module walkthrough, see the [Load balancer guide](../guides/load_balancer.md).

Example configuration: [`examples/IaaS/resources/vayucloud_network_lb_virtualservice`](../../examples/IaaS/resources/vayucloud_network_lb_virtualservice/resource.tf).

## Prerequisites

| Step | Resource / data source |
|------|-------------------------|
| Enable LB | [`vayucloud_network_lb`](network_lb.md) with `zone_id` |
| Wizard options | [`vayucloud_network_lb_virtualservice_options`](../data-sources/network_lb_virtualservice_options.md) — protocols, algorithms, monitors, persistence |
| Pool member VMs | [`vayucloud_virtualmachine_list`](../data-sources/virtualmachine_list.md) using `vayucloud_network_lb.<name>.zone_id` |
| HTTPS cert | [`vayucloud_network_lb_ssl_profile`](network_lb_ssl_profile.md) |
| Public access *(optional)* | [`vayucloud_network_public_ip`](network_public_ip.md) on the VS `vip_ip` |

Set **`zone_id`** from the parent load balancer (`vayucloud_network_lb.<name>.zone_id`), not from a separate zone discovery step.

Changing **`load_balancer_id`** or **`name`** forces replacement. Other fields (port, protocol, pool members, monitors, VIP, and so on) can be updated in place.

### HTTPS protocol

Set `protocol = "https"` in Terraform. The provider maps this to the HAProxy API mode `http` on create/update while keeping `https` in state and plan output.

### VIP stability

Omit `vip_ip` on first create for platform auto-assignment. After create, refresh populates `vip_ip` in state. On later plans the provider keeps that state value when `vip_ip` is still omitted (`UseStateForUnknown` / `ModifyPlan`), so [`vayucloud_network_public_ip`](network_public_ip.md) (`private_ip` = VIP) does not force-replace.

Optionally **pin** `vip_ip` in configuration to the assigned address after the first apply if you want the value explicit in config.

## Example Usage

### HTTP virtual service with pool discovery

```hcl
resource "vayucloud_network_lb" "lb" {
  firewall_id = {{firewall_id}}
  zone_id     = {{zone_id}}
  bandwidth   = "100"
}

data "vayucloud_network_lb_virtualservice_options" "opts" {
  load_balancer_id = vayucloud_network_lb.lb.id
  monitor_type     = "http"
}

data "vayucloud_virtualmachine_list" "pool_vms" {
  zone_id = vayucloud_network_lb.lb.zone_id
}

locals {
  pool_members = [
    for vm in data.vayucloud_virtualmachine_list.pool_vms.virtual_machines : {
      name       = vm.name
      ip_address = vm.ip
      port       = 8080
    }
  ]
}

resource "vayucloud_network_lb_virtualservice" "http" {
  load_balancer_id = vayucloud_network_lb.lb.id
  name             = "app-vs-http"
  zone_id          = vayucloud_network_lb.lb.zone_id
  port             = "80"
  protocol         = "http"
  pool_algorithm   = data.vayucloud_network_lb_virtualservice_options.opts.algorithms[0].value
  monitor          = ["httpchk"]

  dynamic "pool_member" {
    for_each = local.pool_members
    content {
      name       = pool_member.value.name
      ip_address = pool_member.value.ip_address
      port       = pool_member.value.port
    }
  }
}
```

### HTTPS virtual service

```hcl
resource "vayucloud_network_lb" "lb" {
  firewall_id = {{firewall_id}}
  zone_id     = {{zone_id}}
  bandwidth   = "100"
}

resource "vayucloud_network_lb_ssl_profile" "cert" {
  load_balancer_id = vayucloud_network_lb.lb.id
  certificate_name = "my-cert"
  certificate      = file("cert.pem")
  private_key      = file("key.pem")
}

resource "vayucloud_network_lb_virtualservice" "https" {
  load_balancer_id = vayucloud_network_lb.lb.id
  name             = "app-vs-https"
  zone_id          = vayucloud_network_lb.lb.zone_id
  port             = "443"
  protocol         = "https"
  pool_algorithm   = "roundrobin"
  monitor          = ["httpchk"]
  certificate_name = "my-cert.pem"
  # vip_ip = "10.x.x.x" # omit on first create; pin after terraform output vip_ip

  depends_on = [vayucloud_network_lb_ssl_profile.cert]

  pool_member {
    name       = "web-vm-1"
    ip_address = "10.0.0.10"
    port       = 8080
  }
}

resource "vayucloud_network_public_ip" "vs_vip" {
  resource_type           = "loadbalancer"
  resource_id             = tonumber(vayucloud_network_lb.lb.id)
  private_ip              = vayucloud_network_lb_virtualservice.https.vip_ip
  public_ip_pricing_model = "daily"
  retain_on_dissociate    = true

  depends_on = [vayucloud_network_lb_virtualservice.https]
}
```

## Argument Reference

### Required

* `load_balancer_id` — (String) Parent Load Balancer CI Master ID. Forces replacement if changed.
* `name` — (String) Virtual service name. Must be unique per load balancer. Forces replacement if changed.
* `zone_id` — (Number) Zone ID for virtual service placement. Use `vayucloud_network_lb.<name>.zone_id` from the parent load balancer.
* `port` — (String) Listener port (for example `80`, `443`, or comma-separated `80,443`).
* `protocol` — (String) `http`, `https`, or `tcp` (case-insensitive). For `https`, set `certificate_name` and an SSL port such as `443`.

### Optional

* `pool_algorithm` — (String) Pool algorithm API value (for example `roundrobin`, `leastconn`, `source`). Default `roundrobin`.
* `monitor` — (List of String) Health monitor names (for example `httpchk` for HTTP/HTTPS, `tcp-check` for TCP). Defaults based on protocol when omitted.
* `vip_ip` — (String) Private virtual IP for the listener. Optional on create (platform auto-assigns). **Pin after first create** to keep the same VIP on updates and avoid public IP reassociation.
* `persistence_type` — (String) Persistence API value (for example `PERSISTENCE_TYPE_HTTP_COOKIE`).
* `persistence_value` — (String) Persistence input value such as cookie name when required by the persistence type.
* `certificate_name` — (String) Client SSL certificate **storage name on the load balancer**. Required when `protocol` is `https`. Use the `.pem` name from an SSL profile (for example `my-cert.pem`). This is **not** a filesystem path — upload the PEM files with [`vayucloud_network_lb_ssl_profile`](network_lb_ssl_profile.md) first. The provider resolves the full HAProxy path from the LB SSL profile list.

### `pool_member` block

At least one block is required.

* `name` — (String) Pool member name (typically the VM name).
* `ip_address` — (String) Pool member IP address (from `vayucloud_virtualmachine_list` or VM resource output).
* `port` — (Number) Backend port on the pool member.

## Attributes Reference

* `id` — (String) Composite identity `{load_balancer_id}/{name}`.
* `vip_ip` — (String) Private VIP assigned to the virtual service (populated after create/read).
* `ci_status` — (String) CI status (for example `ACTIVE`, `INACTIVE`).
* `audit_id` — (String) Audit identifier for the latest operation.
* `status` — (String) Audit status.
* `last_updated` — (String) Timestamp of the last update.

## Import

Import using the composite ID `{load_balancer_id}/{name}`:

```shell
terraform import vayucloud_network_lb_virtualservice.example <load_balancer_id>/<vs_name>
```

OpenTofu: use the same syntax with `tofu import`.

After import, set `zone_id` and pool members in configuration if not populated from the platform read.

## See also

* [Load balancer guide](../guides/load_balancer.md) — End-to-end HAProxy walkthrough.
* [`vayucloud_network_lb`](network_lb.md) — Enable the parent load balancer (requires `zone_id`).
* [`vayucloud_network_lb_ssl_profile`](network_lb_ssl_profile.md) — Upload certificates for HTTPS.
* [`vayucloud_network_lb_virtualservice_options`](../data-sources/network_lb_virtualservice_options.md) — Discovery for protocols, algorithms, monitors, persistence.
* [`vayucloud_virtualmachine_list`](../data-sources/virtualmachine_list.md) — Discover pool member names and IPs.
* [`vayucloud_network_public_ip`](network_public_ip.md) — NAT public IP to the VS VIP.
