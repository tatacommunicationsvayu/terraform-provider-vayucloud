---
page_title: "vayucloud_network_c2s_vpn_users Data Source"
subcategory: ""
description: |-
  Lists C2S VPN usernames for a firewall using the same read API as vayucloud_network_c2s_vpn.
---

# `vayucloud_network_c2s_vpn_users`

Returns the **user name list** for the C2S VPN on a firewall. Calls the same action-state read as [`vayucloud_network_c2s_vpn`](network_c2s_vpn.md) and exposes only `firewall_id` and `users` (names). Use [`vayucloud_network_c2s_vpn`](network_c2s_vpn.md) when you need the full VPN payload (PSK, IP, peer ID, and so on).

## Example Usage

```hcl
data "vayucloud_network_c2s_vpn_users" "example" {
  firewall_id = {{firewall_id}}
}
```

## Argument Reference

### Required

* `firewall_id` — (Number) Firewall resource ID the C2S VPN is attached to.

## Attributes Reference

* `id` — (String) Same as `firewall_id` as a string.
* `users` — (List of objects) Each object has:
  * `name` — (String) Username. Passwords are not returned by the API.
