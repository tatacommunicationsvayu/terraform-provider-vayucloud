---
page_title: "vayucloud_network_lb_virtualservice_options Data Source"
subcategory: ""
description: |-
  Lists HAProxy virtual service configuration options for a load balancer. Optional filters narrow zones client-side.
---

# `vayucloud_network_lb_virtualservice_options`

Returns virtual service wizard options for a load balancer: protocols, pool algorithms, health monitors, persistence types, and LB-eligible zones.

For the full module walkthrough, see the [Load balancer guide](../guides/load_balancer.md).

## When to use

| Need | Use |
|------|-----|
| Zone for LB + VS + pool VMs | **`zone_id` on [`vayucloud_network_lb`](../resources/network_lb.md)** (validated against firewall before enable) |
| Protocols, algorithms, monitors, persistence | **This data source** |
| Confirm zone is LB-eligible | Optional: compare `zones[].id` with `vayucloud_network_lb.zone_id` |
| Pool member VM names/IPs | [`vayucloud_virtualmachine_list`](virtualmachine_list.md) with `vayucloud_network_lb.zone_id` |
| HTTPS certificate names already on LB | [`vayucloud_network_lb_ssl_profiles`](network_lb_ssl_profiles.md) |

Requires a load balancer CI Master ID (`vayucloud_network_lb.id`).

## Example Usage

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

output "supported_protocols" {
  value = data.vayucloud_network_lb_virtualservice_options.opts.protocols
}

output "lb_eligible_zones" {
  value = data.vayucloud_network_lb_virtualservice_options.opts.zones
}
```

### Filter zones by name (optional validation)

```hcl
data "vayucloud_network_lb_virtualservice_options" "opts" {
  load_balancer_id = vayucloud_network_lb.lb.id
  monitor_type     = "http"

  filter {
    name   = "name"
    values = ["My Zone Name"]
  }
}
```

For HTTPS certificate names, use [`vayucloud_network_lb_ssl_profiles`](network_lb_ssl_profiles.md).

## Argument Reference

### Required

* `load_balancer_id` — (String) Load Balancer CI Master ID.

### Optional

* `monitor_type` — (String) Protocol family used to filter monitors (`http` or `tcp`). Default `http`.
* `filter` — (Block) Repeatable. Each block requires `name` and `values` (list of strings). Applied client-side to narrow **zones** after the API returns data.

## Attributes Reference

* `protocols` — (List of Object) Supported protocols. Use the `value` field in the virtual service resource.
  * `label` — (String) Display label.
  * `value` — (String) API value (`http`, `https`, `tcp`).
  * `ssl_required` — (Bool) Whether SSL is required for this protocol.
* `algorithms` — (List of Object) Supported pool algorithms.
  * `label` — (String) Display label.
  * `value` — (String) API value (for example `roundrobin`).
* `persistence_types` — (List of Object) Supported persistence types.
  * `label` — (String) Display label.
  * `value` — (String) API value.
  * `input_required` — (Bool) Whether a persistence input value is required.
  * `input_label` — (String) Label for the persistence input field.
* `monitors` — (List of Object) Health monitors for the selected `monitor_type`.
  * `name` — (String) Monitor name (for example `httpchk`).
  * `full_path` — (String) Full monitor path.
* `zones` — (List of Object) Zones available for virtual service placement on this load balancer (informational; prefer `zone_id` on the LB resource).
  * `id` — (Number) Zone ID.
  * `name` — (String) Zone name.
  * `environment_id` — (Number) Environment identifier.
  * `environment_name` — (String) Environment name.
  * `department_id` — (Number) Business unit identifier.
  * `department_name` — (String) Business unit name.

## See also

* [Load balancer guide](../guides/load_balancer.md) — End-to-end HAProxy walkthrough.
* [`vayucloud_network_lb`](../resources/network_lb.md) — Parent load balancer; **`zone_id` is set here**.
* [`vayucloud_network_lb_virtualservice`](../resources/network_lb_virtualservice.md) — Create the virtual service.
* [`vayucloud_virtualmachine_list`](virtualmachine_list.md) — Pool member VM discovery.
