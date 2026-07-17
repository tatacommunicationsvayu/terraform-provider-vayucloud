---
page_title: "vayucloud_keypair Resource"
subcategory: ""
description: |-
  Manages an SSH key pair in VayuCloud: generate a new key or upload an existing public key.
---

# `vayucloud_keypair`

Manages an SSH key pair in VayuCloud for use with virtual machines. Public material is stored in the platform; in **create** mode the private key is returned and stored in Terraform state.

**Modes**

* **`create`** — Platform generates a key pair. Set `keypair_type` and `private_key_file_format`. Attributes `private_key_content` and `private_key_filename` are populated.
* **`upload`** — Supply `public_key` for an existing key. No private key is returned.

Key pairs are scoped to an engagement. Reference a key pair from `vayucloud_virtualmachine` using a `login_account` block with `keypair_id`.

**Security:** In `create` mode, protect Terraform state (encryption, restricted backend). Treat `private_key_content` as sensitive.

## Example Usage

### Create a new key pair

```hcl
resource "vayucloud_keypair" "example" {
  name                    = "my-keypair"
  engagement_id           = "1602"
  mode                    = "create"
  keypair_type            = "RSA"
  private_key_file_format = "pem"
}
```

### Upload an existing public key

```hcl
resource "vayucloud_keypair" "example" {
  name          = "my-uploaded-keypair"
  engagement_id = "1602"
  mode          = "upload"
  public_key    = file("~/.ssh/id_rsa.pub")
}
```

## Argument Reference

### Required

* `name` — (String) Key pair name. Unique within the engagement. Changing forces replacement.
* `engagement_id` — (String) Engagement identifier. Changing forces replacement.
* `mode` — (String) `create` or `upload`. Changing forces replacement.

### Optional

* `keypair_type` — (String) For example `RSA` or `ED25519`. Required when `mode` is `create`. Changing forces replacement.
* `private_key_file_format` — (String) `pem` or `ppk`. Required when `mode` is `create`. Changing forces replacement.
* `public_key` — (String, Sensitive) Public key material. Required when `mode` is `upload`. Changing forces replacement.

## Attributes Reference

* `id` — (String) Key pair identifier.
* `private_key_content` — (String, Sensitive) Private key PEM or PPK content when `mode` is `create`.
* `private_key_filename` — (String) Suggested filename when `mode` is `create`.
* `fingerprint` — (String) Key fingerprint.
* `status` — (String) API status field.
* `message` — (String) API message field.

## Import

Format: `<engagement_id>/<keypair_id>`.

```shell
terraform import vayucloud_keypair.example <engagement_id>/<keypair_id>
```

```shell
terraform import vayucloud_keypair.example 1602/12345
```

OpenTofu: use `tofu import` with the same ID format.
