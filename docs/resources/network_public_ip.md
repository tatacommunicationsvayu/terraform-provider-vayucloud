---
page_title: "vayucloud_network_public_ip Resource"
subcategory: ""
description: |-
  Associates a public IP with a VayuCloud platform resource (zone, virtual machine, firewall, bare metal, or load balancer). Create and destroy wait on audit completion.
---

# `vayucloud_network_public_ip`

Associates a public IPv4 address with a private IP on a supported resource using the network operations **associate** API. After the audit completes, the provider reads **action state** (`module=publicIp`, `action=read`) to fill `public_ip`.

**Destroy** runs **dissociate**. The dissociate payload uses **`retain_on_dissociate` from the last applied Terraform state** (not only from config), so change that flag via `apply` before destroying if you need a specific behavior. Destroy also requires `public_ip` to be present in state; if it is empty, run refresh so the provider can read the address from the platform, then destroy again.

## Updates and replacement

* **`retain_on_dissociate`** may be updated **in place** (no API call; it affects the next dissociate).
* **`public_ip_pricing_model`** is fixed at associate time. Changing it in configuration without replacement produces an error.
* **`resource_type`**, **`resource_id`**, and **`private_ip`** define the association identity; changing them is a **replace** (destroy + create).

For `resource_type` `zone` and `virtualmachine`, the provider validates that `resource_id` exists at **plan** (when the client is configured) and again at **create**. Create fails if the same private IP already has an association on that target.

## Example Usage

### Virtual machine

```hcl
resource "vayucloud_network_public_ip" "vm_primary" {
  resource_type           = "virtualmachine"
  resource_id             = {{virtualmachine_id}}
  private_ip              = "10.0.0.10"
  public_ip_pricing_model = "monthly"
  retain_on_dissociate    = true
}
```

### Load balancer virtual service VIP

Associate a public IP to the **private VIP** of an HAProxy virtual service. Pin `vip_ip` on the virtual service (see [`vayucloud_network_lb_virtualservice`](network_lb_virtualservice.md)) so `private_ip` stays stable across updates.

```hcl
resource "vayucloud_network_public_ip" "vs_vip" {
  resource_type           = "loadbalancer"
  resource_id             = tonumber(vayucloud_network_lb.lb.id)
  private_ip              = vayucloud_network_lb_virtualservice.app_vs.vip_ip
  public_ip_pricing_model = "daily"
  retain_on_dissociate    = true

  depends_on = [vayucloud_network_lb_virtualservice.app_vs]
}
```

Use **`https://`** when testing HTTPS listeners. Ensure firewall rules allow the listener port (for example TCP 443).

## Argument Reference

### Required

* `resource_type` — (String) Path segment for the associate API. One of: `zone`, `virtualmachine`, `firewall`, `baremetal`, `loadbalancer`. Changing forces replacement.
* `resource_id` — (Number) Platform identifier of the target resource. Changing forces replacement.
* `private_ip` — (String) Private IPv4 address to NAT with the allocated public IP. Must be valid IPv4. Changing forces replacement.

### Optional

* `public_ip_pricing_model` — (String) Pricing model: `daily` (default), `monthly`, `reserved_1`, `reserved_2`, or `reserved_3` (case-insensitive; mapped for the API). Cannot be changed in place after associate; replace the resource to change pricing.
* `retain_on_dissociate` — (Boolean) Whether to retain the public IP in the platform when dissociating. Default `true`.
* `public_ip` — (String) Optional/computed. Normally left unset; populated from the platform after associate and on read.

## Attributes Reference

In addition to the arguments above, the following attributes are exported:

* `id` — (String) Stored after create as the target **`resource_id` as a decimal string** (not the full association tuple). The association is still defined by `resource_type`, `resource_id`, and `private_ip`.
* `public_ip` — (String) Public IPv4 from action-state read for this association.
* `audit_id` — (String) Audit ID from the last completed associate or dissociate operation tracked by the provider.
* `status` — (String) Audit status from that operation.

## See also

* [Load balancer guide](../guides/load_balancer.md) — VS VIP association pattern.
* [`vayucloud_network_lb_virtualservice`](network_lb_virtualservice.md) — Source of `vip_ip` for load balancer associations.
* [`vayucloud_network_public_ips`](../data-sources/network_public_ips.md) — List public IP inventory.

## Import

Import is not supported for this resource.
