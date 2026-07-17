---
page_title: "vayucloud_network_c2s_vpn_user Resource"
subcategory: ""
description: |-
  Manages C2S VPN users on a firewall where the VPN already exists. User add, remove, and password reset operations are asynchronous.
---

# `vayucloud_network_c2s_vpn_user`

Manages **only** the VPN user list for an existing C2S VPN on a given `firewall_id`. Use this when the VPN is already provisioned (for example via `vayucloud_network_c2s_vpn` or outside Terraform) and you want a separate resource to own user lifecycle.

**Create** adds every user in `users` in one API batch; each username must **not** already exist on the VPN.

**Update** removes users dropped from configuration, adds new users, then applies password resets where the password changed in configuration.

**Delete** removes all users listed in the last applied state from the VPN (if still present).

The provider validates `firewall_id` and duplicate usernames at plan time (`ModifyPlan`).

Changing `firewall_id` forces a new resource.

## Example Usage

```hcl
resource "vayucloud_network_c2s_vpn_user" "example" {
  firewall_id = {{firewall_id}}

  users = [
    {
      name     = "app_user"
      password = var.vpn_user_initial_password
    },
  ]
}
```

Set `vpn_user_initial_password` (sensitive string) via `terraform.tfvars`, `-var`, or `TF_VAR_vpn_user_initial_password`; it must satisfy the same rules as for [`vayucloud_network_c2s_vpn`](network_c2s_vpn.md).

## Argument Reference

### Required

* `firewall_id` — (Number) Firewall ID the C2S VPN is attached to. Changing this forces a new resource.
* `users` — (Block list, at least one) VPN users. Same rules as [`vayucloud_network_c2s_vpn`](network_c2s_vpn.md):
  * `name` — (String) Username (letters, digits, `_`, `-` only).
  * `password` — (String, sensitive) Password with length and complexity rules as on `vayucloud_network_c2s_vpn`.

## Attributes Reference

There are no additional computed attributes; state tracks the configured `firewall_id` and `users` (including sensitive passwords).

## Import

Import is not supported for this resource.
