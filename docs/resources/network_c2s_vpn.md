---
page_title: "vayucloud_network_c2s_vpn Resource"
subcategory: ""
description: |-
  Provides a C2S VPN on an existing VayuCloud network firewall. Create and user mutations are asynchronous; the provider polls audit logs and updates action-state (module=c2svpn).
---

# `vayucloud_network_c2s_vpn`

Manages a C2S VPN attached to a **network firewall**. The platform provisions the VPN asynchronously; the provider waits for audit completion and syncs action-state. Read uses the action-state API (`module=c2svpn`, `action=read`).

At plan time the provider verifies that `firewall_id` and `zone_id` exist in the platform. `zone_id` is required for create and is kept in Terraform state from configuration; the VPN read API does not return it, so you must continue to set `zone_id` in configuration (including after import).

Initial create sends one user to the create API; if `users` contains more than one entry, additional users are added in a follow-up API call. Updates reconcile users by remove → add → password reset (same semantics as the underlying VPN user APIs).

Changing `firewall_id`, `zone_id`, or `pricing_model` forces replacement.

## Example Usage

```hcl
resource "vayucloud_network_c2s_vpn" "example" {
  firewall_id   = {{firewall_id}}
  zone_id       = {{zone_id}}
  pricing_model = "daily"

  users = [
    {
      name     = "john_doe"
      password = var.vpn_user_initial_password
    },
  ]
}
```

Set `vpn_user_initial_password` (sensitive string) via `terraform.tfvars`, `-var`, or `TF_VAR_vpn_user_initial_password`; it must satisfy the password rules below.

## Argument Reference

The following arguments are supported:

### Required

* `firewall_id` — (Number) Firewall resource ID the VPN is created on. Must exist in the platform. Changing this forces a new resource.
* `zone_id` — (Number) Network zone ID associated with the VPN. Must exist in the platform. Required at create; not returned by the read API, so the value in configuration is preserved in state. Changing this forces a new resource.
* `pricing_model` — (String) One of `daily`, `monthly`, `reserved_1`, `reserved_2`, `reserved_3` (case-insensitive). Changing this forces a new resource.
* `users` — (Block list, at least one) VPN users. Nested attributes:
  * `name` — (String) Username. Only letters, digits, `_`, and `-`.
  * `password` — (String, sensitive) Password: length 8–50; must include lowercase, uppercase, digit, and at least one special character from the set `< > ( ) # "` (only characters allowed by the provider’s charset are accepted—for example `#` or `(`). Allowed characters are letters, digits, and `!#$%^&*()_+-=[]{}:;,.?/~`.

Usernames cannot be renamed in place; remove a user and add a new name instead.

## Attributes Reference

In addition to the arguments above, the following attributes are exported:

* `id` — (String) Same as `firewall_id` as a string.
* `audit_id` — (String) Audit identifier from the last completed async operation.
* `status` — (String) Audit status from the last completed operation.
* `pre_shared_key` — (String, sensitive) Pre-shared key from the platform (read-only).
* `vpn_name` — (String) VPN name from the platform.
* `vpn_status` — (String) VPN status from the platform.
* `vpn_ip` — (String) VPN IP from the platform.
* `peer_id` — (String) Peer ID from the platform.
* `vpn_no_of_users` — (Number) Number of VPN users reported by the platform.

## Import

Import is supported using the numeric **firewall** ID (the VPN scope on the platform):

```shell
terraform import vayucloud_network_c2s_vpn.example {{firewall_id}}
```

OpenTofu: use the same syntax with `tofu import`.

After import, set `zone_id`, `pricing_model`, and `users` in configuration to match desired state before the next apply. `zone_id` is not populated from the platform during import; you must supply the correct zone ID explicitly.
