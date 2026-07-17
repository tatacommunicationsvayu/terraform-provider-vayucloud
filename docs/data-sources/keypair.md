---
page_title: "vayucloud_keypair Data Source"
subcategory: ""
description: |-
  Lists SSH key pairs for a VayuCloud engagement.
---

# `vayucloud_keypair`

Lists key pairs registered for `engagement_id`. Use the results to resolve `id` values for automation or to verify existing keys before creating a [`vayucloud_keypair` resource](../resources/keypair.md).

The list response does not include full public key material in typical API responses; the `public_key` attribute may be empty per object.

## Example Usage

```hcl
data "vayucloud_keypair" "all" {
  engagement_id = "1602"
}

output "keypair_names" {
  value = [for kp in data.vayucloud_keypair.all.keypairs : kp.name]
}
```

### Select a key pair by name

```hcl
data "vayucloud_keypair" "all" {
  engagement_id = "1602"
}

locals {
  ssh_key = [
    for kp in data.vayucloud_keypair.all.keypairs : kp
    if kp.name == "my-ssh-key"
  ][0]
}
```

## Argument Reference

### Required

* `engagement_id` — (String) Engagement identifier.

## Attributes Reference

* `id` — (String) Computed identifier (typically derived from the engagement).
* `keypairs` — (List of Object) Each object includes:
  * `id` — (Number) Key pair identifier.
  * `name` — (String) Key pair name.
  * `keypair_type` — (String) For example `RSA`.
  * `public_key` — (String) Often empty in list responses.
  * `engagement_id` — (String) Engagement identifier.
  * `fingerprint` — (String) Key fingerprint.
  * `created_at` — (String) Creation timestamp.
* `status` — (String) API status.
* `message` — (String) API message.
* `response_code` — (Number) API response code (`0` indicates success).
* `raw_response` — (String) Raw JSON payload for troubleshooting.

Use `for` expressions in configuration to filter by `name` or `keypair_type`. To create or upload a key and obtain private key material, use the key pair **resource** in create mode.
