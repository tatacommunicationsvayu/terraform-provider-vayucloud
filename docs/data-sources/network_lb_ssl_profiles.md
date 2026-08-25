---
page_title: "vayucloud_network_lb_ssl_profiles Data Source"
subcategory: ""
description: |-
  Lists uploaded client SSL certificate profiles on a load balancer.
---

# `vayucloud_network_lb_ssl_profiles`

Returns SSL certificate profiles already uploaded on a load balancer. Use the `name` value as `certificate_name` on HTTPS [`vayucloud_network_lb_virtualservice`](../resources/network_lb_virtualservice.md) resources.

To upload a new certificate, use [`vayucloud_network_lb_ssl_profile`](../resources/network_lb_ssl_profile.md). See the [Load balancer guide](../guides/load_balancer.md) for the HTTPS flow.

## Example Usage

```hcl
data "vayucloud_network_lb_ssl_profiles" "certs" {
  load_balancer_id = {{load_balancer_id}}
}

output "cert_names" {
  value = [for p in data.vayucloud_network_lb_ssl_profiles.certs.ssl_profiles : p.name]
}
```

## Argument Reference

### Required

* `load_balancer_id` — (String) Load Balancer CI Master ID.

## Attributes Reference

* `ssl_profiles` — (List of Object) SSL certificate profiles on the load balancer.
  * `name` — (String) Certificate storage name (for example `my-cert.pem`). Use as `certificate_name` on HTTPS virtual services.
  * `full_path` — (String) Full storage path on the load balancer.
