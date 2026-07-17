---
page_title: "vayucloud_network_c2s_vpn Data Source"
subcategory: ""
description: |-
  Reads C2S VPN details for a firewall via the action-state API (module=c2svpn, action=read).
---

# `vayucloud_network_c2s_vpn`

Returns current C2S VPN state for one firewall. This uses the same read path as the [`vayucloud_network_c2s_vpn`](../resources/network_c2s_vpn.md) resource refresh.

## Example Usage

```hcl
data "vayucloud_network_c2s_vpn" "example" {
  firewall_id = {{firewall_id}}
}
```

## Argument Reference

### Required

* `firewall_id` — (Number) Firewall resource ID the VPN is attached to.

## Attributes Reference

* `id` — (String) Same as `firewall_id` as a string.
* `pre_shared_key` — (String, sensitive) Pre-shared key from the platform.
* `vpn_name` — (String) VPN name.
* `vpn_status` — (String) VPN status.
* `vpn_ip` — (String) VPN IP.
* `peer_id` — (String) Peer ID.
* `vpn_no_of_users` — (Number) Number of VPN users reported by the platform.
* `users` — (List of objects) Users returned by the API; each object has:
  * `name` — (String) Username. Passwords are not returned.
* `status` — (String) Top-level API response status (for example `success`).
* `message` — (String) API message text.
* `response_code` — (Number) API response code (`0` indicates success).
* `raw_response` — (String) Raw JSON `data` payload from the action-state response (for debugging).
