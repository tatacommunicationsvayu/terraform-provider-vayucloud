---
page_title: "Network and Connectivity"
description: |-
  VayuCloud firewalls, zones, firewall rules, public IPs, load balancers, virtual services, certificates, and VPN documentation.
---

# Network and Connectivity

This family provides the network foundation and internet-facing services. Build from firewall to zone, then add rules, load balancing, VPN, or public-IP associations.

## Prerequisites

Know the engagement, endpoint, and parent firewall identifiers. A zone must belong to the firewall used by dependent services.

## Firewall and zones

- [Firewall resource](../resources/network_firewall.md)
- [Firewall resource data source](../data-sources/network_firewall.md)
- [Firewall list data source](../data-sources/network_firewall_list.md)
- [Firewall rule resource](../resources/network_firewall_rule.md)
- [Firewall rule data source](../data-sources/network_firewall_rule.md)
- [Firewall rule list data source](../data-sources/network_firewall_rule_list.md)
- [Network zone resource](../resources/network_zone.md)
- [Network zone data source](../data-sources/network_zone.md)
- [Network zone list data source](../data-sources/network_zone_list.md)

## Load balancer and virtual services

- [Load balancer resource](../resources/network_lb.md)
- [Load balancer data source](../data-sources/network_lb.md)
- [Load balancer list data source](../data-sources/network_lb_list.md)
- [Virtual service resource](../resources/network_lb_virtualservice.md)
- [Virtual service options data source](../data-sources/network_lb_virtualservice_options.md)
- [SSL profile resource](../resources/network_lb_ssl_profile.md)
- [SSL profiles data source](../data-sources/network_lb_ssl_profiles.md)

## VPN and public IPs

- [C2S VPN resource](../resources/network_c2s_vpn.md)
- [C2S VPN data source](../data-sources/network_c2s_vpn.md)
- [C2S VPN user resource](../resources/network_c2s_vpn_user.md)
- [C2S VPN users data source](../data-sources/network_c2s_vpn_users.md)
- [Public IP association resource](../resources/network_public_ip.md)
- [Public IP inventory data source](../data-sources/network_public_ips.md)

## Behavior notes

Network operations commonly wait for audit completion. Load-balancer virtual services may receive a VIP when `vip_ip` is omitted; preserve the resulting VIP before creating a dependent public-IP association. HTTPS requires a PEM certificate uploaded through the SSL profile resource.
